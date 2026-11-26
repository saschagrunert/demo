// Package main is a demo used by the tests to verify signal handling.
package main

import (
	"context"
	"errors"
	"fmt"
	"os"

	"github.com/saschagrunert/demo"
	"github.com/urfave/cli/v3"
)

func main() {
	run := demo.NewRun("Interrupt")
	run.Step(nil, demo.S("sleep 10"))
	run.Cleanup(func() error {
		fmt.Println("run cleanup")

		if os.Getenv("FAIL_RUN_CLEANUP") != "" {
			return errors.New("run cleanup failed")
		}

		return nil
	})

	d := demo.New()
	d.Add(run, "run", "run that gets interrupted")
	d.Cleanup(func(ctx context.Context, _ *cli.Command) error {
		fmt.Println("demo cleanup")

		return ctx.Err()
	})
	d.Run()
}
