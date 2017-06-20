package steer

import (
	"fmt"

	"github.com/Masterminds/semver"
	"github.com/deckarep/golang-set"
	"github.com/rodcloutier/helm-steer/pkg/helm"
	"k8s.io/helm/pkg/proto/hapi/release"
)

type Stack struct {
	Spec    ChartSpec `json:"spec"`
	Depends []string  `json:"depends"`
}

type Namespace map[string]Stack

type Plan struct {
	Version    string               `json:version`
	Namespaces map[string]Namespace `json:"namespaces"`
}

func (p *Plan) process(namespaces []string) ([]Command, error) {

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

	for name, ns := range p.Namespaces {
		if !isValidNamespace(name) {
			continue
		}
		for stackName, _ := range ns {
			key := name + "." + stackName
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

	createCommands := func(s mapset.Set, f CommandFactory) []Command {
		cmds := []Command{}
		it := s.Iterator()
		for r := range it.C {
			release := r.(string)
			cmds = append(cmds, f(stackMap[release]))
		}
		return cmds
	}

	cmds := createCommands(delete, NewDeleteCommand)
	cmds = append(cmds, createCommands(install, NewInstallCommand)...)
	cmds = append(cmds, createCommands(upgrade, NewUpgradeCommand)...)

	return cmds, nil
}

func extractUpgrades(known mapset.Set, releasesMap map[string]*release.Release, stackMap map[string]Stack) (mapset.Set, error) {

	upgrade := mapset.NewSet()

	it := known.Iterator()
	for r := range it.C {

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
