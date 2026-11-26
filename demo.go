// Package demo provides a framework for performing live pre-recorded command
// line demos in the terminal.
package demo

import (
	"context"
	"errors"
	"fmt"
	"log"
	"os"
	"os/signal"
	"slices"
	"strings"
	"syscall"
	"time"

	"github.com/saschagrunert/ccli/v3"
	"github.com/urfave/cli/v3"
)

// Demo is the main type for creating command line demos. It wraps a
// cli.Command and manages a set of runs that can be selected via flags.
type Demo struct {
	*cli.Command

	runs     []*runFlag
	setup    func(context.Context, *cli.Command) error
	cleanup  func(context.Context, *cli.Command) error
	palettes struct {
		dark  Palette
		light Palette
	}
}

type runFlag struct {
	run  *Run
	flag cli.Flag
}

const (
	// FlagAll is the flag for running all demos.
	FlagAll = "all"

	// FlagAuto is the flag for running in automatic mode.
	FlagAuto = "auto"

	// FlagAutoTimeout is the flag for the time to wait between steps in
	// automatic mode.
	FlagAutoTimeout = "auto-timeout"

	// FlagBreakPoint is the flag for pausing at breakpoint steps.
	FlagBreakPoint = "with-breakpoints"

	// FlagContinueOnError is the flag for continuing if a step fails.
	FlagContinueOnError = "continue-on-error"

	// FlagContinuously is the flag for running the demos continuously without
	// any end.
	FlagContinuously = "continuously"

	// FlagDryRun is the flag for only printing the commands without executing
	// them.
	FlagDryRun = "dry-run"

	// FlagHideDescriptions is the flag for hiding the descriptions.
	FlagHideDescriptions = "hide-descriptions"

	// FlagImmediate is the flag for disabling the text animations.
	FlagImmediate = "immediate"

	// FlagNoColor is the flag for disabling colored output.
	FlagNoColor = "no-color"

	// FlagTheme allows selecting the output theme.
	FlagTheme = "theme"

	// FlagSkipSteps is the flag for skipping n amount of steps.
	FlagSkipSteps = "skip-steps"

	// FlagShell is the flag for defining the shell that is used to execute the command(s).
	FlagShell = "shell"

	// FlagTypewriterSpeed is the flag for configuring typewriter animation speed (max milliseconds per character).
	FlagTypewriterSpeed = "typewriter-speed"

	// DefaultTypewriterSpeed is the default maximum milliseconds per character for typewriter animation.
	DefaultTypewriterSpeed = 40

	// DefaultShell is the default shell used to execute commands.
	DefaultShell = "bash"

	// ThemeDark is the default theme for terminal backgrounds using dark colors.
	ThemeDark = "dark"

	// ThemeLight is the theme for terminal backgrounds using light colors.
	ThemeLight = "light"

	// ThemeAuto auto-detects the terminal background and picks a palette.
	ThemeAuto = "auto"
)

var errInvalidTheme = errors.New("invalid theme")

func createFlags() []cli.Flag {
	return []cli.Flag{
		&cli.BoolFlag{
			Name:    FlagAll,
			Aliases: []string{"l"},
			Usage:   "run all demos",
		},
		&cli.BoolFlag{
			Name:    FlagAuto,
			Aliases: []string{"a"},
			Usage: "run the demo in automatic mode, " +
				"where every step gets executed automatically",
		},
		&cli.BoolFlag{
			Name:  FlagDryRun,
			Value: false,
			Usage: "only print the commands without executing them",
		},
		&cli.BoolFlag{
			Name:  FlagNoColor,
			Usage: "disable colored output",
		},
		&cli.StringFlag{
			Name:        FlagTheme,
			Value:       ThemeAuto,
			DefaultText: ThemeAuto,
			Usage:       "set output theme palette (auto, dark or light)",
			Validator:   validateTheme,
		},
		&cli.DurationFlag{
			Name:    FlagAutoTimeout,
			Aliases: []string{"t"},
			Usage:   "time to wait between steps in auto mode",
			Value:   1 * time.Second,
		},
		&cli.BoolFlag{
			Name:  FlagBreakPoint,
			Usage: "pause at breakpoint steps",
		},
		&cli.BoolFlag{
			Name:  FlagContinueOnError,
			Usage: "continue if a step fails",
		},
		&cli.BoolFlag{
			Name:    FlagContinuously,
			Aliases: []string{"c"},
			Usage:   "run the demos continuously without any end",
		},
		&cli.BoolFlag{
			Name:    FlagHideDescriptions,
			Aliases: []string{"d"},
			Usage:   "hide descriptions between the steps",
		},
		&cli.BoolFlag{
			Name:    FlagImmediate,
			Aliases: []string{"i"},
			Usage:   "immediately output without the typewriter animation",
		},
		&cli.IntFlag{
			Name:    FlagSkipSteps,
			Aliases: []string{"s"},
			Usage:   "skip the amount of initial steps within the demo",
		},
		&cli.StringFlag{
			Name:        FlagShell,
			Value:       DefaultShell,
			Usage:       "define the shell that is used to execute the command(s)",
			DefaultText: DefaultShell,
		},
		&cli.IntFlag{
			Name:  FlagTypewriterSpeed,
			Usage: "maximum milliseconds per character for typewriter animation",
			Value: DefaultTypewriterSpeed,
		},
	}
}

func collectRunFunctions(
	cmd *cli.Command,
	d *Demo,
	runs []*runFlag,
) []runAction {
	runFns := make([]runAction, 0, len(runs))

	for _, x := range runs {
		if isFlagSet(cmd, x.flag) || cmd.Bool(FlagAll) {
			run := x.run

			runFns = append(runFns, func(ctx context.Context, cmd *cli.Command) error {
				runOpts := d.optionsFrom(ctx, cmd)

				return run.RunWithOptions(&runOpts)
			})
		}
	}

	return runFns
}

type runAction func(context.Context, *cli.Command) error

func isFlagSet(cmd *cli.Command, flag cli.Flag) bool {
	return slices.ContainsFunc(flag.Names(), func(name string) bool {
		return cmd.Bool(name)
	})
}

func createRunSelected(
	demo *Demo, ctx context.Context, cmd *cli.Command, runFns []runAction,
) func() error {
	return func() error {
		for _, runFn := range runFns {
			if err := demo.runWithHooks(ctx, cmd, runFn); err != nil {
				return err
			}
		}

		return nil
	}
}

func runContinuously(ctx context.Context, runSelected func() error) error {
	for {
		if ctx.Err() != nil {
			//nolint:nilerr // context cancellation is the expected exit in continuous mode
			return nil
		}

		if err := runSelected(); err != nil {
			if errors.Is(err, context.Canceled) || errors.Is(err, errInterrupt) {
				return nil
			}

			return err
		}
	}
}

// NewOption configures a demo created through New.
type NewOption func(*Demo)

// WithPalettes overrides both built-in palettes in one call.
func WithPalettes(dark, light Palette) NewOption {
	return func(d *Demo) {
		d.palettes.dark = dark
		d.palettes.light = light
	}
}

// New creates a new Demo instance.
func New(options ...NewOption) *Demo {
	emptyFn := func(context.Context, *cli.Command) error { return nil }
	demo := &Demo{
		Command: ccli.NewCommand(),
		runs:    nil,
		setup:   emptyFn,
		cleanup: emptyFn,
	}
	demo.palettes.dark = DefaultDarkPalette()
	demo.palettes.light = DefaultLightPalette()

	for _, option := range options {
		option(demo)
	}

	demo.Flags = createFlags()
	demo.UseShortOptionHandling = true

	demo.Action = func(ctx context.Context, cmd *cli.Command) error {
		runFns := collectRunFunctions(cmd, demo, demo.runs)
		if len(runFns) == 0 {
			if err := cli.ShowRootCommandHelp(cmd); err != nil {
				return fmt.Errorf("show help: %w", err)
			}

			return nil
		}

		runSelected := createRunSelected(demo, ctx, cmd, runFns)

		if cmd.Bool(FlagContinuously) {
			return runContinuously(ctx, runSelected)
		}

		return runSelected()
	}

	return demo
}

// Setup sets the setup function called before each run.
func (d *Demo) Setup(setupFn func(context.Context, *cli.Command) error) {
	d.setup = setupFn
}

// Cleanup sets the cleanup function called after each run.
func (d *Demo) Cleanup(cleanupFn func(context.Context, *cli.Command) error) {
	d.cleanup = cleanupFn
}

// Add registers a new run with the given flag name and description.
func (d *Demo) Add(run *Run, name, description string) {
	flag := &cli.BoolFlag{
		Name:  name,
		Usage: description,
	}

	d.Flags = append(d.Flags, flag)
	d.runs = append(d.runs, &runFlag{run, flag})
}

const cleanupTimeout = 10 * time.Second

// Run starts the demo and exits the process on error.
func (d *Demo) Run() {
	if err := d.RunE(); err != nil {
		log.Printf("run failed: %v", err)
		os.Exit(1)
	}
}

// RunE starts the demo and returns any error instead of exiting.
//
// SIGINT and SIGTERM cancel the demo and run the cleanup. A second signal
// terminates the process right away.
func (d *Demo) RunE() error {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	go func() {
		<-ctx.Done()
		stop()
	}()

	if runErr := d.Command.Run(ctx, os.Args); runErr != nil {
		if errors.Is(runErr, errInterrupt) || errors.Is(runErr, context.Canceled) {
			return nil
		}

		return fmt.Errorf("run: %w", runErr)
	}

	return nil
}

// runWithHooks runs the demo setup, the run itself and the demo cleanup. If
// the context gets cancelled, for example by a signal, the cleanup still runs
// with a fresh timeout and only errors not caused by the interruption are
// returned.
func (d *Demo) runWithHooks(ctx context.Context, cmd *cli.Command, runFn runAction) error {
	var runErr error

	if err := d.setup(ctx, cmd); err != nil {
		if ctx.Err() == nil {
			return fmt.Errorf("setup: %w", err)
		}
	} else if ctx.Err() == nil {
		runErr = runFn(ctx, cmd)
	}

	cleanupCtx, cancel := cleanupContext(ctx)
	defer cancel()

	if err := d.cleanup(cleanupCtx, cmd); err != nil {
		runErr = errors.Join(runErr, &cleanupError{err: err})
	}

	if ctxErr := ctx.Err(); ctxErr != nil {
		if err := withoutInterruption(runErr, ctxErr); err != nil {
			return err
		}

		return fmt.Errorf("interrupted: %w", ctxErr)
	}

	// A Ctrl-C keypress in raw mode interrupts like a signal does.
	if errors.Is(runErr, errInterrupt) {
		if err := withoutInterruption(runErr, errInterrupt); err != nil {
			return err
		}

		return errInterrupt
	}

	return runErr
}

// cleanupContext returns the context for cleanup functions, which the first
// signal does not cancel. If the demo got interrupted already, the cleanup
// gets a fresh timeout instead.
func cleanupContext(ctx context.Context) (context.Context, context.CancelFunc) {
	if ctx.Err() != nil {
		return context.WithTimeout(context.WithoutCancel(ctx), cleanupTimeout)
	}

	return context.WithoutCancel(ctx), func() {}
}

// withoutInterruption removes the errors caused by the cause of an
// interruption from err, while keeping cleanup errors and all other joined
// errors.
func withoutInterruption(err, cause error) error {
	// Only the top-level joined errors get split up.
	if joined, ok := err.(interface{ Unwrap() []error }); ok {
		var kept []error

		for _, e := range joined.Unwrap() {
			if e = withoutInterruption(e, cause); e != nil {
				kept = append(kept, e)
			}
		}

		return errors.Join(kept...)
	}

	if _, ok := errors.AsType[*cleanupError](err); !ok && errors.Is(err, cause) {
		return nil
	}

	return err
}

func (d *Demo) optionsFrom(ctx context.Context, cmd *cli.Command) Options {
	opts := optionsFrom(ctx, cmd)
	resolvedTheme := resolveTheme(opts.Theme)

	opts.Theme = resolvedTheme
	opts.Palette = d.paletteForTheme(resolvedTheme)

	initColorFunctions(&opts)

	return opts
}

func (d *Demo) paletteForTheme(theme string) Palette {
	if theme == ThemeLight {
		return d.palettes.light
	}

	return d.palettes.dark
}

func resolveTheme(theme string) string {
	normalized, err := normalizedTheme(theme)
	if err != nil {
		return ThemeDark
	}

	if normalized == ThemeAuto {
		if detectLightBackground() {
			return ThemeLight
		}

		return ThemeDark
	}

	return normalized
}

func normalizedTheme(theme string) (string, error) {
	switch strings.ToLower(strings.TrimSpace(theme)) {
	case ThemeAuto:
		return ThemeAuto, nil
	case ThemeLight:
		return ThemeLight, nil
	case ThemeDark:
		return ThemeDark, nil
	default:
		return "", fmt.Errorf("%w %q: allowed values are auto, dark, light", errInvalidTheme, theme)
	}
}

func validateTheme(theme string) error {
	_, err := normalizedTheme(theme)

	return err
}
