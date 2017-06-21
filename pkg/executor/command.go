package executor

import (
	"fmt"
)

type Action func() error

type Command interface {
	// TODO String() string
	Run() error
	Undo() error
}

type BaseCommand struct {
	RunAction  Action
	UndoAction Action
}

var undoStack []Command

func init() {
	undoStack = []Command{}
}

func (c *BaseCommand) Run() error {
	err := c.RunAction()
	if err == nil {
		undoStack = append([]Command{c}, undoStack...)
	}
	return err
}

func (c *BaseCommand) Undo() error {
	return c.UndoAction()
}

func UndoCommands() {
	for _, cmd := range undoStack {
		err := cmd.Undo()
		if err != nil {
			fmt.Println("Error: Undo command failed")
		}
	}
}
