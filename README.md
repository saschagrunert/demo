# demo

[![ci](https://github.com/saschagrunert/demo/actions/workflows/test.yml/badge.svg)](https://github.com/saschagrunert/demo/actions/workflows/test.yml)
[![codecov](https://codecov.io/gh/saschagrunert/demo/branch/main/graph/badge.svg)](https://codecov.io/gh/saschagrunert/demo)
[![docs](https://pkg.go.dev/badge/github.com/saschagrunert/demo)](https://pkg.go.dev/github.com/saschagrunert/demo)

![demo](.github/demo.svg)

### A framework for performing live pre-recorded command line demos in the wild 📼

Recording command line demos can be a difficult topic these days. Doing a video
record has the drawback of lacking flexibility and reduced interactivity during
the demo. Typing everything by our own is error prone and distracts the audience
from the actual topic we want to show them. So we need something in between,
which is easy to use...

This framework should solve the issue by provided interactive demos from your
command line!

# Usage

Every _demo_ is a stand-alone command line application which consist of
multiple _runs_. For example, if we create a demo like this:

```go
package main

import (
	"github.com/saschagrunert/demo"
)

func main() {
	demo.New().Run()
}
```

Then this demo already contains features like _auto-play_. We can verify this
checking the help output of the executable:

```
NAME:
   main - A new cli application

USAGE:
   main [global options]

GLOBAL OPTIONS:
   --all, -l                             run all demos
   --auto, -a                            run the demo in automatic mode, where every step gets executed automatically
   --dry-run                             only print the commands without executing them
   --no-color                            disable colored output
   --theme string                        set the color theme: auto (detected from COLORFGBG), dark or light (default: auto)
   --auto-timeout duration, -t duration  time to wait between steps in auto mode (default: 1s)
   --with-breakpoints                    pause at breakpoint steps
   --continue-on-error                   continue if a step fails
   --continuously, -c                    run the demos continuously without any end
   --hide-descriptions, -d               hide descriptions between the steps
   --immediate, -i                       immediately output without the typewriter animation
   --skip-steps int, -s int              skip the amount of initial steps within the demo (default: 0)
   --shell string                        define the shell that is used to execute the command(s) (default: bash)
   --typewriter-speed int                maximum milliseconds per character for typewriter animation (default: 40)
   --help, -h                            show help
```

The application is based on the [urfave/cli](https://github.com/urfave/cli)
framework, which means that we have every possibility to change the command
before actually running it.

```go
// Create a new demo CLI application
d := demo.New()

// A demo is an usual urfave/cli application, which means
// that we can set its properties as expected:
d.Name = "A demo of something"
d.Usage = "Learn how this framework is being used"
```

Theme support works out of the box (`--theme auto|dark|light`). The `auto`
theme picks the light palette if the `COLORFGBG` environment variable reports a
light terminal background, and the dark palette otherwise. You only need
constructor options when overriding the defaults. Palettes use the attributes of
the [fatih/color](https://github.com/fatih/color) package, and empty fields
fall back to the default palette of the theme, so `demo.Palette{}` keeps the
defaults:

```go
import "github.com/fatih/color"

myDarkPalette := demo.Palette{
	Title:       []color.Attribute{color.FgCyan},
	Description: []color.Attribute{color.FgWhite, color.Faint},
	Command:     []color.Attribute{color.FgGreen},
}
myLightPalette := demo.Palette{
	Title:       []color.Attribute{color.FgBlue},
	Description: []color.Attribute{color.FgHiBlue},
	Command:     []color.Attribute{color.FgGreen},
}

d := demo.New(
	demo.WithPalettes(myDarkPalette, myLightPalette),
)
```

## Creating runs inside demos

To have something to show, we need to create a run and add it to the demo. This
can be done by using the `demo.Add()` method:

```go
func main() {
	// Create a new demo CLI application
	d := demo.New()

	// Register the demo run
	d.Add(example(), "demo-0", "just an example demo run")

	// Run the application, which registers all signal handlers and waits for
	// the app to exit
	d.Run()
}

// example is the single demo run for this application
func example() *demo.Run {
	// A new run contains a title and an optional description
	r := demo.NewRun(
		"Demo Title",
		"Some additional",
		"multi-line description",
		"is possible as well!",
	)

	// A single step can consist of a description and a command to be executed.
	r.Step(demo.S(
		"This is a possible",
		"description of the following command",
		"to be executed",
	), demo.S(
		"echo hello world",
	))

	// Commands do not need to have a description, so we could set it to `nil`
	r.Step(nil, demo.S(
		"echo without description",
		"but this can be executed in",
		"multiple lines as well",
	))

	// It is also not needed at all to provide a command
	r.Step(demo.S(
		"Just a description without a command",
	), nil)

	// Steps that are allowed to fail use StepCanFail
	r.StepCanFail(demo.S("This command may fail"), demo.S("exit 1"))

	// Breakpoints pause execution when --with-breakpoints is set
	r.BreakPoint()

	return r
}
```

The `example()` function creates a new demo run, which itself contains of
multiple steps. These steps are executed in order, can contain a description and
a command to be executed. The elements of a command are joined with spaces into
a single shell command and displayed with one line per element, so
`demo.S("cd x", "ls")` runs `cd x ls`. Use `&&` or separate steps to run
multiple commands.

`StepCanFail` works like `Step` but does not stop the demo when the command
exits with a non-zero status. The `--continue-on-error` flag applies this
behavior to all steps.

`BreakPoint` inserts a pause that only takes effect when `--with-breakpoints`
is passed. This is useful for inserting manual checkpoints in an otherwise
automatic demo. Breakpoints do not count as steps, neither for the `[n/N]`
step counter nor for `--skip-steps`.

## Running a demo

Every run gets a flag with its name, so the example above runs with
`./demo --demo-0`, and `--all` runs all runs in the order they were added.
Without a selected run, the demo prints its help. Invalid flags print a single
line with the error, and `--help` shows all flags.

By default, the demo waits before every description and command. On a terminal,
any key continues; otherwise, for example with piped input, a newline does. With
`--auto`, the demo continues on its own after `--auto-timeout`. `--dry-run`
prints the commands without executing them, but setup and cleanup functions
still run.

## Setup and Cleanup functions

It is also possible to do something before or after each run. For this the setup
and cleanup functions can be set to the demo:

```go
func main() {
	// Create a new demo CLI application
	d := demo.New()

	// Be able to run a Setup/Cleanup function before/after each run
	d.Setup(setup)
	d.Cleanup(cleanup)
}

// setup will run before every demo
func setup(ctx context.Context, _ *cli.Command) error {
	// EnsureWithContext can be used for easy sequential command execution
	return demo.EnsureWithContext(ctx,
		"echo 'Doing first setup...'",
		"echo 'Doing second setup...'",
		"echo 'Doing third setup...'",
	)
}

// cleanup will run after every demo
func cleanup(ctx context.Context, _ *cli.Command) error {
	return demo.EnsureWithContext(ctx, "echo 'Doing cleanup...'")
}
```

A run can have its own setup and cleanup functions as well:

```go
r.Setup(func(ctx context.Context) error {
	return demo.EnsureWithContext(ctx, "mkdir -p /tmp/demo")
})
r.Cleanup(func(ctx context.Context) error {
	return demo.EnsureWithContext(ctx, "rm -rf /tmp/demo")
})
```

The functions run in this order: demo setup, run setup, the steps, run cleanup
and demo cleanup. Both cleanup functions also run if the setup or the run fails
or gets interrupted, and their context is not cancelled by the interruption.
`Ensure`, `EnsureWithContext` and `MustEnsure` run commands through `bash`
without showing their output, which suits setup and cleanup work.

## Working directory

The working directory for command execution can be configured per run or changed
between steps:

```go
func example() *demo.Run {
	r := demo.NewRun("Working Directory Demo")

	// Set the initial working directory for all steps
	r.SetWorkDir("/tmp")

	r.Step(demo.S("Show current directory"), demo.S("pwd"))

	// Change the working directory between steps
	r.Chdir("/home")

	r.Step(demo.S("Now in a different directory"), demo.S("pwd"))

	return r
}
```

`SetWorkDir` sets the initial working directory for the run. `Chdir` inserts a
step that changes the working directory for all subsequent steps and displays
`> cd <dir>` in the output. A relative `Chdir` path is resolved against the
current working directory of the run, while shell features like `~` or `cd -`
are not supported. If no working directory is set, commands run in the current
process directory.

## Environment variables

Custom environment variables can be set for all steps in a run:

```go
func example() *demo.Run {
	r := demo.NewRun("Environment Demo")

	r.SetEnv("MY_VAR=hello", "OTHER_VAR=world")

	r.Step(demo.S("Show env"), demo.S("echo $MY_VAR $OTHER_VAR"))

	return r
}
```

Variables are appended to the current process environment. Multiple calls to
`SetEnv` accumulate variables.

## Signal handling

The demo framework handles SIGINT (Ctrl-C) and SIGTERM. When either signal is
received, the current context is cancelled, which stops a running command, the
typewriter animation or the wait for input. The run-level cleanup runs first,
followed by the demo-level cleanup with a fresh 10-second timeout. A signal
that arrives while a cleanup runs does not cancel it. The demo then exits
cleanly with no error, in continuous mode as well, unless a cleanup failed. A
second signal terminates the process right away, for example if a cleanup
hangs.

While waiting for user input, and during the typewriter animation when not in
`--auto` mode, the terminal is in raw mode, which means the kernel does not
translate Ctrl-C into SIGINT. Instead, the framework detects the Ctrl-C byte
directly and stops the demo, running cleanup as expected. In raw mode, any key
advances the demo, and an escape sequence like an arrow key counts as a single
keypress.

## Programmatic usage

`RunE` starts the demo and returns any error instead of calling `os.Exit`.
`RunContext` does the same with a custom context and arguments, which is useful
for testing or when embedding a demo in a larger application:

```go
d := demo.New()
if err := d.RunContext(ctx, []string{"demo", "--all", "--auto"}); err != nil {
	log.Fatal(err)
}
```

Individual runs can be executed directly with `RunWithOptions`, bypassing the
CLI flag parsing. The options are copied, so they can be reused, and `nil` uses
the defaults. Unlike the CLI flags, a zero `AutoTimeout` means no delay:

```go
r := demo.NewRun("My Run")
r.Step(demo.S("Hello"), demo.S("echo hello"))

err := r.RunWithOptions(&demo.Options{
	Auto:        true,
	AutoTimeout: 500 * time.Millisecond,
	Shell:       "bash",
})
```

See the `Options` struct documentation for the full list of fields. A run
writes to `os.Stdout` and reads from `os.Stdin` by default, which `SetOutput`
and `SetInput` can change.

# Contributing

You want to contribute to this project? Wow, thanks! So please just fork it and
send me a pull request.
