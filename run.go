package demo

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"math/rand/v2"
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
	options     Options
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
	//nolint:containedctx // this is intentional
	context context.Context

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
	TypewriterSpeed  int

	// Cached color functions to avoid repeated conditionals
	cyanPrintf  func(format string, a ...interface{}) string
	whitePrintf func(format string, a ...interface{}) string
	greenPrintf func(format string, a ...interface{}) string
}

func emptyFn() error { return nil }

// NewRun creates a new run for the provided description string.
func NewRun(title string, description ...string) *Run {
	return &Run{
		title:       title,
		description: description,
		steps:       nil,
		out:         os.Stdout,
		options:     Options{},
		setup:       emptyFn,
		cleanup:     emptyFn,
	}
}

// optionsFrom creates a new set of options from the provided context.
func optionsFrom(ctx *cli.Context) Options {
	noColor := ctx.Bool(FlagNoColor)
	opts := Options{
		context:          ctx.Context,
		AutoTimeout:      ctx.Duration(FlagAutoTimeout),
		Auto:             ctx.Bool(FlagAuto),
		BreakPoint:       ctx.Bool(FlagBreakPoint),
		ContinueOnError:  ctx.Bool(FlagContinueOnError),
		HideDescriptions: ctx.Bool(FlagHideDescriptions),
		DryRun:           ctx.Bool(FlagDryRun),
		NoColor:          noColor,
		Immediate:        ctx.Bool(FlagImmediate),
		SkipSteps:        ctx.Int(FlagSkipSteps),
		Shell:            ctx.String(FlagShell),
		TypewriterSpeed:  ctx.Int(FlagTypewriterSpeed),
	}

	// Cache color functions based on NoColor setting
	if noColor {
		opts.cyanPrintf = fmt.Sprintf
		opts.whitePrintf = fmt.Sprintf
		opts.greenPrintf = fmt.Sprintf
	} else {
		opts.cyanPrintf = color.Cyan.Sprintf
		opts.whitePrintf = color.White.Darken().Sprintf
		opts.greenPrintf = color.Green.Sprintf
	}

	return opts
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
	opts := optionsFrom(ctx)

	return r.RunWithOptions(&opts)
}

// RunWithOptions executes the run with the provided Options.
func (r *Run) RunWithOptions(opts *Options) error {
	if opts.context == nil {
		opts.context = context.Background()
	}

	if opts.Shell == "" {
		opts.Shell = "bash"
	}

	if opts.TypewriterSpeed == 0 {
		opts.TypewriterSpeed = DefaultTypewriterSpeed
	}

	// Initialize color functions if not set
	if opts.cyanPrintf == nil {
		if opts.NoColor {
			opts.cyanPrintf = fmt.Sprintf
			opts.whitePrintf = fmt.Sprintf
			opts.greenPrintf = fmt.Sprintf
		} else {
			opts.cyanPrintf = color.Cyan.Sprintf
			opts.whitePrintf = color.White.Darken().Sprintf
			opts.greenPrintf = color.Green.Sprintf
		}
	}

	if err := r.setup(); err != nil {
		return err
	}

	r.options = *opts

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

		if err := step.run(opts.context, i+1, len(r.steps)); err != nil {
			return err
		}
	}

	return r.cleanup()
}

func (r *Run) printTitleAndDescription() error {
	if err := write(r.out, r.options.cyanPrintf("%s\n", r.title)); err != nil {
		return err
	}

	for range r.title {
		if err := write(r.out, r.options.cyanPrintf("=")); err != nil {
			return err
		}
	}

	if err := write(r.out, "\n"); err != nil {
		return err
	}

	if !r.options.HideDescriptions {
		for _, d := range r.description {
			if err := write(
				r.out, r.options.whitePrintf("%s\n", d),
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

func (s *step) run(ctx context.Context, current, maximum int) error {
	if err := s.waitOrSleep(); err != nil {
		return fmt.Errorf("unable to run step: %w", err)
	}

	if len(s.text) > 0 && !s.r.options.HideDescriptions {
		s.echo(current, maximum)
	}

	if s.isBreakPoint {
		return s.wait()
	}

	if len(s.command) > 0 {
		return s.execute(ctx)
	}

	return nil
}

func (s *step) echo(current, maximum int) {
	prepared := make([]string, len(s.text))

	for i, x := range s.text {
		if i == len(s.text)-1 {
			colon := ":"
			if s.command == nil {
				// Do not set the expectation that there is more if no command
				// provided.
				colon = ""
			}

			prepared[i] = s.r.options.whitePrintf(
				"# %s [%d/%d]%s\n",
				x, current, maximum, colon,
			)
		} else {
			prepared[i] = s.r.options.whitePrintf("# %s", x)
		}
	}

	s.print(prepared...)
}

func (s *step) execute(ctx context.Context) error {
	joinedCommand := strings.Join(s.command, " ")
	//nolint:gosec // we purposefully run user-provided code
	cmd := exec.CommandContext(ctx, s.r.options.Shell, "-c", joinedCommand)

	cmd.Stderr = s.r.out
	cmd.Stdout = s.r.out

	// Format command display with line continuations
	displayCommand := strings.Join(s.command, " \\\n    ")
	cmdString := s.r.options.greenPrintf("> %s", displayCommand)
	s.print(cmdString)

	if err := s.waitOrSleep(); err != nil {
		return fmt.Errorf("unable to execute step: %w", err)
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
		var buf strings.Builder
		// Pre-allocate for UTF-8: len(m) gives byte count which is the actual size needed
		buf.Grow(len(m))

		for _, c := range m {
			if !s.r.options.Immediate {
				//nolint:gosec // random sleep timing for visual effect, not security-sensitive
				time.Sleep(time.Duration(rand.IntN(s.r.options.TypewriterSpeed)) * time.Millisecond)
			}

			buf.WriteRune(c)
		}

		if err := write(s.r.out, buf.String()); err != nil {
			return err
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
