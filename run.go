package demo

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"math/rand"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/gookit/color"
	"github.com/urfave/cli/v2"
)

// errOutputNil is the error returned if no output has been set.
var errOutputNil = errors.New("provided output is nil")

// Run is an abstraction for one part of the Demo. A demo can contain multiple
// runs.
type Run struct {
	title       string
	description []string
	steps       []step
	out         io.Writer
	options     *Options
	setup       func() error
	cleanup     func() error
}

type step struct {
	r             *Run
	text, command []string
	canFail       bool
}

// Options specify the run options.
type Options struct {
	AutoTimeout      time.Duration
	Auto             bool
	HideDescriptions bool
	Immediate        bool
	SkipSteps        int
	Shell            string
}

func emptyFn() error { return nil }

// NewRun creates a new run for the provided description string.
func NewRun(title string, description ...string) *Run {
	return &Run{
		title:       title,
		description: description,
		steps:       nil,
		out:         os.Stdout,
		options:     nil,
		setup:       emptyFn,
		cleanup:     emptyFn,
	}
}

// optionsFrom creates a new set of options from the provided context.
func optionsFrom(ctx *cli.Context) Options {
	return Options{
		AutoTimeout:      ctx.Duration(FlagAutoTimeout),
		Auto:             ctx.Bool(FlagAuto),
		HideDescriptions: ctx.Bool(FlagHideDescriptions),
		Immediate:        ctx.Bool(FlagImmediate),
		SkipSteps:        ctx.Int(FlagSkipSteps),
		Shell:            ctx.String(FlagShell),
	}
}

// S is a short-hand for converting string slice syntaxes.
func S(s ...string) []string {
	return s
}

// SetOutput can be used to replace the default output for the Run.
func (r *Run) SetOutput(output io.Writer) error {
	if output == nil {
		return errOutputNil
	}
	r.out = output

	return nil
}

// Setup sets the cleanup function called before this run.
func (r *Run) Setup(setupFn func() error) {
	r.setup = setupFn
}

// Cleanup sets the cleanup function called after this run.
func (r *Run) Cleanup(cleanupFn func() error) {
	r.cleanup = cleanupFn
}

// Step creates a new step on the provided run.
func (r *Run) Step(text, command []string) {
	r.steps = append(r.steps, step{r, text, command, false})
}

// StepCanFail creates a new step which can fail on execution.
func (r *Run) StepCanFail(text, command []string) {
	r.steps = append(r.steps, step{r, text, command, true})
}

// Run executes the run in the provided CLI context.
func (r *Run) Run(ctx *cli.Context) error {
	return r.RunWithOptions(optionsFrom(ctx))
}

// RunWithOptions executes the run with the provided Options.
func (r *Run) RunWithOptions(opts Options) error {
	if opts.Shell == "" {
		opts.Shell = "bash"
	}

	if err := r.setup(); err != nil {
		return err
	}

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

	return r.cleanup()
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
	if !r.options.HideDescriptions {
		for _, d := range r.description {
			if err := write(
				r.out, color.White.Darken().Sprintf("%s\n", d),
			); err != nil {
				return err
			}
		}
		if err := write(r.out, "\n"); err != nil {
			return err
		}
	}

	return nil
}

func write(w io.Writer, str string) error {
	_, err := w.Write([]byte(str))

	return err
}

func (s *step) run(current, max int) error {
	if err := s.waitOrSleep(); err != nil {
		return fmt.Errorf("unable to run step: %v: %w", s, err)
	}
	if len(s.text) > 0 && !s.r.options.HideDescriptions {
		s.echo(current, max)
	}
	if len(s.command) > 0 {
		return s.execute()
	}

	return nil
}

func (s *step) echo(current, max int) {
	prepared := []string{}
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
	cmd := exec.Command(s.r.options.Shell, "-c", joinedCommand) //nolint:gosec // we purposefully run user-provided code

	cmd.Stderr = s.r.out
	cmd.Stdout = s.r.out

	cmdString := color.Green.Sprintf("> %s", strings.Join(s.command, " \\\n    "))
	s.print(cmdString)
	if err := s.waitOrSleep(); err != nil {
		return fmt.Errorf("unable to execute step: %v: %w", s, err)
	}
	err := cmd.Run()
	if s.canFail {
		return nil
	}
	s.print("")

	if err != nil {
		return fmt.Errorf("step command failed: %w", err)
	}

	return nil
}

//nolint:forbidigo // print is intended here
func (s *step) print(msg ...string) error {
	for _, m := range msg {
		for _, c := range m {
			if !s.r.options.Immediate {
				//nolint:gosec // the sleep has no security implications
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
			return fmt.Errorf("unable to read newline: %w", err)
		}
		// Move cursor up again
		if err := write(s.r.out, "\x1b[1A"); err != nil {
			return err
		}
	}

	return nil
}
