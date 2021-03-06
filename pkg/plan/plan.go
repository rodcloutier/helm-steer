package plan

import (
	"errors"
	"fmt"
	"io/ioutil"

	"github.com/Masterminds/semver"
	"github.com/deckarep/golang-set"
	"github.com/ghodss/yaml"
	"k8s.io/helm/pkg/proto/hapi/release"

	"github.com/rodcloutier/helm-steer/pkg/helm"
)

type Action int

const (
	actionInstall Action = iota
	actionUpgrade
	actionDelete
)

type Release struct {
	Spec    ReleaseSpec `json:"spec"`
	Depends []string    `json:"depends"`

	action  Action
	release *release.Release
}

type Namespace struct {
	Releases map[string]Release
}

type Plan struct {
	Namespaces map[string]Namespace `json:"namespaces"`
	Version    string               `json:"version"`
}

type Operation struct {
	Description string
	Command     []string
}

type UndoableOperation struct {
	Run  Operation
	Undo Operation
}

// Process will process the plan to extract a dependencies sorted list
// of operations to perform
func (p *Plan) Process(namespaces []string) ([]UndoableOperation, error) {

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
	rawCurrentReleases, err := helm.List()
	if err != nil {
		fmt.Println("Error: Failed to fetch helm list: %s", err)
		return nil, err
	}

	specifiedReleases := mapset.NewSet()
	specifiedReleasesMap := make(map[string]Release)
	for namespaceName, ns := range p.Namespaces {
		if !isValidNamespace(namespaceName) {
			continue
		}
		for releaseName, _ := range ns.Releases {
			key := namespaceName + "." + releaseName
			specifiedReleases.Add(key)
			specifiedReleasesMap[key] = ns.Releases[releaseName]
		}
	}

	if specifiedReleases.Cardinality() == 0 {
		fmt.Println("Nothing to do, no release found")
		return nil, nil
	}

	currentReleases := mapset.NewSet()
	currentReleasesMap := make(map[string]*release.Release)
	for _, r := range rawCurrentReleases {
		if !isValidNamespace(r.Namespace) {
			continue
		}
		_, ok := p.Namespaces[r.Namespace]
		if !ok {
			continue
		}
		key := r.Namespace + "." + r.Name
		currentReleases.Add(key)
		currentReleasesMap[key] = r
	}

	// TODO (rod): delete is a special case where we do not have a Release defined
	// we need to see how to handle this
	// delete := currentReleases.Difference(specifiedReleases)

	/// TODO (rod): Validate that the chart names match the same release name
	// in the same namespace

	install := specifiedReleases.Difference(currentReleases)
	known := specifiedReleases.Intersect(currentReleases)

	specifiedReleasesMap = bindReleases(known, specifiedReleasesMap, currentReleasesMap)

	upgrade := known
	// upgrade, err := extractUpgrades(known, releasesMap, specifiedReleasesMap)
	if err != nil {
		return nil, err
	}

	fmt.Println("Resolving dependencies")

	setAction := func(s mapset.Set, action Action) {
		for r := range s.Iter() {
			name := r.(string)
			release := specifiedReleasesMap[name]
			release.action = action
			specifiedReleasesMap[name] = release
		}
	}
	setAction(install, actionInstall)
	setAction(upgrade, actionUpgrade)

	releases := install.Union(upgrade).ToSlice()
	graph := make(dependencyGraph, len(releases))
	for i, s := range releases {
		graph[i] = specifiedReleasesMap[s.(string)]
	}

	graph, err = resolveDependencies(graph)
	if err != nil {
		fmt.Printf("Error: Failed to resolve dependencies: %s\n", err)
		return nil, err
	}

	fmt.Println("Creating list of operations to perform")
	return createOperations(graph)
}

func bindReleases(releases mapset.Set, specifiedReleasesMap map[string]Release, currentReleasesMap map[string]*release.Release) map[string]Release {

	boundReleaseMap := specifiedReleasesMap

	for n := range releases.Iter() {
		name := n.(string)
		release := specifiedReleasesMap[name]
		release.SetRelease(currentReleasesMap[name])
		boundReleaseMap[name] = release
	}
	return boundReleaseMap
}

// createOperations creates a list of operations based on the specified dependency graph
func createOperations(graph dependencyGraph) ([]UndoableOperation, error) {

	operations := map[Action]func(Release) UndoableOperation{
		actionInstall: func(s Release) UndoableOperation {
			return UndoableOperation{
				Run: Operation{
					Description: fmt.Sprintf("Installing %s", s),
					Command:     s.Spec.installCmd(),
				},
				Undo: Operation{
					Description: fmt.Sprintf("Deleting %s", s),
					Command:     s.Spec.deleteCmd(),
				},
			}
		},
		actionUpgrade: func(s Release) UndoableOperation {
			return UndoableOperation{
				Run: Operation{
					Description: fmt.Sprintf("Upgrading %s", s),
					Command:     s.Spec.upgradeCmd(),
				},
				Undo: Operation{
					Description: fmt.Sprintf("Rollback on %s", s),
					// TODO catch the case were the Version is 1 or release is nil
					Command: func() []string {
						if s.release == nil || s.release.Version <= 1 {
							return s.Spec.deleteCmd()
						}
						return s.Spec.rollbackCmd(s.release.Version - 1)
					}(),
				},
			}
		},
	}

	ops := []UndoableOperation{}
	for _, r := range graph {
		s := r.(Release)
		ops = append(ops, operations[s.action](s))
	}

	return ops, nil
}

func extractUpgrades(known mapset.Set, releasesMap map[string]*release.Release, specifiedReleasesMap map[string]Release) (mapset.Set, error) {

	upgrade := mapset.NewSet()

	for r := range known.Iter() {

		release := r.(string)
		specifiedVersion := specifiedReleasesMap[release].Version()

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

// Conform will apply the name and namespaces to the contained Releases
func (p *Plan) conform() {
	for namespaceName, ns := range p.Namespaces {
		for releaseName, release := range ns.Releases {
			release.conform(namespaceName, releaseName)
			ns.Releases[releaseName] = release
		}
	}
}

func (p Plan) verify() (bool, error) {

	if len(p.Namespaces) == 1 {
		return true, nil
	}

	// We need to make sure that all release names are unique
	names := make([]mapset.Set, len(p.Namespaces))
	i := 0
	for _, ns := range p.Namespaces {
		s := mapset.NewSet()
		for name := range ns.Releases {
			s.Add(name)
		}
		names[i] = s
		i++
	}

	duplicated := names[0]
	for _, n := range names[1:] {
		duplicated = duplicated.Intersect(n)
	}

	if duplicated.Cardinality() > 0 {
		return false, fmt.Errorf("Found duplicated release name %s", duplicated.ToSlice())
	}

	return true, nil
}

// Load will load a plan file and return the plan
func Load(planPath string) (*Plan, error) {
	// Read the plan.yaml file specified
	content, err := ioutil.ReadFile(planPath)
	if err != nil {
		return nil, err
	}
	return loadString(content)
}

func loadString(content []byte) (*Plan, error) {
	var plan Plan
	err := yaml.Unmarshal(content, &plan)
	if err != nil {
		fmt.Println("err:%v\n", err)
		return nil, err
	}

	valid, err := plan.verify()
	if !valid {
		return &plan, err
	}

	plan.conform()

	return &plan, nil
}

// --- Release ------------------------------------------------------------------

func (r *Release) conform(namespace, name string) {
	r.Spec.Conform(namespace, name)
}

// String returns the string representation of a release
func (r Release) String() string {
	return fmt.Sprintf("%s", r.Spec)
}

// Name returns the release name for the release
func (r Release) Name() string {
	return r.Spec.name
}

// Deps returns a list of releases on which the current release depends
func (r Release) Deps() []string {
	return r.Depends
}

// Version returns the version of the release
func (r Release) Version() string {
	return r.Spec.Version()
}

func (r *Release) SetRelease(helmRelease *release.Release) {
	r.release = helmRelease
}

// --- Dependency resolution --------------------------------------------------

type GraphNode interface {
	Name() string
	Deps() []string
}

type dependencyGraph []GraphNode

func (g dependencyGraph) print() {

	for _, n := range g {
		fmt.Printf("%s -> %s\n", n.Name(), n.Deps())
	}
}

// resolveDependencies uses topological sort to resolve the node dependencies
// http://dnaeon.github.io/dependency-graph-resolution-algorithm-in-go/
func resolveDependencies(graph dependencyGraph) (dependencyGraph, error) {

	// A map that contains the name to the actual object
	nodeNames := make(map[string]GraphNode)

	// A map that contains the node and their dependencies
	nodeDependencies := make(map[string]mapset.Set)

	// Populate the maps
	for _, node := range graph {
		nodeNames[node.Name()] = node

		dependencySet := mapset.NewSet()
		for _, dep := range node.Deps() {
			dependencySet.Add(dep)
		}
		nodeDependencies[node.Name()] = dependencySet
	}

	// Iteratively find and remove nodes from the graph which have no dependencies.
	// If at some point there are still nodes in the graph and we cannot find
	// nodes without dependencies, that means we have a circular dependency
	var resolved dependencyGraph
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
			var g []GraphNode
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
