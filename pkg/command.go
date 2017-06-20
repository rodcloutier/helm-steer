package steer

import (
	"fmt"
)

type action func() error

type Command struct {
	action action
	undo   action
	stack  Stack
}

var undoStack []*Command

func init() {
	undoStack = []*Command{}
}

type CommandFactory func(Stack) Command

func NewInstallCommand(s Stack) Command {
	install := func() error {
		return s.Spec.install()
	}
	delete := func() error {
		return nil
	}
	return Command{action: install, undo: delete, stack: s}
}

func NewDeleteCommand(s Stack) Command {
	delete := func() error {
		return nil
	}
	rollback := func() error {
		return nil
	}
	return Command{action: delete, undo: rollback, stack: s}
}

func NewUpgradeCommand(s Stack) Command {
	upgrade := func() error {
		return s.Spec.upgrade()
	}
	rollback := func() error {
		return nil
	}
	return Command{action: upgrade, undo: rollback, stack: s}
}

func (c *Command) Run() error {
	err := c.action()
	if err == nil {
		undoStack = append([]*Command{c}, undoStack...)
	}
	return err
}

func UndoCommands() {
	for _, cmd := range undoStack {
		err := cmd.undo()
		if err != nil {
			fmt.Println("Error: Undo command failed")
		}
	}
}
