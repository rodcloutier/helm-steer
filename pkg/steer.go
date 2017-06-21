package steer

import (
	"fmt"

	"github.com/rodcloutier/helm-steer/pkg/executor"
	"github.com/rodcloutier/helm-steer/pkg/plan"
)

func Steer(planPath string, namespaces []string, dr bool) error {

	executor.DryRun = dr

	pl, err := plan.Load(planPath)
	if err != nil {
		return err
	}

	cmdArgs, err := pl.Process(namespaces)
	if err != nil {
		return err
	}

	// Build the actual commands
	ranCommands := []executor.Command{}
	for _, cmdArgs := range cmdArgs {
		cmd := executor.NewExecutableCommand("helm", cmdArgs.Run)
		err = cmd.Run()
		if err != nil {
			fmt.Println("Error: Last command failed. Undoing previous commands")
			// executor.UndoCommands()
			undoCommands(ranCommands)
			return err
		}
		if len(cmdArgs.Undo) > 0 {
			undoCmd := executor.NewExecutableCommand("helm", cmdArgs.Undo)
			ranCommands = append([]executor.Command{undoCmd}, ranCommands...)
		}
	}
	return nil
}

func undoCommands(cmds []executor.Command) {
	for _, cmd := range cmds {
		err := cmd.Run()
		if err != nil {
			fmt.Println("Failed to perform undo command %s", cmd)
		}
	}
}
