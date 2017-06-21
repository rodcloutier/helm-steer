package plan

import (
	"errors"
	"fmt"
	"io/ioutil"

	"github.com/Masterminds/semver"
	"github.com/deckarep/golang-set"
	"github.com/ghodss/yaml"
	"github.com/rodcloutier/helm-steer/pkg/executor"
	"github.com/rodcloutier/helm-steer/pkg/helm"
	"k8s.io/helm/pkg/proto/hapi/release"
)

type Stack struct {
	Spec ChartSpec `json:"spec"`
	Deps []string  `json:"depends"`
}

type Namespace map[string]Stack

type Plan struct {
	Namespaces map[string]Namespace `json:"namespaces"`
	Version    string               `json:"version"`
}

type StackCommand struct {
	stack Stack
	executor.BaseCommand
}
type StackCommandFactory func(Stack) *StackCommand

func newStackCommand(s Stack, run, undo executor.Action) *StackCommand {
	return &StackCommand{
		stack: s,
		BaseCommand: executor.BaseCommand{
			RunAction:  run,
			UndoAction: undo,
		},
	}
}

func (p *Plan) Process(namespaces []string) ([]*StackCommand, error) {

	// TODO do this per namespace ?
	var isValidNamespace func(string) bool

	if len(namespaces) == 0 {
		isValidNamespace = func(string) bool {
			return true
		}
	} else {
		namespacesMap := map[string]bool{}
		for _, ns := range namespaces {
			namespacesMap[ns] = true
		}
		isValidNamespace = func(ns string) bool {
			_, ok := namespacesMap[ns]
			return ok
		}
	}

	// List the currently installed chart deployments
	// helm list
	currentReleases, err := helm.List()
	if err != nil {
		return nil, err
	}

	releases := mapset.NewSet()
	releasesMap := make(map[string]*release.Release)

	specifiedReleases := mapset.NewSet()
	stackMap := make(map[string]Stack)

	for namespaceName, ns := range p.Namespaces {
		if !isValidNamespace(namespaceName) {
			continue
		}
		for stackName, _ := range ns {
			key := namespaceName + "." + stackName
			specifiedReleases.Add(key)
			stackMap[key] = ns[stackName]
		}
	}

	if specifiedReleases.Cardinality() == 0 {
		fmt.Println("Nothing to do, no stack found")
		return nil, nil
	}

	for _, r := range currentReleases {
		if !isValidNamespace(r.Namespace) {
			continue
		}
		_, ok := p.Namespaces[r.Namespace]
		if !ok {
			continue
		}
		key := r.Namespace + "." + r.Name
		releases.Add(key)
		releasesMap[key] = r
	}

	install := specifiedReleases.Difference(releases)
	delete := releases.Difference(specifiedReleases)

	known := specifiedReleases.Intersect(releases)
	upgrade, err := extractUpgrades(known, releasesMap, stackMap)
	if err != nil {
		return nil, err
	}

	createCommands := func(s mapset.Set, f StackCommandFactory) []*StackCommand {
		cmds := []*StackCommand{}
		for r := range s.Iter() {
			release := r.(string)
			cmds = append(cmds, f(stackMap[release]))
		}
		return cmds
	}

	noop := func() error { return nil }

	newDeleteCommand := func(s Stack) *StackCommand {
		return newStackCommand(s, noop, noop)
	}
	newInstallCommand := func(s Stack) *StackCommand {
		return newStackCommand(s, s.Spec.install, noop)
	}
	newUpgradeCommand := func(s Stack) *StackCommand {
		return newStackCommand(s, s.Spec.upgrade, noop)
	}

	cmds := createCommands(delete, newDeleteCommand)
	cmds = append(cmds, createCommands(install, newInstallCommand)...)
	cmds = append(cmds, createCommands(upgrade, newUpgradeCommand)...)

	fmt.Println("Resolving dependencies")
	cmds, err = resolveDependencies(cmds)

	return cmds, err
}

func extractUpgrades(known mapset.Set, releasesMap map[string]*release.Release, stackMap map[string]Stack) (mapset.Set, error) {

	upgrade := mapset.NewSet()

	for r := range known.Iter() {

		// TODO check for semver parsable version

		release := r.(string)
		deployedVersion := releasesMap[release].Chart.Metadata.Version
		specifiedVersion := stackMap[release].Spec.Version

		deployedSemver, err := semver.NewVersion(deployedVersion)
		if err != nil {
			return nil, err
		}

		equalConstraint, err := semver.NewConstraint("= " + specifiedVersion)
		if err != nil {
			return nil, err
		}

		// If version deployed == specified
		if equalConstraint.Check(deployedSemver) {
			// nothing to do, but status not yet known
			continue
		}

		// If version deployed != specified
		upgrade.Add(release)
	}
	return upgrade, nil
}

// Conform will apply the name and namespaces to the contained Stack
func (p *Plan) Conform() {
	for namespaceName, ns := range p.Namespaces {
		for stackName, _ := range ns {
			stack := ns[stackName]
			stack.Conform(namespaceName, stackName)
			ns[stackName] = stack
		}
	}
}

func (s *Stack) Conform(namespaceName, stackName string) {
	// TODO should we validate that the names correspond to the expected value?
	s.Spec.Name = stackName
	s.Spec.Namespace = namespaceName
}

// Load will load a plan file and return the plan
func Load(planPath string) (*Plan, error) {
	// Read the plan.yaml file specified
	content, err := ioutil.ReadFile(planPath)
	if err != nil {
		return nil, err
	}

	var plan Plan
	err = yaml.Unmarshal(content, &plan)
	if err != nil {
		fmt.Println("err:%v\n", err)
		return nil, err
	}

	plan.Conform()

	return &plan, nil
}

// Returns the name of the release targeted by the command, namespaced
func (s *StackCommand) namespacedName() string {
	return s.stack.Spec.Namespace + "." + s.stack.Spec.Name
}

// Returns the dependencies of the release targeted by the command, namespaced
func (s *StackCommand) namespacedDeps() []string {
	deps := []string{}
	for _, dep := range s.stack.Deps {
		deps = append(deps, s.stack.Spec.Namespace+"."+dep)
	}
	return deps
}

// resolveDependencies uses topological sort to resolve the command dependencies
// http://dnaeon.github.io/dependency-graph-resolution-algorithm-in-go/
func resolveDependencies(cmds []*StackCommand) ([]*StackCommand, error) {

	// A map that contains the name to the actual object
	cmdNames := make(map[string]*StackCommand)

	// A map that contains the commands and their dependencies
	cmdDependencies := make(map[string]mapset.Set)

	// Populate the maps
	for _, cmd := range cmds {
		name := cmd.namespacedName()
		cmdNames[name] = cmd

		dependencySet := mapset.NewSet()
		for _, dep := range cmd.namespacedDeps() {
			dependencySet.Add(dep)
		}
		cmdDependencies[name] = dependencySet
	}

	// Iteratively find and remove nodes from the graph which have no dependencies.
	// If at some point there are still nodes in the graph and we cannot find
	// nodes without dependencies, that means we have a circular dependency
	var resolved []*StackCommand
	for len(cmdDependencies) != 0 {
		// Get all the nodes from the graph which have no dependecies
		readySet := mapset.NewSet()
		for name, deps := range cmdDependencies {
			if deps.Cardinality() == 0 {
				readySet.Add(name)
			}
		}

		// If there aren't any ready nodes, then we have a circular dependency
		if readySet.Cardinality() == 0 {
			var g []*StackCommand
			for name := range cmdDependencies {
				g = append(g, cmdNames[name])
			}
			return g, errors.New("Circular dependency found")
		}

		// Remove the ready nodes and add them to the resolved graph
		for name := range readySet.Iter() {
			delete(cmdDependencies, name.(string))
			resolved = append(resolved, cmdNames[name.(string)])
		}

		// Also make sure to remove the ready nodes from the remaining node
		// dependencies as well
		for name, deps := range cmdDependencies {
			diff := deps.Difference(readySet)
			cmdDependencies[name] = diff
		}
	}

	return resolved, nil
}
