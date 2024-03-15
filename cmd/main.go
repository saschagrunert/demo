package main

import (
	. "github.com/saschagrunert/demo" //nolint:stylecheck // dot imports are intended here
	"github.com/urfave/cli/v2"
)

func main() {
	// Create a new demo CLI application
	demo := New()

	// A demo is an usual urfave/cli application, which means
	// that we can set its properties as expected:
	demo.Name = "A demo of something"
	demo.Usage = "Learn how this framework is being used"
	demo.HideVersion = true

	// Be able to run a Setup/Cleanup function before/after each run
	demo.Setup(setup)
	demo.Cleanup(cleanup)

	// Register the demo run
	demo.Add(example(), "demo-0", "just an example demo run")

	// Run the application, which registers all signal handlers and waits for
	// the app to exit
	demo.Run()
}

// setup will run before every demo.
func setup(*cli.Context) error {
	// Ensure can be used for easy sequential command execution
	return Ensure(
		"echo 'Doing first setup…'",
		"echo 'Doing second setup…'",
		"echo 'Doing third setup…'",
	)
}

// setup will run after every demo.
func cleanup(*cli.Context) error {
	return Ensure("echo 'Doing cleanup…'")
}

// example is the single demo run for this application.
func example() *Run {
	// A new run contains a title and an optional description
	r := NewRun(
		"Demo Title",
		"Some additional",
		"multi-line description",
		"is possible as well!",
	)

	// A single step can consist of a description and a command to be executed
	r.Step(S(
		"This is a possible",
		"description of the following command",
		"to be executed",
	), S(
		"echo hello world",
	))

	// Commands do not need to have a description, so we could set it to `nil`
	r.Step(nil, S(
		"echo without description",
		"but this can be executed in",
		"multiple lines as well",
	))

	// It is also not needed at all to provide a command
	r.Step(S(
		"Just a description without a command",
	), nil)

	return r
}
