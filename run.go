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
	r                     *Run
	text, command         []string
	canFail, isBreakPoint bool
}

// Options specify the run options.
type Options struct {
	AutoTimeout      time.Duration
	Auto             bool
	BreakPoint       bool
	ContinueOnError  bool
	HideDescriptions bool
	DryRun           bool
	NoColor          bool
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
		BreakPoint:       ctx.Bool(FlagBreakPoint),
		ContinueOnError:  ctx.Bool(FlagContinueOnError),
		HideDescriptions: ctx.Bool(FlagHideDescriptions),
		DryRun:           ctx.Bool(FlagDryRun),
		NoColor:          ctx.Bool(FlagNoColor),
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
	r.steps = append(r.steps, step{r, text, command, false, false})
}

// StepCanFail creates a new step which can fail on execution.
func (r *Run) StepCanFail(text, command []string) {
	r.steps = append(r.steps, step{r, text, command, true, false})
}

// BreakPoint creates a new step which can fail on execution.
func (r *Run) BreakPoint() {
	r.steps = append(r.steps, step{r, nil, nil, true, true})
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

		if r.options.ContinueOnError {
			step.canFail = true
		}

		if err := step.run(i+1, len(r.steps)); err != nil {
			return err
		}
	}

	return r.cleanup()
}

func (r *Run) printTitleAndDescription() error {
	p := color.Cyan.Sprintf
	if r.options.NoColor {
		p = fmt.Sprintf
	}

	if err := write(r.out, p("%s\n", r.title)); err != nil {
		return err
	}

	for range r.title {
		if err := write(r.out, p("=")); err != nil {
			return err
		}
	}

	if err := write(r.out, "\n"); err != nil {
		return err
	}

	if !r.options.HideDescriptions {
		p = color.White.Darken().Sprintf
		if r.options.NoColor {
			p = fmt.Sprintf
		}

		for _, d := range r.description {
			if err := write(
				r.out, p("%s\n", d),
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
	if err != nil {
		return fmt.Errorf("write: %w", err)
	}

	return nil
}

func (s *step) run(current, maximum int) error {
	if err := s.waitOrSleep(); err != nil {
		return fmt.Errorf("unable to run step: %v: %w", s, err)
	}

	if len(s.text) > 0 && !s.r.options.HideDescriptions {
		s.echo(current, maximum)
	}

	if s.isBreakPoint {
		return s.wait()
	}

	if len(s.command) > 0 {
		return s.execute()
	}

	return nil
}

func (s *step) echo(current, maximum int) {
	p := color.White.Darken().Sprintf
	if s.r.options.NoColor {
		p = fmt.Sprintf
	}

	prepared := []string{}

	for i, x := range s.text {
		if i == len(s.text)-1 {
			colon := ":"
			if s.command == nil {
				// Do not set the expectation that there is more if no command
				// provided.
				colon = ""
			}

			prepared = append(
				prepared,
				p(
					"# %s [%d/%d]%s\n",
					x, current, maximum, colon,
				),
			)
		} else {
			m := p("# %s", x)
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

	p := color.Green.Sprintf
	if s.r.options.NoColor {
		p = fmt.Sprintf
	}

	cmdString := p("> %s", strings.Join(s.command, " \\\n    "))
	s.print(cmdString)

	if err := s.waitOrSleep(); err != nil {
		return fmt.Errorf("unable to execute step: %v: %w", s, err)
	}

	if s.r.options.DryRun {
		return nil
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

func (s *step) print(msg ...string) error {
	for _, m := range msg {
		for _, c := range m {
			if !s.r.options.Immediate {
				const maximum = 40
				//nolint:gosec,gomnd // the sleep has no security implications and is randomly chosen
				time.Sleep(time.Duration(rand.Intn(maximum)) * time.Millisecond)
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

func (s *step) wait() error {
	if !s.r.options.BreakPoint {
		return nil
	}

	if err := write(s.r.out, "bp"); err != nil {
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

	return nil
}
