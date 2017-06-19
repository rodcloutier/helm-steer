package steer

import (
	"fmt"
	"io/ioutil"

	"github.com/ghodss/yaml"
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

var dryRun bool

func Steer(planPath string, dr bool) error {

	dryRun = dr

	// TODO make sure helm is in the path
	// TODO make sure helm is initialized

	// Read the plan.yaml file specified
	content, err := ioutil.ReadFile(planPath)
	if err != nil {
		return err
	}

	var plan Plan
	err = yaml.Unmarshal(content, &plan)
	if err != nil {
		fmt.Println("err:%v\n", err)
		return err
	}

	for name, namespace := range plan.Namespaces {

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
			err = stack.Spec.install()
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
