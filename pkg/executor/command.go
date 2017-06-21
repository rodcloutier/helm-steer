package executor

import (
	"fmt"
	"io"
	"os/exec"
	"strings"
)

type Command interface {
	String() string
	Run(w io.Writer) error
}

type executableCommand struct {
	entrypoint string
	args       []string
}

func NewExecutableCommand(e string, args []string) Command {
	return &executableCommand{
		entrypoint: e,
		args:       args,
	}
}

func (c executableCommand) String() string {
	items := []string{c.entrypoint}
	items = append(items, c.args...)
	return fmt.Sprintf(strings.Join(items, " "))
}

func (c executableCommand) Run(w io.Writer) error {
	cmd := exec.Command(c.entrypoint, c.args...)
	cmd.Stdout = w
	cmd.Stderr = w
	err := cmd.Start()
	if err != nil {
		return err
	}
	err = cmd.Wait()
	return err
}
