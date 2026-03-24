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

	"github.com/fatih/color"
	"github.com/mattn/go-isatty"
	"github.com/urfave/cli/v3"
	"golang.org/x/term"
)

var (
	// errOutputNil is the error returned if no output has been set.
	errOutputNil = errors.New("provided output is nil")

	// errInputNil is the error returned if no input has been set.
	errInputNil = errors.New("provided input is nil")
)

// Run is an abstraction for one part of the Demo. A demo can contain multiple
// runs.
type Run struct {
	title       string
	description []string
	steps       []step
	out         io.Writer
	in          *bufio.Reader
	inFile      *os.File
	options     Options
	setup       func() error
	cleanup     func() error
	dir         string
	env         []string
}

type step struct {
	text, command         []string
	canFail, isBreakPoint bool
	dir                   string
}

// Options specify the run options.
type Options struct {
	// Context is stored to pass through to exec.CommandContext for step execution.
	//nolint:containedctx // required to thread context through to subprocesses
	Context context.Context

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
	cyanSprintf  func(format string, a ...interface{}) string
	whiteSprintf func(format string, a ...interface{}) string
	greenSprintf func(format string, a ...interface{}) string
}

func emptyFn() error { return nil }

// NewRun creates a new run for the provided description string.
func NewRun(title string, description ...string) *Run {
	return &Run{
		title:       title,
		description: description,
		steps:       nil,
		out:         os.Stdout,
		in:          bufio.NewReader(os.Stdin),
		inFile:      os.Stdin,
		options:     Options{},
		setup:       emptyFn,
		cleanup:     emptyFn,
	}
}

// optionsFrom creates a new set of options from the provided command.
func optionsFrom(ctx context.Context, cmd *cli.Command) Options {
	noColor := cmd.Bool(FlagNoColor)
	opts := Options{
		Context:          ctx,
		AutoTimeout:      cmd.Duration(FlagAutoTimeout),
		Auto:             cmd.Bool(FlagAuto),
		BreakPoint:       cmd.Bool(FlagBreakPoint),
		ContinueOnError:  cmd.Bool(FlagContinueOnError),
		HideDescriptions: cmd.Bool(FlagHideDescriptions),
		DryRun:           cmd.Bool(FlagDryRun),
		NoColor:          noColor,
		Immediate:        cmd.Bool(FlagImmediate),
		SkipSteps:        cmd.Int(FlagSkipSteps),
		Shell:            cmd.String(FlagShell),
		TypewriterSpeed:  cmd.Int(FlagTypewriterSpeed),
	}

	initColorFunctions(&opts)

	return opts
}

func initColorFunctions(opts *Options) {
	if opts.NoColor {
		opts.cyanSprintf = fmt.Sprintf
		opts.whiteSprintf = fmt.Sprintf
		opts.greenSprintf = fmt.Sprintf
	} else {
		opts.cyanSprintf = color.CyanString
		opts.whiteSprintf = color.New(color.FgWhite, color.Faint).SprintfFunc()
		opts.greenSprintf = color.GreenString
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

// SetInput can be used to replace the default input (os.Stdin) for the Run.
func (r *Run) SetInput(input io.Reader) error {
	if input == nil {
		return errInputNil
	}

	r.in = bufio.NewReader(input)

	if f, ok := input.(*os.File); ok {
		r.inFile = f
	} else {
		r.inFile = nil
	}

	return nil
}

// Setup sets the setup function called before this run.
func (r *Run) Setup(setupFn func() error) {
	r.setup = setupFn
}

// Cleanup sets the cleanup function called after this run.
func (r *Run) Cleanup(cleanupFn func() error) {
	r.cleanup = cleanupFn
}

// SetWorkDir sets the working directory for all steps in this run.
func (r *Run) SetWorkDir(dir string) {
	r.dir = dir
}

// SetEnv sets environment variables for all steps in this run.
// Each entry should be in the form "KEY=VALUE". These are appended
// to the current process environment.
func (r *Run) SetEnv(env ...string) {
	r.env = append(r.env, env...)
}

// Step creates a new step on the provided run.
func (r *Run) Step(text, command []string) {
	r.steps = append(r.steps, step{text: text, command: command})
}

// StepCanFail creates a new step which can fail on execution.
func (r *Run) StepCanFail(text, command []string) {
	r.steps = append(r.steps, step{text: text, command: command, canFail: true})
}

// Chdir creates a step that changes the working directory for subsequent steps.
func (r *Run) Chdir(dir string) {
	r.steps = append(r.steps, step{dir: dir})
}

// BreakPoint creates a new step which can fail on execution.
func (r *Run) BreakPoint() {
	r.steps = append(r.steps, step{canFail: true, isBreakPoint: true})
}

// Run executes the run in the provided CLI context.
func (r *Run) Run(ctx context.Context, cmd *cli.Command) error {
	opts := optionsFrom(ctx, cmd)

	return r.RunWithOptions(&opts)
}

// RunWithOptions executes the run with the provided Options.
func (r *Run) RunWithOptions(opts *Options) error {
	if opts.Context == nil {
		opts.Context = context.Background()
	}

	if opts.Shell == "" {
		opts.Shell = "bash"
	}

	if opts.TypewriterSpeed == 0 {
		opts.TypewriterSpeed = DefaultTypewriterSpeed
	}

	if opts.cyanSprintf == nil {
		initColorFunctions(opts)
	}

	if err := r.setup(); err != nil {
		return err
	}

	r.options = *opts

	if err := r.printTitleAndDescription(); err != nil {
		return err
	}

	visibleSteps := r.countVisibleSteps()
	current := 0

	for _, s := range r.steps {
		// Always apply Chdir steps, even when skipped
		if s.dir != "" {
			r.dir = s.dir

			if !r.options.HideDescriptions {
				cdStr := r.options.greenSprintf("> cd %s", s.dir)
				if err := write(r.out, cdStr+"\n"); err != nil {
					return err
				}
			}

			continue
		}

		current++

		if r.options.SkipSteps >= current {
			continue
		}

		if r.options.ContinueOnError {
			s.canFail = true
		}

		if err := s.run(r, current, visibleSteps); err != nil {
			return err
		}
	}

	return r.cleanup()
}

func (r *Run) printTitleAndDescription() error {
	if err := write(r.out, r.options.cyanSprintf("%s\n", r.title)); err != nil {
		return err
	}

	for range r.title {
		if err := write(r.out, r.options.cyanSprintf("=")); err != nil {
			return err
		}
	}

	if err := write(r.out, "\n"); err != nil {
		return err
	}

	if !r.options.HideDescriptions {
		for _, d := range r.description {
			if err := write(
				r.out, r.options.whiteSprintf("%s\n", d),
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

func (r *Run) countVisibleSteps() int {
	count := 0

	for _, s := range r.steps {
		if s.dir == "" {
			count++
		}
	}

	return count
}

func (s *step) run(r *Run, current, maximum int) error {
	if err := s.waitOrSleep(r); err != nil {
		return fmt.Errorf("unable to run step: %w", err)
	}

	if len(s.text) > 0 && !r.options.HideDescriptions {
		if err := s.echo(r, current, maximum); err != nil {
			return err
		}
	}

	if s.isBreakPoint {
		return s.wait(r)
	}

	if len(s.command) > 0 {
		return s.execute(r)
	}

	return nil
}

func (s *step) echo(r *Run, current, maximum int) error {
	prepared := make([]string, len(s.text))

	for i, x := range s.text {
		if i == len(s.text)-1 {
			colon := ":"
			if s.command == nil {
				colon = ""
			}

			prepared[i] = r.options.whiteSprintf(
				"# %s [%d/%d]%s\n",
				x, current, maximum, colon,
			)
		} else {
			prepared[i] = r.options.whiteSprintf("# %s", x)
		}
	}

	return s.print(r, prepared...)
}

func (s *step) execute(r *Run) error {
	joinedCommand := strings.Join(s.command, " ")
	//nolint:gosec // we purposefully run user-provided code
	cmd := exec.CommandContext(r.options.Context, r.options.Shell, "-c", joinedCommand)

	cmd.Stderr = r.out
	cmd.Stdout = r.out
	cmd.Stdin = r.inFile
	cmd.Dir = r.dir

	if len(r.env) > 0 {
		cmd.Env = append(os.Environ(), r.env...)
	}

	displayCommand := strings.Join(s.command, " \\\n    ")
	cmdString := r.options.greenSprintf("> %s", displayCommand)

	if err := s.print(r, cmdString); err != nil {
		return err
	}

	if err := s.waitOrSleep(r); err != nil {
		return fmt.Errorf("unable to execute step: %w", err)
	}

	if r.options.DryRun {
		return nil
	}

	err := cmd.Run()

	if s.canFail {
		return nil
	}

	if err := s.print(r, ""); err != nil {
		return err
	}

	if err != nil {
		return fmt.Errorf("step command failed: %w", err)
	}

	return nil
}

func (s *step) print(r *Run, msg ...string) error {
	for _, m := range msg {
		if r.options.Immediate {
			if err := write(r.out, m); err != nil {
				return err
			}
		} else {
			if err := s.typewrite(r, m); err != nil {
				return err
			}
		}

		if err := write(r.out, "\n"); err != nil {
			return err
		}
	}

	return nil
}

func (s *step) typewrite(r *Run, m string) error {
	restore, raw := r.enterRawMode()
	defer restore()

	for _, c := range m {
		//nolint:gosec // random sleep timing for visual effect, not security-sensitive
		time.Sleep(time.Duration(rand.IntN(r.options.TypewriterSpeed)) * time.Millisecond)

		ch := string(c)
		if raw && c == '\n' {
			ch = "\r\n"
		}

		if err := write(r.out, ch); err != nil {
			return err
		}
	}

	return nil
}

func (s *step) waitOrSleep(r *Run) error {
	if r.options.Auto {
		time.Sleep(r.options.AutoTimeout)

		return nil
	}

	restore, raw := r.enterRawMode()

	if err := write(r.out, "\u2026"); err != nil {
		restore()

		return err
	}

	if err := r.readInput(raw); err != nil {
		restore()

		return err
	}

	restore()

	if raw {
		// In raw mode, Enter doesn't produce a visible newline,
		// so just clear the prompt on the current line.
		return write(r.out, "\r\x1b[K")
	}

	return moveCursorUp(r.out)
}

func (s *step) wait(r *Run) error {
	if !r.options.BreakPoint {
		return nil
	}

	restore, raw := r.enterRawMode()

	if err := write(r.out, "bp"); err != nil {
		restore()

		return err
	}

	if err := r.readInput(raw); err != nil {
		restore()

		return err
	}

	restore()

	if raw {
		return write(r.out, "\r\x1b[K")
	}

	return moveCursorUp(r.out)
}

// enterRawMode puts the terminal into raw mode if stdin is a terminal.
// It returns a restore function and whether raw mode was activated.
func (r *Run) enterRawMode() (func(), bool) {
	if r.inFile == nil {
		return func() {}, false
	}

	fd := r.inFile.Fd()
	if !isatty.IsTerminal(fd) && !isatty.IsCygwinTerminal(fd) {
		return func() {}, false
	}

	//nolint:gosec // fd is a valid file descriptor from os.File
	intFd := int(fd)

	oldState, err := term.MakeRaw(intFd)
	if err != nil {
		return func() {}, false
	}

	return func() { _ = term.Restore(intFd, oldState) }, true
}

// readInput reads input, using single-byte read in raw mode or line read otherwise.
func (r *Run) readInput(raw bool) error {
	if raw {
		_, err := r.in.ReadByte()
		if err != nil {
			return fmt.Errorf("unable to read keypress: %w", err)
		}

		return nil
	}

	_, err := r.in.ReadBytes('\n')
	if err != nil {
		return fmt.Errorf("unable to read newline: %w", err)
	}

	return nil
}

func moveCursorUp(w io.Writer) error {
	if !isTerminal(w) {
		return nil
	}

	return write(w, "\x1b[1A")
}

func isTerminal(w io.Writer) bool {
	if f, ok := w.(*os.File); ok {
		return isatty.IsTerminal(f.Fd()) || isatty.IsCygwinTerminal(f.Fd())
	}

	return false
}
