package demo_test

import (
	"errors"
	"flag"
	"strings"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/saschagrunert/demo"
	"github.com/urfave/cli/v2"
)

var (
	errSetupFailed   = errors.New("setup failed")
	errCleanupFailed = errors.New("cleanup failed")
	errWriteError    = errors.New("write error")
)

var _ = t.Describe("Run", func() {
	var (
		sut         *demo.Run
		out         *strings.Builder
		opts        demo.Options
		title       = "Test Title"
		description = []string{"Some", "description"}
	)

	BeforeEach(func() {
		sut = demo.NewRun(title, description...)
		Expect(sut).NotTo(BeNil())

		out = &strings.Builder{}
		err := sut.SetOutput(out)
		Expect(err).ToNot(HaveOccurred())

		opts = demo.Options{
			AutoTimeout: 0,
			Auto:        true,
			Immediate:   true,
			SkipSteps:   0,
		}
	})

	It("should succeed to run", func() {
		// Given
		// When
		err := sut.RunWithOptions(&opts)

		// Then
		Expect(err).ToNot(HaveOccurred())
		Expect(out).To(ContainSubstring(title))
		Expect(out).To(ContainSubstring(description[0]))
		Expect(out).To(ContainSubstring(description[1]))
	})

	It("should succeed to run with step", func() {
		// Given
		const (
			descriptionText = "Description Text"
			command         = "echo 'Some step'"
		)
		sut.Step(demo.S(descriptionText), demo.S(command))

		// When
		err := sut.RunWithOptions(&opts)

		// Then
		Expect(err).ToNot(HaveOccurred())
		Expect(out).To(ContainSubstring(title))
		Expect(out).To(ContainSubstring(description[0]))
		Expect(out).To(ContainSubstring(description[1]))
		Expect(out).To(ContainSubstring(descriptionText))
		Expect(out).To(ContainSubstring(command))
	})

	It("should succeed to run with step which can fail", func() {
		// Given
		const (
			descriptionText = "Description Text"
			command         = "exit 1"
		)
		sut.StepCanFail(demo.S(descriptionText), demo.S(command))

		// When
		err := sut.RunWithOptions(&opts)

		// Then
		Expect(err).ToNot(HaveOccurred())
	})

	It("should fail to set nil output", func() {
		// Given
		// When
		err := sut.SetOutput(nil)

		// Then
		Expect(err).To(HaveOccurred())
	})

	It("should succeed to run from a cli context", func() {
		// Given
		app := cli.NewApp()
		flagSet := &flag.FlagSet{}
		flagSet.Bool(demo.FlagAuto, true, "")
		flagSet.Bool(demo.FlagImmediate, true, "")

		ctx := cli.NewContext(app, flagSet, nil)

		// When
		err := sut.Run(ctx)

		// Then
		Expect(err).ToNot(HaveOccurred())
	})

	It("should succeed to run with setup and cleanup", func() {
		// Given
		setupCalled := false
		cleanupCalled := false

		sut.Setup(func() error {
			setupCalled = true

			return nil
		})

		sut.Cleanup(func() error {
			cleanupCalled = true

			return nil
		})

		// When
		err := sut.RunWithOptions(&opts)

		// Then
		Expect(err).ToNot(HaveOccurred())
		Expect(setupCalled).To(BeTrue())
		Expect(cleanupCalled).To(BeTrue())
	})

	It("should succeed to run with breakpoint disabled", func() {
		// Given
		opts.BreakPoint = false
		sut.BreakPoint()

		// When
		err := sut.RunWithOptions(&opts)

		// Then
		Expect(err).ToNot(HaveOccurred())
	})

	It("should succeed to run with hidden descriptions", func() {
		// Given
		opts.HideDescriptions = true
		sut.Step(demo.S("Hidden description"), demo.S("echo test"))

		// When
		err := sut.RunWithOptions(&opts)

		// Then
		Expect(err).ToNot(HaveOccurred())
		Expect(out.String()).ToNot(ContainSubstring("Hidden description"))
	})

	It("should succeed to run with dry-run", func() {
		// Given
		opts.DryRun = true
		sut.Step(demo.S("Test step"), demo.S("echo should not run"))

		// When
		err := sut.RunWithOptions(&opts)

		// Then
		Expect(err).ToNot(HaveOccurred())
		Expect(out.String()).To(ContainSubstring("echo should not run"))
	})

	It("should succeed to run with no-color", func() {
		// Given
		opts.NoColor = true
		sut.Step(demo.S("Test step"), demo.S("echo test"))

		// When
		err := sut.RunWithOptions(&opts)

		// Then
		Expect(err).ToNot(HaveOccurred())
	})

	It("should succeed to run with skip steps", func() {
		// Given
		opts.SkipSteps = 1
		sut.Step(demo.S("Skipped step"), demo.S("echo skipped"))
		sut.Step(demo.S("Executed step"), demo.S("echo executed"))

		// When
		err := sut.RunWithOptions(&opts)

		// Then
		Expect(err).ToNot(HaveOccurred())
		Expect(out.String()).ToNot(ContainSubstring("Skipped step"))
		Expect(out.String()).To(ContainSubstring("Executed step"))
	})

	It("should succeed to run with continue on error", func() {
		// Given
		opts.ContinueOnError = true
		sut.Step(demo.S("Failing step"), demo.S("exit 1"))
		sut.Step(demo.S("Should still run"), demo.S("echo after error"))

		// When
		err := sut.RunWithOptions(&opts)

		// Then
		Expect(err).ToNot(HaveOccurred())
		Expect(out.String()).To(ContainSubstring("Should still run"))
	})

	It("should succeed to run with custom shell", func() {
		// Given
		opts.Shell = "sh"
		sut.Step(demo.S("Shell test"), demo.S("echo test"))

		// When
		err := sut.RunWithOptions(&opts)

		// Then
		Expect(err).ToNot(HaveOccurred())
	})

	It("should succeed to run with custom typewriter speed", func() {
		// Given
		opts.TypewriterSpeed = 10
		opts.Immediate = false
		opts.Auto = true
		opts.AutoTimeout = 0
		sut.Step(demo.S("Speed test"), demo.S("echo test"))

		// When
		err := sut.RunWithOptions(&opts)

		// Then
		Expect(err).ToNot(HaveOccurred())
	})

	It("should handle write errors gracefully", func() {
		// Given
		badWriter := &errorWriter{}
		err := sut.SetOutput(badWriter)
		Expect(err).ToNot(HaveOccurred())

		sut.Step(demo.S("Test step"), demo.S("echo test"))

		// When
		err = sut.RunWithOptions(&opts)

		// Then
		Expect(err).To(HaveOccurred())
	})

	It("should fail when setup returns error", func() {
		// Given
		sut.Setup(func() error {
			return errSetupFailed
		})

		// When
		err := sut.RunWithOptions(&opts)

		// Then
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("setup failed"))
	})

	It("should fail when cleanup returns error", func() {
		// Given
		sut.Cleanup(func() error {
			return errCleanupFailed
		})

		// When
		err := sut.RunWithOptions(&opts)

		// Then
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("cleanup failed"))
	})

	It("should fail when step command fails", func() {
		// Given
		sut.Step(demo.S("Failing command"), demo.S("exit 42"))

		// When
		err := sut.RunWithOptions(&opts)

		// Then
		Expect(err).To(HaveOccurred())
	})

	It("should show title without description when description is empty", func() {
		// Given
		emptySut := demo.NewRun("Title Only")
		err := emptySut.SetOutput(out)
		Expect(err).ToNot(HaveOccurred())

		// When
		err = emptySut.RunWithOptions(&opts)

		// Then
		Expect(err).ToNot(HaveOccurred())
		Expect(out.String()).To(ContainSubstring("Title Only"))
	})

	It("should handle step with only description", func() {
		// Given
		sut.Step(demo.S("Description only step"), nil)

		// When
		err := sut.RunWithOptions(&opts)

		// Then
		Expect(err).ToNot(HaveOccurred())
		Expect(out.String()).To(ContainSubstring("Description only step"))
	})

	It("should handle step with multiline command", func() {
		// Given
		sut.Step(demo.S("Multiline test"), demo.S("echo line1", "echo line2"))

		// When
		err := sut.RunWithOptions(&opts)

		// Then
		Expect(err).ToNot(HaveOccurred())
		Expect(out.String()).To(ContainSubstring("echo line1"))
		Expect(out.String()).To(ContainSubstring("echo line2"))
	})

	It("should run with all flags from context", func() {
		// Given
		app := cli.NewApp()
		flagSet := &flag.FlagSet{}
		flagSet.Bool(demo.FlagAuto, true, "")
		flagSet.Bool(demo.FlagImmediate, true, "")
		flagSet.Bool(demo.FlagHideDescriptions, true, "")
		flagSet.Bool(demo.FlagDryRun, true, "")
		flagSet.Bool(demo.FlagNoColor, true, "")
		flagSet.Bool(demo.FlagContinueOnError, true, "")
		flagSet.Int(demo.FlagSkipSteps, 0, "")
		flagSet.String(demo.FlagShell, "sh", "")
		flagSet.Int(demo.FlagTypewriterSpeed, 20, "")

		ctx := cli.NewContext(app, flagSet, nil)
		sut.Step(demo.S("Test all flags"), demo.S("echo test"))

		// When
		err := sut.Run(ctx)

		// Then
		Expect(err).ToNot(HaveOccurred())
	})

	It("should run with breakpoint flag set but BreakPoint not called", func() {
		// Given
		app := cli.NewApp()
		flagSet := &flag.FlagSet{}
		flagSet.Bool(demo.FlagAuto, true, "")
		flagSet.Bool(demo.FlagImmediate, true, "")
		flagSet.Bool(demo.FlagBreakPoint, true, "")

		ctx := cli.NewContext(app, flagSet, nil)
		sut.Step(demo.S("Test"), demo.S("echo test"))

		// When
		err := sut.Run(ctx)

		// Then
		Expect(err).ToNot(HaveOccurred())
	})

	It("should properly handle empty title", func() {
		// Given
		emptySut := demo.NewRun("")
		err := emptySut.SetOutput(out)
		Expect(err).ToNot(HaveOccurred())

		// When
		err = emptySut.RunWithOptions(&opts)

		// Then
		Expect(err).ToNot(HaveOccurred())
	})
})

// errorWriter is a writer that always returns an error.
type errorWriter struct{}

func (e *errorWriter) Write(p []byte) (int, error) {
	return 0, errWriteError
}
