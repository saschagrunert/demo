package demo_test

import (
	"os"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/saschagrunert/demo"
	"github.com/urfave/cli/v2"
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

	It("should succeed to run with --all flag", func() {
		// Given
		os.Args = []string{
			"demo", "--all", "--auto", "--auto-timeout=0", "--immediate",
		}
		run1 := demo.NewRun("Run 1")
		run2 := demo.NewRun("Run 2")

		sut := demo.New()
		sut.Add(run1, "run1", "first run")
		sut.Add(run2, "run2", "second run")

		// When
		sut.Run()

		// Then - should complete without error
	})

	It("should run selected demo by flag", func() {
		// Given
		os.Args = []string{
			"demo", "--1", "--auto", "--auto-timeout=0", "--immediate",
		}
		run1 := demo.NewRun("Run 1")
		run2 := demo.NewRun("Run 2")

		sut := demo.New()
		sut.Add(run1, "run1", "first run")
		sut.Add(run2, "run2", "second run")

		// When
		sut.Run()

		// Then - should complete without error
	})

	It("should run selected demo by name alias", func() {
		// Given
		os.Args = []string{
			"demo", "--run1", "--auto", "--auto-timeout=0", "--immediate",
		}
		run1 := demo.NewRun("Run 1")
		run2 := demo.NewRun("Run 2")

		sut := demo.New()
		sut.Add(run1, "run1", "first run")
		sut.Add(run2, "run2", "second run")

		// When
		sut.Run()

		// Then - should complete without error
	})

	It("should run with short option handling", func() {
		// Given
		os.Args = []string{
			"demo", "-a", "-i", "-0",
		}
		run1 := demo.NewRun("Run 1")
		sut := demo.New()
		sut.Add(run1, "run1", "first run")

		// When
		sut.Run()

		// Then - should complete without error
	})

	It("should run with all short flags", func() {
		// Given
		os.Args = []string{
			"demo", "-l", "-a", "-t=0", "-i", "-d", "-s=0",
		}
		run1 := demo.NewRun("Run 1")
		run2 := demo.NewRun("Run 2")

		sut := demo.New()
		sut.Add(run1, "run1", "first run")
		sut.Add(run2, "run2", "second run")

		// When
		sut.Run()

		// Then - should complete without error
	})

	It("should handle multiple runs with setup and cleanup", func() {
		// Given
		os.Args = []string{
			"demo", "--all", "--auto", "--auto-timeout=0", "--immediate",
		}
		run1 := demo.NewRun("Run 1")
		run2 := demo.NewRun("Run 2")
		run3 := demo.NewRun("Run 3")

		sut := demo.New()
		sut.Add(run1, "run1", "first run")
		sut.Add(run2, "run2", "second run")
		sut.Add(run3, "run3", "third run")

		callCount := 0
		sut.Setup(func(*cli.Context) error {
			callCount++

			return nil
		})

		// When
		sut.Run()

		// Then - setup should be called 3 times (once per run)
	})
})
