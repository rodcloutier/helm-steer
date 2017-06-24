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

type Operation struct {
	Description string
	Run         []string
	Undo        []string
}

type Action int

const (
	actionInstall Action = iota
	actionUpgrade
	actionDelete
)

func (p *Plan) Process(namespaces []string) ([]Operation, error) {

	// TODO (rod) do this per namespace ?
	isValidNamespace := func(string) bool { return true }
	if len(namespaces) != 0 {
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
	currentReleases, err := helm.List()
	if err != nil {
		fmt.Println("Error: Failed to fetch helm list: %s", err)
		return nil, err
	}

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

	releases := mapset.NewSet()
	releasesMap := make(map[string]*release.Release)
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

	// TODO (rod): delete is a special case where we do not have a Stack defined
	// we need to see how to handle this
	// delete := releases.Difference(specifiedReleases)

	install := specifiedReleases.Difference(releases)
	known := specifiedReleases.Intersect(releases)
	upgrade, err := extractUpgrades(known, releasesMap, stackMap)
	if err != nil {
		return nil, err
	}

	fmt.Println("Resolving dependencies")

	graph := createDependencyGraph(map[Action]mapset.Set{
		actionInstall: install,
		actionUpgrade: upgrade,
	}, stackMap)

	graph, err = resolveDependencies(graph)
	if err != nil {
		fmt.Printf("Error: Failed to resolve dependencies: %s\n", err)
		return nil, err
	}

	fmt.Println("Creating list of operations to perform")
	return createOperations(graph)
}

// createOperations creates a list of operations based on the specified dependency graph
func createOperations(graph dependencyGraph) ([]Operation, error) {

	operations := map[Action]func(Stack) Operation{
		actionInstall: func(s Stack) Operation {
			return Operation{
				Description: fmt.Sprintf("Installing %s", s),
				Run:         s.Spec.installCmd(),
				Undo:        []string{},
			}
		},
		actionUpgrade: func(s Stack) Operation {
			return Operation{
				Description: fmt.Sprintf("Upgrading %s", s),
				Run:         s.Spec.upgradeCmd(),
				Undo:        []string{},
			}
		},
	}

	ops := []Operation{}
	for _, r := range graph {
		ops = append(ops, operations[r.action](r.stack))
	}

	return ops, nil
}

func extractUpgrades(known mapset.Set, releasesMap map[string]*release.Release, stackMap map[string]Stack) (mapset.Set, error) {

	upgrade := mapset.NewSet()

	for r := range known.Iter() {

		release := r.(string)
		specifiedVersion := stackMap[release].Spec.Version

		// No version is specified, we must asssume that we will potentially
		// upgrade.
		// (rod) Maybe we could eventually do a search to find out if
		// there is a potential upgrade
		if specifiedVersion == "" {
			upgrade.Add(release)
			continue
		}

		deployedVersion := releasesMap[release].Chart.Metadata.Version
		deployedSemver, err := semver.NewVersion(deployedVersion)
		if err != nil {
			fmt.Printf("Error: Failed to parse semver `%s`\n", deployedVersion)
			return nil, err
		}

		constraint := "= " + specifiedVersion
		equalConstraint, err := semver.NewConstraint(constraint)
		if err != nil {
			fmt.Printf("Error: Failed to create constraint `%s`\n", constraint)
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
func (p *Plan) conform() {
	for namespaceName, ns := range p.Namespaces {
		for stackName, _ := range ns {
			stack := ns[stackName]
			stack.conform(namespaceName, stackName)
			ns[stackName] = stack
		}
	}
}

func (s *Stack) conform(namespaceName, stackName string) {
	// TODO should we validate that the names correspond to the expected value?
	s.Spec.Name = stackName
	s.Spec.Namespace = namespaceName
}

// String returns the string representation of a stack
func (s Stack) String() string {
	return fmt.Sprintf("%s", s.Spec)
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

	plan.conform()

	return &plan, nil
}

// --- Dependency resolution --------------------------------------------------

type dependencyNode struct {
	stack  Stack
	action Action
}

type dependencyGraph []*dependencyNode

func createDependencyGraph(sets map[Action]mapset.Set, stackMap map[string]Stack) dependencyGraph {

	insertNodes := func(g dependencyGraph, s mapset.Set, a Action) dependencyGraph {
		for r := range s.Iter() {
			name := r.(string)
			g = append(g, &dependencyNode{stack: stackMap[name], action: a})
		}
		return g
	}

	graph := dependencyGraph{}
	for action, set := range sets {
		graph = insertNodes(graph, set, action)
	}

	return graph
}

// Returns the name of the release targeted by the node, namespaced
func (n *dependencyNode) namespacedName() string {
	return n.stack.Spec.Namespace + "." + n.stack.Spec.Name
}

// Returns the dependencies of the release targeted by the node, namespaced
func (n *dependencyNode) namespacedDeps() []string {
	deps := []string{}
	for _, dep := range n.stack.Deps {
		deps = append(deps, n.stack.Spec.Namespace+"."+dep)
	}
	return deps
}

// resolveDependencies uses topological sort to resolve the node dependencies
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
