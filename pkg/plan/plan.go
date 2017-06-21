package plan

import (
	"errors"
	"fmt"
	"io/ioutil"

	"github.com/Masterminds/semver"
	"github.com/deckarep/golang-set"
	"github.com/ghodss/yaml"
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

type Command struct {
	Run  []string
	Undo []string
}

type Action int

const (
	actionInstall Action = iota
	actionUpgrade
	actionDelete
)

func (p *Plan) Process(namespaces []string) ([]Command, error) {

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

	insertNodes := func(g dependencyGraph, s mapset.Set, a Action) dependencyGraph {
		for r := range s.Iter() {
			name := r.(string)
			g = append(g, &dependencyNode{stack: stackMap[name], action: a})
		}
		return g
	}

	graph := dependencyGraph{}
	graph = insertNodes(graph, delete, actionDelete)
	graph = insertNodes(graph, install, actionInstall)
	graph = insertNodes(graph, upgrade, actionUpgrade)

	graph, err = resolveDependencies(graph)
	if err != nil {
		return nil, err
	}

	commands := map[Action]func(Stack) Command{
		actionInstall: func(s Stack) Command {
			return Command{
				Run:  s.Spec.installCmd(),
				Undo: []string{},
			}
		},
		actionDelete: func(s Stack) Command {
			return Command{
				Run:  []string{},
				Undo: []string{},
			}
		},
		actionUpgrade: func(s Stack) Command {
			return Command{
				Run:  s.Spec.upgradeCmd(),
				Undo: []string{},
			}
		},
	}

	cmds := []Command{}
	for _, r := range graph {
		cmds = append(cmds, commands[r.action](r.stack))
	}

	return cmds, nil
}

func extractUpgrades(known mapset.Set, releasesMap map[string]*release.Release, stackMap map[string]Stack) (mapset.Set, error) {

	upgrade := mapset.NewSet()

	for r := range known.Iter() {

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

// --- Dependency resolution --------------------------------------------------

type dependencyNode struct {
	stack  Stack
	action Action
}
type dependencyGraph []*dependencyNode

// Returns the name of the release targeted by the command, namespaced
func (n *dependencyNode) namespacedName() string {
	return n.stack.Spec.Namespace + "." + n.stack.Spec.Name
}

// Returns the dependencies of the release targeted by the command, namespaced
func (n *dependencyNode) namespacedDeps() []string {
	deps := []string{}
	for _, dep := range n.stack.Deps {
		deps = append(deps, n.stack.Spec.Namespace+"."+dep)
	}
	return deps
}

// resolveDependencies uses topological sort to resolve the command dependencies
// http://dnaeon.github.io/dependency-graph-resolution-algorithm-in-go/
func resolveDependencies(graph dependencyGraph) (dependencyGraph, error) {

	// A map that contains the name to the actual object
	nodeNames := make(map[string]*dependencyNode)

	// A map that contains the node and their dependencies
	nodeDependencies := make(map[string]mapset.Set)

	// Populate the maps
	for _, node := range graph {
		name := node.namespacedName()
		nodeNames[name] = node

		dependencySet := mapset.NewSet()
		for _, dep := range node.namespacedDeps() {
			dependencySet.Add(dep)
		}
		nodeDependencies[name] = dependencySet
	}

	// Iteratively find and remove nodes from the graph which have no dependencies.
	// If at some point there are still nodes in the graph and we cannot find
	// nodes without dependencies, that means we have a circular dependency
	var resolved []*dependencyNode
	for len(nodeDependencies) != 0 {
		// Get all the nodes from the graph which have no dependecies
		readySet := mapset.NewSet()
		for name, deps := range nodeDependencies {
			if deps.Cardinality() == 0 {
				readySet.Add(name)
			}
		}

		// If there aren't any ready nodes, then we have a circular dependency
		if readySet.Cardinality() == 0 {
			var g []*dependencyNode
			for name := range nodeDependencies {
				g = append(g, nodeNames[name])
			}
			return g, errors.New("Circular dependency found")
		}

		// Remove the ready nodes and add them to the resolved graph
		for name := range readySet.Iter() {
			delete(nodeDependencies, name.(string))
			resolved = append(resolved, nodeNames[name.(string)])
		}

		// Also make sure to remove the ready nodes from the remaining node
		// dependencies as well
		for name, deps := range nodeDependencies {
			diff := deps.Difference(readySet)
			nodeDependencies[name] = diff
		}
	}

	return resolved, nil
}
