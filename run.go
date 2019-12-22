package demo

import (
	"bufio"
	"fmt"
	"math/rand"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/gookit/color"
	"github.com/pkg/errors"
	"github.com/urfave/cli/v2"
)

const bash = "bash"

type Run struct {
	description []string
	steps       []step
}

type step struct {
	text, command []string
}

func NewRun(description ...string) *Run {
	return &Run{description, nil}
}

func S(s ...string) []string {
	return s
}

func (r *Run) Step(text []string, command []string) {
	r.steps = append(r.steps, step{text, command})
}

func (r *Run) Run(ctx *cli.Context) error {
	r.printTitleAndDescription()
	for i, step := range r.steps {
		if ctx.Int("skip-steps") > i {
			continue
		}
		if err := step.run(ctx, i+1, len(r.steps)); err != nil {
			return err
		}
	}
	return nil
}

func (r *Run) printTitleAndDescription() {
	for i, d := range r.description {
		if i == 0 {
			color.Cyan.Println(d)
			for range d {
				color.Cyan.Print("=")
			}
			fmt.Printf("\n")
		} else {
			color.White.Darken().Println(d)
		}
	}
}

func Ensure(commands ...string) {
	for _, c := range commands {
		cmd := exec.Command(bash, "-c", c)
		cmd.Stderr = nil
		cmd.Stdout = nil
		_ = cmd.Run()
	}
}

func (s *step) run(ctx *cli.Context, current, max int) error {
	if err := waitOrSleep(ctx); err != nil {
		return errors.Wrapf(err, "unable to run step: %v", s)
	}
	if len(s.text) > 0 {
		s.echo(ctx, current, max)
	}
	if len(s.command) > 0 {
		return s.execute(ctx)
	}
	return nil
}

func (s *step) echo(ctx *cli.Context, current, max int) {
	prepared := []string{" "}
	for i, x := range s.text {
		if i == len(s.text)-1 {
			prepared = append(
				prepared,
				color.White.Darken().Sprintf(
					"# %s [%d/%d]:\n",
					x, current, max,
				),
			)
		} else {
			m := color.White.Darken().Sprintf("# %s", x)
			prepared = append(prepared, m)
		}
	}
	print(ctx, prepared...)
}

func (s *step) execute(ctx *cli.Context) error {
	joinedCommand := strings.Join(s.command, " ")
	cmd := exec.Command(bash, "-c", joinedCommand)

	cmd.Stderr = os.Stderr
	cmd.Stdout = os.Stdout

	cmdString := color.Green.Sprintf("> %s", strings.Join(s.command, " \\\n    "))
	print(ctx, cmdString)
	if err := waitOrSleep(ctx); err != nil {
		return errors.Wrapf(err, "unable to execute step: %v", s)
	}
	return errors.Wrap(cmd.Run(), "step command failed")
}

func print(ctx *cli.Context, msg ...string) {
	for _, m := range msg {
		for _, c := range m {
			if !ctx.Bool("immediate") {
				time.Sleep(time.Duration(rand.Intn(40)) * time.Millisecond)
			}
			fmt.Printf("%c", c)
		}
		println()
	}
}

func waitOrSleep(ctx *cli.Context) error {
	if ctx.Bool("auto") {
		time.Sleep(ctx.Duration("auto-timeout"))
	} else {
		fmt.Print("…")
		_, err := bufio.NewReader(os.Stdin).ReadBytes('\n')
		if err != nil {
			return errors.Wrap(err, "unable to read newline")
		}
		fmt.Printf("\x1b[1A") // Move cursor up again
	}
	return nil
}
