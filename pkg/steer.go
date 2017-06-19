package steer

import (
	"fmt"
	"io/ioutil"

	"github.com/ghodss/yaml"
)

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

	return plan.run()
}
