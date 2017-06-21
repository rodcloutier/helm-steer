package steer

import (
	"fmt"

	"github.com/rodcloutier/helm-steer/pkg/executor"
	"github.com/rodcloutier/helm-steer/pkg/plan"
)

var dryRun bool

func Steer(planPath string, namespaces []string, dr bool) error {

	dryRun = dr

	pl, err := plan.Load(planPath)
	if err != nil {
		return err
	}

	cmds, err := pl.Process(namespaces)
	if err != nil {
		return err
	}

	for _, cmd := range cmds {

		err = cmd.Run()
		if err != nil {
			fmt.Println("Error: Last command failed. Undoing previous commands")
			executor.UndoCommands()
			return err
		}
	}
	return nil
}
