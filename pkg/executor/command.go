package executor

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
)

type Command interface {
	Run() error
}

type executableCommand struct {
	entrypoint string
	args       []string
}

var DryRun bool

func init() {
	DryRun = false
}

func NewExecutableCommand(e string, args []string) Command {
	return &executableCommand{
		entrypoint: e,
		args:       args,
	}
}

func (c executableCommand) Run() error {

	fmt.Printf("%s %s\n", c.entrypoint, strings.Trim(fmt.Sprint(c.args), "[]"))
	if DryRun {
		return nil
	}

	cmd := exec.Command(c.entrypoint, c.args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	err := cmd.Start()
	if err != nil {
		return err
	}
	err = cmd.Wait()
	return err
}
