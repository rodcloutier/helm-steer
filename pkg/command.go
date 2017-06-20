package steer

import (
	"errors"
	"fmt"

	"github.com/deckarep/golang-set"
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

type CommandFactory func(Stack) *Command

func NewInstallCommand(s Stack) *Command {
	install := func() error {
		return s.Spec.install()
	}
	delete := func() error {
		return nil
	}
	return &Command{action: install, undo: delete, stack: s}
}

func NewDeleteCommand(s Stack) *Command {
	delete := func() error {
		return nil
	}
	rollback := func() error {
		return nil
	}
	return &Command{action: delete, undo: rollback, stack: s}
}

func NewUpgradeCommand(s Stack) *Command {
	upgrade := func() error {
		return s.Spec.upgrade()
	}
	rollback := func() error {
		return nil
	}
	return &Command{action: upgrade, undo: rollback, stack: s}
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

// Returns the name of the release targeted by the command, namespaced
func (c *Command) name() string {
	return c.stack.Spec.Namespace + "." + c.stack.Spec.Name
}

// Returns the dependencies of the release targeted by the command, namespaced
func (c *Command) deps() []string {
	deps := []string{}
	for _, dep := range c.stack.Deps {
		deps = append(deps, c.stack.Spec.Namespace+"."+dep)
	}
	return deps
}

// ResolveDependencies uses topological sort to resolve the command dependencies
// http://dnaeon.github.io/dependency-graph-resolution-algorithm-in-go/
func ResolveDependencies(cmds []*Command) ([]*Command, error) {

	// A map that contains the name to the actual object
	cmdNames := make(map[string]*Command)

	// A map that contains the commands and their dependencies
	cmdDependencies := make(map[string]mapset.Set)

	// Populate the maps
	for _, cmd := range cmds {
		cmdNames[cmd.name()] = cmd

		dependencySet := mapset.NewSet()
		for _, dep := range cmd.deps() {
			dependencySet.Add(dep)
		}
		cmdDependencies[cmd.name()] = dependencySet
	}

	// Iteratively find and remove nodes from the graph which have no dependencies.
	// If at some point there are still nodes in the graph and we cannot find
	// nodes without dependencies, that means we have a circular dependency
	var resolved []*Command
	for len(cmdDependencies) != 0 {
		// Get all the nodes from the graph which have no dependecies
		readySet := mapset.NewSet()
		for name, deps := range cmdDependencies {
			if deps.Cardinality() == 0 {
				readySet.Add(name)
			}
		}

		// If there aren't any ready nodes, then we have a circular dependency
		if readySet.Cardinality() == 0 {
			var g []*Command
			for name := range cmdDependencies {
				g = append(g, cmdNames[name])
			}
			return g, errors.New("Circular dependency found")
		}

		// Remove the ready nodes and add them to the resolved graph
		for name := range readySet.Iter() {
			delete(cmdDependencies, name.(string))
			resolved = append(resolved, cmdNames[name.(string)])
		}

		// Also make sure to remove the ready nodes from the remaining node
		// dependencies as well
		for name, deps := range cmdDependencies {
			diff := deps.Difference(readySet)
			cmdDependencies[name] = diff
		}
	}

	return resolved, nil
}
