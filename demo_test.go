package demo_test

import (
	"context"
	"os"
	"sync"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/saschagrunert/demo"
	"github.com/urfave/cli/v3"
)

// osArgsMu protects os.Args from concurrent test access.
//
//nolint:gochecknoglobals // required for test synchronization
var osArgsMu sync.Mutex

func withArgs(args []string, fn func()) {
	osArgsMu.Lock()

	defer osArgsMu.Unlock()

	orig := os.Args

	defer func() { os.Args = orig }()

	os.Args = args

	fn()
}

var _ = Describe("Demo", func() {
	It("should succeed to run", func() {
		withArgs([]string{"demo"}, func() {
			sut := demo.New()
			Expect(sut.RunE()).To(Succeed())
		})
	})

	It("should succeed to run with example", func() {
		withArgs([]string{
			"demo", "--auto", "--auto-timeout=0", "--immediate", "--title",
		}, func() {
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

			sut.Setup(func(context.Context, *cli.Command) error {
				succeed.setup = true

				return nil
			})
			sut.Cleanup(func(context.Context, *cli.Command) error {
				succeed.cleanup = true

				return nil
			})

			Expect(sut.RunE()).To(Succeed())
			Expect(succeed.setup).To(BeTrue())
			Expect(succeed.cleanup).To(BeTrue())
		})
	})

	It("should succeed to run with --all flag", func() {
		withArgs([]string{
			"demo", "--all", "--auto", "--auto-timeout=0", "--immediate",
		}, func() {
			run1 := demo.NewRun("Run 1")
			run2 := demo.NewRun("Run 2")

			sut := demo.New()
			sut.Add(run1, "run1", "first run")
			sut.Add(run2, "run2", "second run")

			Expect(sut.RunE()).To(Succeed())
		})
	})

	It("should run selected demo by flag", func() {
		withArgs([]string{
			"demo", "--run2", "--auto", "--auto-timeout=0", "--immediate",
		}, func() {
			run1 := demo.NewRun("Run 1")
			run2 := demo.NewRun("Run 2")

			sut := demo.New()
			sut.Add(run1, "run1", "first run")
			sut.Add(run2, "run2", "second run")

			Expect(sut.RunE()).To(Succeed())
		})
	})

	It("should run selected demo by name alias", func() {
		withArgs([]string{
			"demo", "--run1", "--auto", "--auto-timeout=0", "--immediate",
		}, func() {
			run1 := demo.NewRun("Run 1")
			run2 := demo.NewRun("Run 2")

			sut := demo.New()
			sut.Add(run1, "run1", "first run")
			sut.Add(run2, "run2", "second run")

			Expect(sut.RunE()).To(Succeed())
		})
	})

	It("should run with short option handling", func() {
		withArgs([]string{
			"demo", "-a", "-i", "--run1",
		}, func() {
			run1 := demo.NewRun("Run 1")
			sut := demo.New()
			sut.Add(run1, "run1", "first run")

			Expect(sut.RunE()).To(Succeed())
		})
	})

	It("should run with all short flags", func() {
		withArgs([]string{
			"demo", "-l", "-a", "-t=0", "-i", "-d", "-s=0",
		}, func() {
			run1 := demo.NewRun("Run 1")
			run2 := demo.NewRun("Run 2")

			sut := demo.New()
			sut.Add(run1, "run1", "first run")
			sut.Add(run2, "run2", "second run")

			Expect(sut.RunE()).To(Succeed())
		})
	})

	It("should handle multiple runs with setup and cleanup", func() {
		withArgs([]string{
			"demo", "--all", "--auto", "--auto-timeout=0", "--immediate",
		}, func() {
			run1 := demo.NewRun("Run 1")
			run2 := demo.NewRun("Run 2")
			run3 := demo.NewRun("Run 3")

			sut := demo.New()
			sut.Add(run1, "run1", "first run")
			sut.Add(run2, "run2", "second run")
			sut.Add(run3, "run3", "third run")

			callCount := 0

			sut.Setup(func(context.Context, *cli.Command) error {
				callCount++

				return nil
			})

			Expect(sut.RunE()).To(Succeed())
			Expect(callCount).To(Equal(3))
		})
	})
})
