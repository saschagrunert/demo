package demo

import (
	"fmt"
	"log"
	"os"
	"os/signal"
	"strconv"
	"time"

	"github.com/urfave/cli/v2"
)

type Demo struct {
	*cli.App

	runs    []*runFlag
	setup   func(*cli.Context) error
	cleanup func(*cli.Context) error
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

	// FlagAutoTimeout is the flag for the timeout to be waited when `auto` is
	// enabled.
	FlagAutoTimeout = "auto-timeout"

	// FlagBreakPoint is the flag for doing`auto` but with breakpoint.
	FlagBreakPoint = "with-breakpoints"

	// FlagContinueOnError is the flag for steps continue running if
	// there is an error.
	FlagContinueOnError = "continue-on-error"

	// FlagContinuously is the flag for running the demos continuously without
	// any end.
	FlagContinuously = "continuously"

	// DryRun only prints the command in the stdout.
	FlagDryRun = "dry-run"

	// FlagHideDescriptions is the flag for hiding the descriptions.
	FlagHideDescriptions = "hide-descriptions"

	// FlagImmediate is the flag for disabling the text animations.
	FlagImmediate = "immediate"

	// NoColor true to print without colors, special characters for writing into file.
	FlagNoColor = "no-color"

	// FlagSkipSteps is the flag for skipping n amount of steps.
	FlagSkipSteps = "skip-steps"

	// FlagShell is the flag for defining the shell that is used to execute the command(s).
	FlagShell = "shell"
)

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
			Usage: "run the demo and only prints the commands",
		},
		&cli.BoolFlag{
			Name:  FlagNoColor,
			Usage: "run the demo and output to be without colors",
		},
		&cli.DurationFlag{
			Name:    FlagAutoTimeout,
			Aliases: []string{"t"},
			Usage:   "the timeout to be waited when `auto` is enabled",
			Value:   1 * time.Second,
		},
		&cli.BoolFlag{
			Name:  FlagBreakPoint,
			Usage: "breakpoint",
		},
		&cli.BoolFlag{
			Name:  FlagContinueOnError,
			Usage: "continue if there a step fails",
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
			Usage:       "define the shell that is used to execute the command(s)",
			DefaultText: "bash",
		},
	}
}

func collectRunFunctions(ctx *cli.Context, runs []*runFlag) []cli.ActionFunc {
	runFns := make([]cli.ActionFunc, 0, len(runs))

	for _, x := range runs {
		if isFlagSet(ctx, x.flag) || ctx.Bool(FlagAll) {
			runFns = append(runFns, x.run.Run)
		}
	}

	return runFns
}

func isFlagSet(ctx *cli.Context, flag cli.Flag) bool {
	for _, name := range flag.Names() {
		if ctx.Bool(name) {
			return true
		}
	}

	return false
}

func createRunSelected(demo *Demo, ctx *cli.Context, runFns []cli.ActionFunc) func() error {
	return func() error {
		for _, runFn := range runFns {
			if err := demo.setup(ctx); err != nil {
				return err
			}

			if err := runFn(ctx); err != nil {
				return err
			}

			if err := demo.cleanup(ctx); err != nil {
				return err
			}
		}

		return nil
	}
}

func runContinuously(ctx *cli.Context, runSelected func() error) error {
	for {
		select {
		case <-ctx.Done():
			return fmt.Errorf("context cancelled: %w", ctx.Err())
		default:
			if err := runSelected(); err != nil {
				return err
			}
		}
	}
}

// New creates a new Demo instance.
func New() *Demo {
	app := cli.NewApp()
	app.UseShortOptionHandling = true
	app.Flags = createFlags()

	emptyFn := func(*cli.Context) error { return nil }
	demo := &Demo{
		App:     app,
		runs:    nil,
		setup:   emptyFn,
		cleanup: emptyFn,
	}

	app.Action = func(ctx *cli.Context) error {
		runFns := collectRunFunctions(ctx, demo.runs)
		runSelected := createRunSelected(demo, ctx, runFns)

		if ctx.Bool(FlagContinuously) {
			return runContinuously(ctx, runSelected)
		}

		return runSelected()
	}

	return demo
}

// Setup sets the cleanup function called before each run.
func (d *Demo) Setup(setupFn func(*cli.Context) error) {
	d.setup = setupFn
}

// Cleanup sets the cleanup function called after each run.
func (d *Demo) Cleanup(cleanupFn func(*cli.Context) error) {
	d.cleanup = cleanupFn
}

func (d *Demo) Add(run *Run, name, description string) {
	flag := &cli.BoolFlag{
		Name:    strconv.Itoa(len(d.runs)),
		Aliases: []string{name},
		Usage:   description,
	}
	d.Flags = append(d.Flags, flag)
	d.runs = append(d.runs, &runFlag{run, flag})
}

// Run starts the demo.
func (d *Demo) Run() {
	// Catch interrupts and cleanup
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)

	done := make(chan bool)

	go func() {
		for {
			select {
			case <-c:
				// Create a minimal context for cleanup on interrupt
				ctx := cli.NewContext(d.App, nil, nil)
				if err := d.cleanup(ctx); err != nil {
					log.Printf("unable to cleanup: %v", err)
				}

				os.Exit(0)
			case <-done:
				return
			}
		}
	}()

	if err := d.App.Run(os.Args); err != nil {
		log.Printf("run failed: %v", err)
		signal.Stop(c)
		close(done)
		os.Exit(1)
	}

	signal.Stop(c)
	close(done)
}
