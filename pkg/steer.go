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

	operations, err := pl.Process(namespaces)
	if err != nil {
		return err
	}

	// Build the actual commands
	operationStack := []plan.UndoableOperation{}
	for _, operation := range operations {
		run := operation.Run
		fmt.Println(run.Description)
		cmd := executor.NewExecutableCommand("helm", run.Command)
		fmt.Fprintf(debugWriter, "Executing `%s` ...\n", cmd)
		if dryRun {
			continue
		}
		err = cmd.Run(outputWriter)
		if err != nil {
			fmt.Println("Error: Last command failed. Undoing previous commands")
			// Undo the commands
			for _, operation := range operationStack {
				undo := operation.Undo
				cmd := executor.NewExecutableCommand("helm", undo.Command)
				fmt.Println(undo.Description)
				fmt.Fprintf(debugWriter, "Executing `%s` ...\n", cmd)
				err := cmd.Run(outputWriter)
				if err != nil {
					fmt.Println("Failed while undoing command")
				}
			}
			return err
		}
		operationStack = append([]plan.UndoableOperation{operation}, operationStack...)
	}
	return nil
}
