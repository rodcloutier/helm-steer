package steer

import (
	"fmt"
)

var dryRun bool

func Steer(planPath string, namespaces []string, dr bool) error {

	dryRun = dr

	plan, err := Load(planPath)
	if err != nil {
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
