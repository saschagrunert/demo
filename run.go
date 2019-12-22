package demo

import (
	"bufio"
	"fmt"
	"io"
	"math/rand"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/gookit/color"
	"github.com/pkg/errors"
	"github.com/urfave/cli/v2"
)

// Every command will be executed in a new bash subshell. This is the only
// external dependency we need
const bash = "bash"

// Run is an abstraction for one part of the Demo. A demo can contain multiple
// runs.
type Run struct {
	title       string
	description []string
	steps       []step
	out         io.Writer
	options     *Options
}

type step struct {
	r             *Run
	text, command []string
}

// Options specify the run options
type Options struct {
	AutoTimeout time.Duration
	Auto        bool
	Immediate   bool
	SkipSteps   int
}

// NewRun creates a new run for the provided description string
func NewRun(title string, description ...string) *Run {
	return &Run{
		title:       title,
		description: description,
		steps:       nil,
		out:         os.Stdout,
		options:     nil,
	}
}

// optionsFrom creates a new set of options from the provided context
func optionsFrom(ctx *cli.Context) Options {
	return Options{
		AutoTimeout: ctx.Duration("auto-timeout"),
		Auto:        ctx.Bool("auto"),
		Immediate:   ctx.Bool("immediate"),
		SkipSteps:   ctx.Int("skip-steps"),
	}
}

// S is a short-hand for converting string slice syntaxes
func S(s ...string) []string {
	return s
}

// SetOutput can be used to replace the default output for the Run
func (r *Run) SetOutput(output io.Writer) error {
	if output == nil {
		return errors.New("provided output is nil")
	}
	r.out = output
	return nil
}

// Step creates a new step on the provided run
func (r *Run) Step(text, command []string) {
	r.steps = append(r.steps, step{r, text, command})
}

// Run executes the run in the provided CLI context
func (r *Run) Run(ctx *cli.Context) error {
	return r.RunWithOptions(optionsFrom(ctx))
}

// RunWithOptions executes the run with the provided Options
func (r *Run) RunWithOptions(opts Options) error {
	r.options = &opts

	if err := r.printTitleAndDescription(); err != nil {
		return err
	}
	for i, step := range r.steps {
		if r.options.SkipSteps > i {
			continue
		}
		if err := step.run(i+1, len(r.steps)); err != nil {
			return err
		}
	}
	return nil
}

func (r *Run) printTitleAndDescription() error {
	if err := write(r.out, color.Cyan.Sprintf("%s\n", r.title)); err != nil {
		return err
	}
	for range r.title {
		if err := write(r.out, color.Cyan.Sprint("=")); err != nil {
			return err
		}
	}
	if err := write(r.out, "\n"); err != nil {
		return err
	}
	for _, d := range r.description {
		if err := write(
			r.out, color.White.Darken().Sprintf("%s\n", d),
		); err != nil {
			return err
		}
	}
	return nil
}

func write(w io.Writer, str string) error {
	_, err := w.Write([]byte(str))
	return err
}

// Ensure executes the provided commands in order
func Ensure(commands ...string) error {
	for _, c := range commands {
		cmd := exec.Command(bash, "-c", c)
		cmd.Stderr = nil
		cmd.Stdout = nil
		if err := cmd.Run(); err != nil {
			return err
		}
	}
	return nil
}

func (s *step) run(current, max int) error {
	if err := s.waitOrSleep(); err != nil {
		return errors.Wrapf(err, "unable to run step: %v", s)
	}
	if len(s.text) > 0 {
		s.echo(current, max)
	}
	if len(s.command) > 0 {
		return s.execute()
	}
	return nil
}

func (s *step) echo(current, max int) {
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
	s.print(prepared...)
}

func (s *step) execute() error {
	joinedCommand := strings.Join(s.command, " ")
	cmd := exec.Command(bash, "-c", joinedCommand)

	cmd.Stderr = s.r.out
	cmd.Stdout = s.r.out

	cmdString := color.Green.Sprintf("> %s", strings.Join(s.command, " \\\n    "))
	s.print(cmdString)
	if err := s.waitOrSleep(); err != nil {
		return errors.Wrapf(err, "unable to execute step: %v", s)
	}
	return errors.Wrap(cmd.Run(), "step command failed")
}

func (s *step) print(msg ...string) error {
	for _, m := range msg {
		for _, c := range m {
			if !s.r.options.Immediate {
				time.Sleep(time.Duration(rand.Intn(40)) * time.Millisecond)
			}
			if err := write(s.r.out, fmt.Sprintf("%c", c)); err != nil {
				return err
			}
		}
		if err := write(s.r.out, "\n"); err != nil {
			return err
		}
	}
	return nil
}

func (s *step) waitOrSleep() error {
	if s.r.options.Auto {
		time.Sleep(s.r.options.AutoTimeout)
	} else {
		if err := write(s.r.out, "…"); err != nil {
			return err
		}
		_, err := bufio.NewReader(os.Stdin).ReadBytes('\n')
		if err != nil {
			return errors.Wrap(err, "unable to read newline")
		}
		// Move cursor up again
		if err := write(s.r.out, "\x1b[1A"); err != nil {
			return err
		}
	}
	return nil
}
