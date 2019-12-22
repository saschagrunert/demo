package demo

import (
	"fmt"
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

func New() *Demo {
	app := cli.NewApp()
	app.UseShortOptionHandling = true

	app.Flags = []cli.Flag{
		&cli.BoolFlag{
			Name:    "all",
			Aliases: []string{"l"},
			Usage:   "run all demos",
		},
		&cli.BoolFlag{
			Name:    "auto",
			Aliases: []string{"a"},
			Usage: "run the demo in automatic mode, " +
				"where every step gets executed automatically",
		},
		&cli.DurationFlag{
			Name:    "auto-timeout",
			Aliases: []string{"t"},
			Usage:   "the timeout to be waited when `auto` is enabled",
			Value:   3 * time.Second,
		},
		&cli.BoolFlag{
			Name:    "continuously",
			Aliases: []string{"c"},
			Usage:   "run the demos continuously without any end",
		},
		&cli.BoolFlag{
			Name:    "immediate",
			Aliases: []string{"i"},
			Usage:   "immediately output without the typewriter animation",
		},
		&cli.IntFlag{
			Name:    "skip-steps",
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
			if ctx.Bool("all") || isSet {
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
		if ctx.Bool("continuously") {
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
				_ = d.Cleanup(nil)
			}
			os.Exit(0)
		}
	}()

	if err := d.App.Run(os.Args); err != nil {
		fmt.Printf("run failed: %v", err)
		os.Exit(1)
	}
}
