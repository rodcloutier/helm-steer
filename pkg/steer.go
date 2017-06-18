package steer

import (
	"fmt"
	"io/ioutil"

	"github.com/ghodss/yaml"
)

type Plan struct {
	Charts []ChartSpec `json:"charts"`
}

var skippedInstallFlags = []string{
	"chart",
	"name",
	"name-template",
	"replace",
}

func Steer(planPath string) error {

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

	// For each chart, run the install command
	// if not present
	//      install
	for _, c := range plan.Charts {
		err = c.install()
		if err != nil {
			fmt.Printf("Error: Chart %s (%s) failed to install\n", c.Name, c.Chart)
			return err
		}
	}
	// else
	//      upgrade

	return nil
}
