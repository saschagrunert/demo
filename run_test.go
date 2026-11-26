package demo_test

import (
	"context"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/fatih/color"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/types"
	"github.com/saschagrunert/demo"
)

var (
	errSetupFailed   = errors.New("setup failed")
	errCleanupFailed = errors.New("cleanup failed")
	errWriteError    = errors.New("write error")
)

var _ = Describe("Run", func() {
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

	It("should fail to set nil input", func() {
		// Given
		// When
		err := sut.SetInput(nil)

		// Then
		Expect(err).To(HaveOccurred())
	})

	It("should succeed to set input", func() {
		// Given
		input := strings.NewReader("\n")

		// When
		err := sut.SetInput(input)

		// Then
		Expect(err).ToNot(HaveOccurred())
	})

	It("should succeed to run in non-auto mode with input", func() {
		// Given
		// Each step calls waitOrSleep twice (before text, before execute),
		// so we need enough newlines for: 1 initial waitOrSleep + 1 echo + 1 execute waitOrSleep
		input := strings.NewReader("\n\n\n")
		err := sut.SetInput(input)
		Expect(err).ToNot(HaveOccurred())

		sut.Step(demo.S("Interactive step"), demo.S("echo hello"))

		manualOpts := demo.Options{
			Auto:      false,
			Immediate: true,
		}

		// When
		err = sut.RunWithOptions(&manualOpts)

		// Then
		Expect(err).ToNot(HaveOccurred())
		Expect(out.String()).To(ContainSubstring("echo hello"))
	})

	It("should succeed to run with setup and cleanup", func() {
		// Given
		setupCalled := false
		cleanupCalled := false

		sut.Setup(func(context.Context) error {
			setupCalled = true

			return nil
		})

		sut.Cleanup(func(context.Context) error {
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

		sut.Step(demo.S("Hidden description"), demo.S("echo $((40 + 2))"))

		// When
		err := sut.RunWithOptions(&opts)

		// Then
		Expect(err).ToNot(HaveOccurred())
		Expect(out.String()).ToNot(ContainSubstring("Hidden description"))
		Expect(out.String()).ToNot(ContainSubstring(description[0]))
		Expect(out.String()).To(ContainSubstring("> echo $((40 + 2))\n42\n"))
	})

	It("should succeed to run with dry-run", func() {
		// Given
		opts.DryRun = true

		marker := filepath.Join(GinkgoT().TempDir(), "marker")
		sut.Step(demo.S("Test step"), demo.S("touch", marker))

		// When
		err := sut.RunWithOptions(&opts)

		// Then
		Expect(err).ToNot(HaveOccurred())
		Expect(out.String()).To(ContainSubstring("> touch"))
		Expect(marker).ToNot(BeAnExistingFile())
	})

	It("should succeed to run with no-color", func() {
		// Given
		enableColor()

		opts.NoColor = true

		sut.Step(demo.S("Test step"), demo.S("echo test"))

		// When
		err := sut.RunWithOptions(&opts)

		// Then
		Expect(err).ToNot(HaveOccurred())
		Expect(out.String()).To(ContainSubstring("> echo test"))
		Expect(out.String()).ToNot(ContainSubstring("\x1b["))
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

		sut.Step(demo.S("Shell test"), demo.S(`echo "shell=$0"`))

		// When
		err := sut.RunWithOptions(&opts)

		// Then
		Expect(err).ToNot(HaveOccurred())
		Expect(out.String()).To(ContainSubstring("shell=sh\n"))
	})

	DescribeTable("should succeed to run with custom typewriter speed",
		func(speed int, matcher types.GomegaMatcher) {
			// Given
			sut.Step(demo.S("Speed test"), demo.S("echo test"))

			// When
			start := time.Now()
			err := sut.RunWithOptions(&demo.Options{Auto: true, TypewriterSpeed: speed})

			// Then
			Expect(err).ToNot(HaveOccurred())
			Expect(out.String()).To(ContainSubstring("# Speed test [1/1]:"))
			Expect(time.Since(start)).To(matcher)
		},
		// About 30 characters with a random delay below the speed each, so
		// with the default speed of 40ms this would take about 600ms.
		Entry("without delays", 1, BeNumerically("<", 200*time.Millisecond)),
		Entry("with up to 10ms delay", 10, BeNumerically(">", 20*time.Millisecond)),
	)

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

	It("should fail when setup returns error and still clean up", func() {
		// Given
		cleanedUp := false

		sut.Setup(func(context.Context) error {
			return errSetupFailed
		})
		sut.Cleanup(func(context.Context) error {
			cleanedUp = true

			return nil
		})
		sut.Step(demo.S("Not executed"), demo.S("echo step"))

		// When
		err := sut.RunWithOptions(&opts)

		// Then
		Expect(err).To(MatchError(errSetupFailed))
		Expect(cleanedUp).To(BeTrue())
		Expect(out.String()).ToNot(ContainSubstring("Not executed"))
	})

	It("should fail when cleanup returns error", func() {
		// Given
		sut.Cleanup(func(context.Context) error {
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

	It("should run with all flags from options", func() {
		// Given
		opts := demo.Options{
			Auto:             true,
			Immediate:        true,
			HideDescriptions: true,
			DryRun:           true,
			NoColor:          true,
			ContinueOnError:  true,
			SkipSteps:        0,
			Shell:            "sh",
			TypewriterSpeed:  20,
		}

		sut.Step(demo.S("Test all flags"), demo.S("echo test"))

		// When
		err := sut.RunWithOptions(&opts)

		// Then
		Expect(err).ToNot(HaveOccurred())
	})

	It("should run with breakpoint flag set but BreakPoint not called", func() {
		// Given
		opts := demo.Options{
			Auto:       true,
			Immediate:  true,
			BreakPoint: true,
		}

		sut.Step(demo.S("Test"), demo.S("echo test"))

		// When
		err := sut.RunWithOptions(&opts)

		// Then
		Expect(err).ToNot(HaveOccurred())
	})

	It("should succeed to run with SetWorkDir", func() {
		// Given
		dir := GinkgoT().TempDir()
		sut.SetWorkDir(dir)
		sut.Step(demo.S("Working dir test"), demo.S("pwd"))

		// When
		err := sut.RunWithOptions(&opts)

		// Then
		Expect(err).ToNot(HaveOccurred())
		Expect(out.String()).To(ContainSubstring("> pwd\n" + dir + "\n"))
	})

	It("should succeed to run with Chdir between steps", func() {
		// Given
		first, second := GinkgoT().TempDir(), GinkgoT().TempDir()
		sut.SetWorkDir(first)
		sut.Step(demo.S("First step"), demo.S("pwd"))
		sut.Chdir(second)
		sut.Step(demo.S("After chdir"), demo.S("pwd"))

		// When
		err := sut.RunWithOptions(&opts)

		// Then
		Expect(err).ToNot(HaveOccurred())
		Expect(out.String()).To(ContainSubstring("> pwd\n" + first + "\n"))
		Expect(out.String()).To(ContainSubstring("> pwd\n" + second + "\n"))
	})

	It("should restore the work dir after Chdir for the next execution", func() {
		// Given
		first, second := GinkgoT().TempDir(), GinkgoT().TempDir()
		sut.SetWorkDir(first)
		sut.Step(demo.S("Before chdir"), demo.S("pwd"))
		sut.Chdir(second)
		sut.Step(demo.S("After chdir"), demo.S("pwd"))
		Expect(sut.RunWithOptions(&opts)).To(Succeed())
		out.Reset()

		// When
		err := sut.RunWithOptions(&opts)

		// Then
		Expect(err).ToNot(HaveOccurred())
		Expect(out.String()).To(ContainSubstring("> pwd\n" + first + "\n"))
		Expect(out.String()).To(ContainSubstring("> pwd\n" + second + "\n"))
	})

	It("should apply Chdir even when steps are skipped", func() {
		// Given
		opts.SkipSteps = 1

		dir := GinkgoT().TempDir()

		sut.Step(demo.S("Skipped step"), demo.S("echo skipped"))
		sut.Chdir(dir)
		sut.Step(demo.S("After chdir"), demo.S("pwd"))

		// When
		err := sut.RunWithOptions(&opts)

		// Then
		Expect(err).ToNot(HaveOccurred())
		Expect(out.String()).ToNot(ContainSubstring("Skipped step"))
		Expect(out.String()).To(ContainSubstring("> pwd\n" + dir + "\n"))
	})

	It("should not count Chdir in step numbering", func() {
		// Given
		sut.Step(demo.S("First"), demo.S("echo one"))
		sut.Chdir("/tmp")
		sut.Step(demo.S("Second"), demo.S("pwd"))

		// When
		err := sut.RunWithOptions(&opts)

		// Then
		Expect(err).ToNot(HaveOccurred())
		Expect(out.String()).To(ContainSubstring("[1/2]"))
		Expect(out.String()).To(ContainSubstring("[2/2]"))
	})

	It("should succeed to run with Chdir and dry-run", func() {
		// Given
		opts.DryRun = true

		dir := GinkgoT().TempDir()
		sut.Chdir(dir)
		sut.Step(demo.S("Dry run chdir"), demo.S("pwd"))

		// When
		err := sut.RunWithOptions(&opts)

		// Then
		Expect(err).ToNot(HaveOccurred())
		Expect(out.String()).To(ContainSubstring("> cd " + dir + "\n"))
		Expect(out.String()).To(HaveSuffix("> pwd\n"))
	})

	It("should succeed to run with SetEnv", func() {
		// Given
		sut.SetEnv("DEMO_TEST_VAR=hello123")
		sut.Step(demo.S("Env test"), demo.S("echo $DEMO_TEST_VAR"))

		// When
		err := sut.RunWithOptions(&opts)

		// Then
		Expect(err).ToNot(HaveOccurred())
		Expect(out.String()).To(ContainSubstring("hello123"))
	})

	It("should succeed to run with multiple SetEnv calls", func() {
		// Given
		sut.SetEnv("VAR1=first")
		sut.SetEnv("VAR2=second")
		sut.Step(demo.S("Multi env"), demo.S("echo $VAR1 $VAR2"))

		// When
		err := sut.RunWithOptions(&opts)

		// Then
		Expect(err).ToNot(HaveOccurred())
		Expect(out.String()).To(ContainSubstring("first second"))
	})

	It("should display Chdir as a command", func() {
		// Given
		sut.Chdir("/tmp")
		sut.Step(demo.S("After chdir"), demo.S("pwd"))

		// When
		err := sut.RunWithOptions(&opts)

		// Then
		Expect(err).ToNot(HaveOccurred())
		Expect(out.String()).To(ContainSubstring("cd /tmp"))
	})

	It("should hide Chdir display when descriptions are hidden", func() {
		// Given
		opts.HideDescriptions = true

		sut.Chdir("/tmp")
		sut.Step(demo.S("After chdir"), demo.S("pwd"))

		// When
		err := sut.RunWithOptions(&opts)

		// Then
		Expect(err).ToNot(HaveOccurred())
		Expect(out.String()).ToNot(ContainSubstring("cd /tmp"))
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

	It("should clamp negative SkipSteps to zero", func() {
		sut.Step(demo.S("First"), demo.S("echo first"))
		sut.Step(demo.S("Second"), demo.S("echo second"))

		opts.SkipSteps = -5

		err := sut.RunWithOptions(&opts)
		Expect(err).ToNot(HaveOccurred())
		Expect(out.String()).To(ContainSubstring("First"))
		Expect(out.String()).To(ContainSubstring("Second"))
	})

	It("should use default TypewriterSpeed for non-positive values", func() {
		opts.TypewriterSpeed = -10
		opts.Immediate = false
		opts.Auto = true
		opts.AutoTimeout = 0

		sut.Step(demo.S("Speed test"), demo.S("echo test"))

		err := sut.RunWithOptions(&opts)
		Expect(err).ToNot(HaveOccurred())
	})

	It("should return both errors when run and cleanup fail", func() {
		sut.Step(demo.S("Failing step"), demo.S("exit 1"))
		sut.Cleanup(func(context.Context) error {
			return errCleanupFailed
		})

		err := sut.RunWithOptions(&opts)
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("exit"))
		Expect(err.Error()).To(ContainSubstring("cleanup failed"))
	})

	It("should stop on context cancellation", func() {
		ctx, cancel := context.WithCancel(context.Background())
		cancel()

		cancelledOpts := opts
		cancelledOpts.Context = ctx

		hooks := 0

		sut.Setup(func(context.Context) error {
			hooks++

			return nil
		})
		sut.Cleanup(func(context.Context) error {
			hooks++

			return nil
		})
		sut.Step(demo.S("Should not run"), demo.S("echo unreachable"))

		err := sut.RunWithOptions(&cancelledOpts)
		Expect(err).To(MatchError(context.Canceled))
		Expect(out.String()).To(BeEmpty())
		Expect(hooks).To(BeZero())
	})

	It("should call cleanup even when a step fails", func() {
		cleanupCalled := false

		sut.Cleanup(func(context.Context) error {
			cleanupCalled = true

			return nil
		})

		sut.Step(demo.S("Failing"), demo.S("exit 1"))

		err := sut.RunWithOptions(&opts)
		Expect(err).To(HaveOccurred())
		Expect(cleanupCalled).To(BeTrue())
	})

	It("should stop auto-mode sleep on context cancellation", func() {
		ctx, cancel := context.WithCancel(context.Background())

		cancelOpts := demo.Options{
			Auto:        true,
			AutoTimeout: 10 * time.Second,
			Context:     ctx,
			Immediate:   true,
		}

		sut.Step(demo.S("Slow step"), demo.S("echo test"))

		go func() {
			time.Sleep(50 * time.Millisecond)
			cancel()
		}()

		err := sut.RunWithOptions(&cancelOpts)
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("context"))
	})

	It("should not wait at breakpoints when they are disabled", func() {
		// Given
		Expect(sut.SetInput(strings.NewReader("\n\n\n\n"))).To(Succeed())

		sut.Step(demo.S("First"), demo.S("echo one"))
		sut.BreakPoint()
		sut.Step(demo.S("Second"), demo.S("echo two"))

		// When
		err := sut.RunWithOptions(&demo.Options{Immediate: true, NoColor: true})

		// Then
		Expect(err).ToNot(HaveOccurred())
		Expect(out.String()).To(ContainSubstring("First [1/2]"))
		Expect(out.String()).To(ContainSubstring("Second [2/2]"))
		Expect(out.String()).ToNot(ContainSubstring("bp"))
	})

	It("should wait at breakpoints when they are enabled", func() {
		// Given
		Expect(sut.SetInput(strings.NewReader("\n\n\n\n\n"))).To(Succeed())

		sut.Step(demo.S("First"), demo.S("echo one"))
		sut.BreakPoint()
		sut.Step(demo.S("Second"), demo.S("echo two"))

		// When
		err := sut.RunWithOptions(&demo.Options{Immediate: true, NoColor: true, BreakPoint: true})

		// Then
		Expect(err).ToNot(HaveOccurred())
		Expect(out.String()).To(ContainSubstring("bp"))
		Expect(out.String()).To(ContainSubstring("Second [2/2]"))
	})

	It("should not count breakpoints for skipped steps", func() {
		// Given
		opts.SkipSteps = 1

		sut.Step(demo.S("First"), demo.S("echo one"))
		sut.BreakPoint()
		sut.Step(demo.S("Second"), demo.S("echo two"))

		// When
		err := sut.RunWithOptions(&opts)

		// Then
		Expect(err).ToNot(HaveOccurred())
		Expect(out.String()).ToNot(ContainSubstring("First"))
		Expect(out.String()).To(ContainSubstring("Second [2/2]"))
	})

	It("should accept a last input line without a newline", func() {
		// Given
		Expect(sut.SetInput(strings.NewReader("\nx"))).To(Succeed())
		sut.Step(demo.S("Step"), demo.S("echo hello"))

		// When
		err := sut.RunWithOptions(&demo.Options{Immediate: true})

		// Then
		Expect(err).ToNot(HaveOccurred())
	})

	It("should stop waiting for input on context cancellation", func() {
		// Given
		reader, writer := io.Pipe()
		defer writer.Close()

		Expect(sut.SetInput(reader)).To(Succeed())
		sut.Step(demo.S("Step"), demo.S("echo hello"))

		ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
		defer cancel()

		// When
		err := sut.RunWithOptions(&demo.Options{Immediate: true, Context: ctx})

		// Then
		Expect(err).To(MatchError(context.DeadlineExceeded))
	})

	It("should report an interrupted command as cancellation", func() {
		// Given
		ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
		defer cancel()

		sut.StepCanFail(demo.S("Slow step"), demo.S("sleep 10"))

		// When
		start := time.Now()
		err := sut.RunWithOptions(&demo.Options{Auto: true, Immediate: true, Context: ctx})

		// Then
		Expect(err).To(MatchError(context.DeadlineExceeded))
		Expect(time.Since(start)).To(BeNumerically("<", 5*time.Second))
	})

	It("should not wait for background processes holding the output", func() {
		// Given
		sut.Step(nil, demo.S("sleep 10 &"))

		// When
		start := time.Now()
		err := sut.RunWithOptions(&opts)

		// Then
		Expect(err).ToNot(HaveOccurred())
		Expect(time.Since(start)).To(BeNumerically("<", 5*time.Second))
	})

	It("should stop the typewriter animation on context cancellation", func() {
		// Given
		ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
		defer cancel()

		sut.Step(demo.S(strings.Repeat("x", 100)), nil)

		// When
		start := time.Now()
		err := sut.RunWithOptions(&demo.Options{
			Auto:            true,
			Context:         ctx,
			TypewriterSpeed: 200,
		})

		// Then
		Expect(err).To(MatchError(context.DeadlineExceeded))
		Expect(time.Since(start)).To(BeNumerically("<", 5*time.Second))
		Expect(out.String()).ToNot(ContainSubstring("\x1b"))
	})

	It("should not wait at breakpoints within the skipped steps", func() {
		// Given
		Expect(sut.SetInput(strings.NewReader("\n\n"))).To(Succeed())

		sut.Step(demo.S("First"), demo.S("echo one"))
		sut.BreakPoint()
		sut.Step(demo.S("Second"), demo.S("echo two"))

		// When
		err := sut.RunWithOptions(&demo.Options{
			Immediate:  true,
			NoColor:    true,
			BreakPoint: true,
			SkipSteps:  1,
		})

		// Then
		Expect(err).ToNot(HaveOccurred())
		Expect(out.String()).ToNot(ContainSubstring("bp"))
		Expect(out.String()).To(ContainSubstring("Second [2/2]"))
	})

	It("should leave piped input after the newline to the step commands", func() {
		// Given
		reader, writer, err := os.Pipe()
		Expect(err).ToNot(HaveOccurred())

		DeferCleanup(reader.Close)

		_, err = writer.WriteString("\n\nhello\n")
		Expect(err).ToNot(HaveOccurred())
		Expect(writer.Close()).To(Succeed())

		Expect(sut.SetInput(reader)).To(Succeed())
		sut.Step(nil, demo.S("read x; echo got=$x"))

		// When
		err = sut.RunWithOptions(&demo.Options{Immediate: true})

		// Then
		Expect(err).ToNot(HaveOccurred())
		Expect(out.String()).To(ContainSubstring("got=hello\n"))
	})

	It("should not modify reused options", func() {
		// Given
		origNoColor := color.NoColor
		color.NoColor = false

		defer func() { color.NoColor = origNoColor }()

		reused := demo.Options{Auto: true, Immediate: true}
		Expect(sut.RunWithOptions(&reused)).To(Succeed())
		Expect(out.String()).To(ContainSubstring("\x1b["))
		Expect(reused).To(Equal(demo.Options{Auto: true, Immediate: true}))

		out.Reset()

		reused.NoColor = true

		// When
		err := sut.RunWithOptions(&reused)

		// Then
		Expect(err).ToNot(HaveOccurred())
		Expect(out.String()).ToNot(ContainSubstring("\x1b["))
	})

	It("should run with nil options", func() {
		// Given
		run := demo.NewRun("Nil options")
		Expect(run.SetOutput(out)).To(Succeed())

		// When
		err := run.RunWithOptions(nil)

		// Then
		Expect(err).ToNot(HaveOccurred())
		Expect(out.String()).To(ContainSubstring("Nil options"))
	})

	It("should fail with an invalid theme", func() {
		// Given
		opts.Theme = "bogus"

		// When
		err := sut.RunWithOptions(&opts)

		// Then
		Expect(err).To(MatchError(ContainSubstring(`invalid theme "bogus"`)))
	})

	It("should pass the run context to setup and cleanup", func() {
		// Given
		type ctxKey struct{}

		ctx := context.WithValue(context.Background(), ctxKey{}, "value")

		var setupValue, cleanupValue any

		sut.Setup(func(ctx context.Context) error {
			setupValue = ctx.Value(ctxKey{})

			return nil
		})
		sut.Cleanup(func(ctx context.Context) error {
			cleanupValue = ctx.Value(ctxKey{})

			return nil
		})

		// When
		err := sut.RunWithOptions(&demo.Options{Auto: true, Immediate: true, Context: ctx})

		// Then
		Expect(err).ToNot(HaveOccurred())
		Expect(setupValue).To(Equal("value"))
		Expect(cleanupValue).To(Equal("value"))
	})

	It("should run the cleanup with a usable context when interrupted", func() {
		// Given
		ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
		defer cancel()

		var cleanupErr error

		sut.Step(nil, demo.S("sleep 10"))
		sut.Cleanup(func(ctx context.Context) error {
			cleanupErr = demo.EnsureWithContext(ctx, "true")

			return nil
		})

		// When
		err := sut.RunWithOptions(&demo.Options{Auto: true, Immediate: true, Context: ctx})

		// Then
		Expect(err).To(MatchError(context.DeadlineExceeded))
		Expect(cleanupErr).ToNot(HaveOccurred())
	})

	It("should use the auto theme for empty options", func() {
		// Given
		GinkgoT().Setenv("COLORFGBG", "15;7")
		enableColor()

		// When
		err := sut.RunWithOptions(&demo.Options{Auto: true, Immediate: true})

		// Then
		Expect(err).ToNot(HaveOccurred())
		Expect(out.String()).To(HavePrefix(blueTitle + title))
	})

	It("should pass a file input to the step commands", func() {
		// Given
		reader, writer, err := os.Pipe()
		Expect(err).ToNot(HaveOccurred())

		DeferCleanup(reader.Close)

		_, err = writer.WriteString("piped input\n")
		Expect(err).ToNot(HaveOccurred())
		Expect(writer.Close()).To(Succeed())

		Expect(sut.SetInput(reader)).To(Succeed())
		sut.Step(nil, demo.S("cat"))

		// When
		err = sut.RunWithOptions(&opts)

		// Then
		Expect(err).ToNot(HaveOccurred())
		Expect(out.String()).To(ContainSubstring("> cat\npiped input\n"))
	})
})

// errorWriter is a writer that always returns an error.
type errorWriter struct{}

func (e *errorWriter) Write(p []byte) (int, error) {
	return 0, errWriteError
}
