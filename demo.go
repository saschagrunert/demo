package demo

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"time"

	"github.com/saschagrunert/ccli/v3"
	"github.com/urfave/cli/v3"
)

type Demo struct {
	*cli.Command

	runs    []*runFlag
	setup   func(context.Context, *cli.Command) error
	cleanup func(context.Context, *cli.Command) error
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

	// FlagBreakPoint is the flag for doing `auto` but with breakpoint.
	FlagBreakPoint = "with-breakpoints"

	// FlagContinueOnError is the flag for steps continue running if
	// there is an error.
	FlagContinueOnError = "continue-on-error"

	// FlagContinuously is the flag for running the demos continuously without
	// any end.
	FlagContinuously = "continuously"

	// FlagDryRun only prints the command in the stdout.
	FlagDryRun = "dry-run"

	// FlagHideDescriptions is the flag for hiding the descriptions.
	FlagHideDescriptions = "hide-descriptions"

	// FlagImmediate is the flag for disabling the text animations.
	FlagImmediate = "immediate"

	// FlagNoColor true to print without colors, special characters for writing into file.
	FlagNoColor = "no-color"

	// FlagSkipSteps is the flag for skipping n amount of steps.
	FlagSkipSteps = "skip-steps"

	// FlagShell is the flag for defining the shell that is used to execute the command(s).
	FlagShell = "shell"

	// FlagTypewriterSpeed is the flag for configuring typewriter animation speed (max milliseconds per character).
	FlagTypewriterSpeed = "typewriter-speed"

	// DefaultTypewriterSpeed is the default maximum milliseconds per character for typewriter animation.
	DefaultTypewriterSpeed = 40
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
		&cli.IntFlag{
			Name:  FlagTypewriterSpeed,
			Usage: "maximum milliseconds per character for typewriter animation",
			Value: DefaultTypewriterSpeed,
		},
	}
}

func collectRunFunctions(cmd *cli.Command, runs []*runFlag) []runAction {
	runFns := make([]runAction, 0, len(runs))

	for _, x := range runs {
		if isFlagSet(cmd, x.flag) || cmd.Bool(FlagAll) {
			runFns = append(runFns, x.run.Run)
		}
	}

	return runFns
}

type runAction func(context.Context, *cli.Command) error

func isFlagSet(cmd *cli.Command, flag cli.Flag) bool {
	for _, name := range flag.Names() {
		if cmd.Bool(name) {
			return true
		}
	}

	return false
}

func createRunSelected(demo *Demo, ctx context.Context, cmd *cli.Command, runFns []runAction) func() error {
	return func() error {
		for _, runFn := range runFns {
			if err := demo.setup(ctx, cmd); err != nil {
				return err
			}

			if err := runFn(ctx, cmd); err != nil {
				return err
			}

			if err := demo.cleanup(ctx, cmd); err != nil {
				return err
			}
		}

		return nil
	}
}

func runContinuously(ctx context.Context, runSelected func() error) error {
	for {
		if err := ctx.Err(); err != nil {
			return fmt.Errorf("context cancelled: %w", err)
		}

		if err := runSelected(); err != nil {
			return err
		}
	}
}

// New creates a new Demo instance.
func New() *Demo {
	emptyFn := func(context.Context, *cli.Command) error { return nil }
	demo := &Demo{
		Command: ccli.NewCommand(),
		runs:    nil,
		setup:   emptyFn,
		cleanup: emptyFn,
	}

	demo.Flags = createFlags()
	demo.UseShortOptionHandling = true

	demo.Action = func(ctx context.Context, cmd *cli.Command) error {
		runFns := collectRunFunctions(cmd, demo.runs)
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
func (d *Demo) RunE() error {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	interrupted := make(chan os.Signal, 1)

	signal.Notify(interrupted, os.Interrupt)
	defer signal.Stop(interrupted)

	done := make(chan struct{})

	go func() {
		select {
		case <-interrupted:
			cancel()

			cleanupCtx, cleanupCancel := context.WithTimeout(context.Background(), cleanupTimeout)
			defer cleanupCancel()

			if err := d.cleanup(cleanupCtx, d.Command); err != nil {
				log.Printf("unable to cleanup: %v", err)
			}

			os.Exit(0)
		case <-done:
			return
		}
	}()

	err := d.Command.Run(ctx, os.Args)

	close(done)

	if err != nil {
		return fmt.Errorf("run: %w", err)
	}

	return nil
}
