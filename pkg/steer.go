package steer

import (
	"fmt"
	"io"

	"github.com/rodcloutier/helm-steer/pkg/executor"
	"github.com/rodcloutier/helm-steer/pkg/plan"
)

func Steer(outputWriter, debugWriter io.Writer, planPath string, namespaces []string, dryRun bool) error {

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

		fmt.Println(cmdArgs.Description)
		cmd := executor.NewExecutableCommand("helm", cmdArgs.Run)
		fmt.Fprintf(debugWriter, "Executing `%s` ...\n", cmd)
		if dryRun {
			continue
		}
		err = cmd.Run(outputWriter)
		if err != nil {
			fmt.Println("Error: Last command failed. Undoing previous commands")
			// Undo the commands
			for _, undoCmd := range ranCommands {
				fmt.Fprintf(debugWriter, "Executing `%s` ...\n", undoCmd)
				err := undoCmd.Run(outputWriter)
				if err != nil {
					fmt.Println("Failed to perform undo command %s", cmd)
				}
			}
			return err
		}
		if len(cmdArgs.Undo) > 0 {
			undoCmd := executor.NewExecutableCommand("helm", cmdArgs.Undo)
			ranCommands = append([]executor.Command{undoCmd}, ranCommands...)
		}
	}
	return nil
}
