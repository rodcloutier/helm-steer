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

func (p *Plan) process() error {

	// List the currently installed chart deployments
	// helm list
	releases, err := helm.List()
	if err != nil {
		return err
	}

	deployedReleases := mapset.NewSet()
	deployedReleasesMap := make(map[string]*release.Release)

	for _, r := range releases {
		key := r.Namespace + "." + r.Name
		deployedReleases.Add(key)
		deployedReleasesMap[key] = r
	}

	specifiedReleases := mapset.NewSet()
	specifiedReleasesMap := make(map[string]Stack)

	for name, namespace := range p.Namespaces {
		for stackName, _ := range namespace {
			key := name + "." + stackName
			specifiedReleases.Add(key)
			specifiedReleasesMap[key] = namespace[stackName]
		}
	}

	install := specifiedReleases.Difference(deployedReleases)
	delete := deployedReleases.Difference(specifiedReleases)
	known := specifiedReleases.Intersect(deployedReleases)

	upgrade := mapset.NewSet()
	rollback := mapset.NewSet()

	it := known.Iterator()
	for r := range it.C {

		// TODO check for semver parsable version

		release := r.(string)
		deployedVersion := deployedReleasesMap[release].Chart.Metadata.Version
		specifiedVersion := specifiedReleasesMap[release].Spec.Version

		deployedSemver, err := semver.NewVersion(deployedVersion)
		if err != nil {
			return nil
		}

		equalConstraint, err := semver.NewConstraint("= " + specifiedVersion)
		if err != nil {
			return nil
		}

		// If version deployed == specified
		if equalConstraint.Check(deployedSemver) {
			// nothing to do, but status not yet known
			continue
		}

		// If version deployed > specified
		greater, err := semver.NewConstraint("> " + specifiedVersion)
		if err != nil {
			return nil
		}
		if greater.Check(deployedSemver) {
			rollback.Add(release)
			continue
		}

		// If version deployed < specified
		upgrade.Add(release)
	}

	fmt.Printf("install %+v\n", install)
	fmt.Printf("delete %+v\n", delete)
	fmt.Printf("upgrade %+v\n", upgrade)
	fmt.Printf("rollback %+v\n", rollback)

	return nil
}

func (p *Plan) run() error {

	for name, namespace := range p.Namespaces {

		fmt.Printf("Processing namespace \"%s\"\n", name)

		for stackName, stack := range namespace {

			// Validate spec
			if stack.Spec.Name != stackName {
				if stack.Spec.Name != "" {
					fmt.Println("Warning: Mismatch between stack name (%s) and stack flag --name (%s). Using %s\n", stackName, stack.Spec.Name, stackName)
				}
				stack.Spec.Name = stackName
			}

			if stack.Spec.Namespace != name {
				if stack.Spec.Namespace != "" {
					fmt.Println("Warning: Mismatch between namespace name (%s) and stack flag --namespace (%s). Using %s\n", name, stack.Spec.Namespace, name)
				}
				stack.Spec.Namespace = name
			}
			// For each chart spec, run the install command
			// if not present
			//      install
			err := stack.Spec.install()
			if err != nil {
				fmt.Printf("Error: Stack %s (%s) failed to install\n", stackName, stack.Spec)
				return err
			}
			// else
			//      upgrade
		}
	}
	return nil
}
