package steer

import (
	"fmt"
	"io/ioutil"

	"github.com/ghodss/yaml"
)

var dryRun bool

func Steer(planPath string, namespaces []string, dr bool) error {

	dryRun = dr

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

	cmds, err := plan.process(namespaces)
	if err != nil {
		return err
	}

	for _, cmd := range cmds {

		err = cmd.Run()
		if err != nil {
			fmt.Println("Error: Last command failed. Undoing previous commands")
			UndoCommands()
			return err
		}
	}
	return nil
}
