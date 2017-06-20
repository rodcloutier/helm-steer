package steer

import (
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

func (p *Plan) process() ([]Command, error) {

	// TODO do this per namespace
	// TODO allow to target only a specific namespace

	// List the currently installed chart deployments
	// helm list
	currentReleases, err := helm.List()
	if err != nil {
		return nil, err
	}

	releases := mapset.NewSet()
	releasesMap := make(map[string]*release.Release)

	for _, r := range currentReleases {

		// Check if the plan targets the namespace
		_, ok := p.Namespaces[r.Namespace]
		if !ok {
			continue
		}
		key := r.Namespace + "." + r.Name
		releases.Add(key)
		releasesMap[key] = r
	}

	specifiedReleases := mapset.NewSet()
	stackMap := make(map[string]Stack)

	for name, namespace := range p.Namespaces {
		for stackName, _ := range namespace {
			key := name + "." + stackName
			specifiedReleases.Add(key)
			stackMap[key] = namespace[stackName]
		}
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
