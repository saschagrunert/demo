package main

import (
	. "github.com/saschagrunert/demo" // nolint
)

func main() {
	demo := New()
	demo.Name = "A demo of something"
	demo.Usage = "Learn how this framework is being used"
	demo.HideVersion = true

	demo.Add(example(), "name", "description")
	demo.Run()
}

func example() *Run {
	r := NewRun(
		"Title",
		"Some additional",
		"multiline description",
	)

	r.Step(S(
		"This is a possible",
		"description of the command",
		"to be executed",
	), S(
		"echo hello world",
	))

	// Commands to not need to have a description
	r.Step(nil, S(
		"echo without description",
	))

	// It is also not needed to provide a command
	r.Step(S(
		"Just a description without a command",
	), nil)

	return r
}
