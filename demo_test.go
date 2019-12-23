package demo_test

import (
	"os"

	. "github.com/onsi/ginkgo"

	"github.com/saschagrunert/demo"
)

var _ = t.Describe("Demo", func() {
	BeforeEach(func() {
		os.Args = []string{
			"demo", "--auto", "--immediate",
		}
	})

	It("should succeed to run", func() {
		// Given
		sut := demo.New()

		// When
		sut.Run()
	})

	It("should succeed to run with example", func() {
		// Given
		example := func() *demo.Run {
			r := demo.NewRun(
				"Title",
				"Some additional",
				"multiline description",
			)

			r.Step(demo.S(
				"This is a possible",
				"description of the command",
				"to be executed",
			), demo.S(
				"echo hello world",
			))

			// Commands to not need to have a description
			r.Step(nil, demo.S(
				"echo without description",
			))

			// It is also not needed to provide a command
			r.Step(demo.S(
				"Just a description without a command",
			), nil)

			return r
		}
		sut := demo.New()
		sut.Add(example(), "title", "description")

		// When
		sut.Run()
	})
})
