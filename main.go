package main

import (
	"os"

	"github.com/urfave/cli"

	"github.com/rodcloutier/helm-steer/pkg"
)

func main() {

	app := cli.NewApp()
	app.Name = "helm-steer"
	app.Usage = "install multiple charts according to a plan"
	app.Action = func(c *cli.Context) error {
		if c.NArg() == 0 {
			return cli.NewExitError("missing expected plan file", 1)
		}
		plan := c.Args()[0]
		return steer.Steer(plan)
	}

	app.Run(os.Args)
}
