package steer

import (
	"fmt"
	"io/ioutil"

	"github.com/Masterminds/semver"
	"github.com/asaskevich/govalidator"
	"github.com/deckarep/golang-set"
	"github.com/ghodss/yaml"
	"github.com/rodcloutier/helm-steer/pkg/helm"
	"k8s.io/helm/pkg/proto/hapi/release"
)

type Stack struct {
	Spec ChartSpec `valid:"required" json:"spec"`
	Deps []string  `valid:"optional" json:"depends"`
}

type Namespace map[string]Stack

type Plan struct {
	Namespaces map[string]Namespace `valid:"-" json:"namespaces"`
	Version    *string              `valid:"semver,required" json:"version"`
}

func (p *Plan) process(namespaces []string) ([]*Command, error) {

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

	createCommands := func(s mapset.Set, f CommandFactory) []*Command {
		cmds := []*Command{}
		for r := range s.Iter() {
			release := r.(string)
			cmds = append(cmds, f(stackMap[release]))
		}
		return cmds
	}

	cmds := createCommands(delete, NewDeleteCommand)
	cmds = append(cmds, createCommands(install, NewInstallCommand)...)
	cmds = append(cmds, createCommands(upgrade, NewUpgradeCommand)...)

	fmt.Println("Resolving dependencies")
	cmds, err = ResolveDependencies(cmds)

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

func (s *Stack) Conform(namespaceName, stackName string) {
	// TODO should we validate that the names correspond to the expected value?
	s.Spec.Name = stackName
	s.Spec.Namespace = namespaceName
}

// Conform will apply the name and namespaces to the contained Stack
func (p *Plan) Conform() {
	for namespaceName, ns := range p.Namespaces {
		for stackName, stack := range ns {
			stack.Conform(namespaceName, stackName)
			ns[stackName] = stack
		}
	}
}

func (p *Plan) Validate() (bool, error) {

	result, err := govalidator.ValidateStruct(p)
	if err != nil {
		return result, err
	}

	for _, ns := range p.Namespaces {
		for name, stack := range ns {
			result, err = govalidator.ValidateStruct(stack)
			if err != nil {
				return result, fmt.Errorf("'%s' stack field %s", name, err)
			}
		}
	}
	return result, err
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
		fmt.Println("Error: %v\n", err)
		return nil, err
	}

	_, err = plan.Validate()
	if err != nil {
		fmt.Printf("Error: %s\n", err)
		return nil, err
	}

	plan.Conform()

	// _, err = plan.Validate()

	return &plan, err
}
