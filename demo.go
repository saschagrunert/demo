package demo

import (
	"fmt"
	"log"
	"os"
	"os/signal"
	"time"

	"github.com/urfave/cli/v2"
)

type Demo struct {
	*cli.App
	runs    []*runFlag
	Setup   func(*cli.Context) error
	Cleanup func(*cli.Context) error
}

type runFlag struct {
	run  *Run
	flag cli.Flag
}

const (
	FlagAll          = "all"
	FlagAuto         = "auto"
	FlagAutoTimeout  = "auto-timeout"
	FlagContinuously = "continuously"
	FlagImmediate    = "immediate"
	FlagSkipSteps    = "skip-steps"
)

func New() *Demo {
	app := cli.NewApp()
	app.UseShortOptionHandling = true

	app.Flags = []cli.Flag{
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
		&cli.DurationFlag{
			Name:    FlagAutoTimeout,
			Aliases: []string{"t"},
			Usage:   "the timeout to be waited when `auto` is enabled",
			Value:   3 * time.Second,
		},
		&cli.BoolFlag{
			Name:    FlagContinuously,
			Aliases: []string{"c"},
			Usage:   "run the demos continuously without any end",
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
	}

	demo := &Demo{App: app, runs: nil}

	app.Action = func(ctx *cli.Context) error {
		runFns := []cli.ActionFunc{}

		for _, x := range demo.runs {
			isSet := false
			for _, name := range x.flag.Names() {
				if ctx.Bool(name) {
					isSet = true
				}
			}
			if ctx.Bool(FlagAll) || isSet {
				runFns = append(runFns, x.run.Run)
			}
		}

		runSelected := func() error {
			for _, runFn := range runFns {
				if demo.Setup != nil {
					if err := demo.Setup(ctx); err != nil {
						return err
					}
				}
				if err := runFn(ctx); err != nil {
					return err
				}
				if demo.Cleanup != nil {
					if err := demo.Cleanup(nil); err != nil {
						return err
					}
				}
			}
			return nil
		}
		if ctx.Bool(FlagContinuously) {
			for {
				if err := runSelected(); err != nil {
					return err
				}
			}
		}
		return runSelected()
	}

	return demo
}

func (d *Demo) Add(run *Run, name, description string) {
	flag := &cli.BoolFlag{
		Name:    fmt.Sprintf("%d", len(d.runs)),
		Aliases: []string{name},
		Usage:   description,
	}
	d.Flags = append(d.Flags, flag)
	d.runs = append(d.runs, &runFlag{run, flag})
}

func (d *Demo) Run() {
	// Catch interrupts and cleanup
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)
	go func() {
		for range c {
			if d.Cleanup != nil {
				if err := d.Cleanup(nil); err != nil {
					log.Printf("unable to cleanup: %v", err)
				}
			}
			os.Exit(0)
		}
	}()

	if err := d.App.Run(os.Args); err != nil {
		log.Printf("run failed: %v", err)
		os.Exit(1)
	}
}
