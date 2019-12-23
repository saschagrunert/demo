package demo_test

import (
	"os"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/urfave/cli/v2"

	"github.com/saschagrunert/demo"
)

var _ = t.Describe("Demo", func() {

	It("should succeed to run", func() {
		// Given
		os.Args = []string{"demo"}
		sut := demo.New()

		// When
		sut.Run()
	})

	It("should succeed to run with example", func() {
		// Given
		os.Args = []string{
			"demo", "--auto", "--auto-timeout=0", "--immediate", "-0",
		}
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
			), demo.S("echo hello world"))

			r.Step(nil, demo.S("echo without description"))
			r.Step(demo.S("Just a description without a command"), nil)
			return r
		}
		sut := demo.New()
		sut.Add(example(), "title", "description")

		type success struct{ setup, cleanup bool }
		succeed := &success{}

		sut.Setup(func(*cli.Context) error {
			succeed.setup = true
			return nil
		})
		sut.Cleanup(func(*cli.Context) error {
			succeed.cleanup = true
			return nil
		})

		// When
		sut.Run()

		// Then
		Expect(succeed.setup).To(BeTrue())
		Expect(succeed.cleanup).To(BeTrue())
	})
})
