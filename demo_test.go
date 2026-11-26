package demo_test

import (
	"context"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/fatih/color"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gbytes"
	"github.com/onsi/gomega/gexec"
	"github.com/onsi/gomega/types"
	"github.com/saschagrunert/demo"
	"github.com/urfave/cli/v3"
)

const (
	appName         = "demo"
	autoFlag        = "--auto"
	autoTimeoutFlag = "--auto-timeout=0"
	immediateFlag   = "--immediate"
	run1Flag        = "--run1"
	run2Flag        = "--run2"
	allFlag         = "--all"

	blueTitle    = "\x1b[34m"
	cyanTitle    = "\x1b[36m"
	magentaTitle = "\x1b[35m"
)

// newTestRun creates a run that writes into the returned builder.
func newTestRun(title string, description ...string) (*demo.Run, *strings.Builder) {
	run := demo.NewRun(title, description...)
	out := &strings.Builder{}
	Expect(run.SetOutput(out)).To(Succeed())

	return run, out
}

// newTestDemo creates a demo that writes its help and errors to the
// GinkgoWriter.
func newTestDemo(options ...demo.Option) *demo.Demo {
	sut := demo.New(options...)
	sut.Writer = GinkgoWriter
	sut.ErrWriter = GinkgoWriter

	return sut
}

// runDemo runs the demo with the provided flags and without any delays.
func runDemo(sut *demo.Demo, flags ...string) error {
	args := append([]string{appName, autoFlag, autoTimeoutFlag, immediateFlag}, flags...)

	return sut.RunContext(context.Background(), args) //nolint:wrapcheck // test helper
}

// enableColor enables colored output for the current spec.
func enableColor() {
	orig := color.NoColor
	color.NoColor = false

	DeferCleanup(func() { color.NoColor = orig })
}

// expectTitleColor runs a demo with a single run and checks the title color.
func expectTitleColor(expected string, flags ...string) {
	enableColor()

	run, out := newTestRun("Run 1")
	sut := newTestDemo()
	sut.Add(run, "run1", "first run")

	Expect(runDemo(sut, append(flags, run1Flag)...)).To(Succeed())
	Expect(out.String()).To(HavePrefix(expected + "Run 1"))
}

// expectCustomPalette runs a demo with custom palettes and checks that the
// light palette is used.
func expectCustomPalette(flags ...string) {
	enableColor()

	run, out := newTestRun("Run 1", "desc")
	sut := newTestDemo(demo.WithPalettes(
		demo.Palette{Title: []color.Attribute{color.FgRed}},
		demo.Palette{Title: []color.Attribute{color.FgMagenta}},
	))
	sut.Add(run, "run1", "first run")

	Expect(runDemo(sut, append(flags, run1Flag)...)).To(Succeed())
	Expect(out.String()).To(HavePrefix(magentaTitle + "Run 1"))
	// The description falls back to the default light palette.
	Expect(out.String()).To(ContainSubstring("\x1b[94mdesc"))
}

var _ = Describe("Demo", func() {
	DescribeTable("should show the help if no run is selected",
		func(flags ...string) {
			out := &strings.Builder{}
			run, runOut := newTestRun("Run 1")
			sut := newTestDemo()
			sut.Writer = out
			sut.Add(run, "run1", "first run")

			Expect(runDemo(sut, flags...)).To(Succeed())
			Expect(out.String()).To(ContainSubstring("GLOBAL OPTIONS"))
			Expect(runOut.String()).To(BeEmpty())
		},
		Entry("without flags"),
		Entry("in continuous mode", "--continuously"),
	)

	It("should run the example with setup and cleanup", func() {
		run, out := newTestRun("Title", "Some additional", "multiline description")
		run.Step(demo.S("This is a possible", "description"), demo.S("echo hello world"))
		run.Step(nil, demo.S("echo without", "description"))
		run.Step(demo.S("Just a description without a command"), nil)

		var calls []string

		sut := newTestDemo()
		sut.Add(run, "title", "description")
		sut.Setup(func(context.Context, *cli.Command) error {
			calls = append(calls, "setup")

			return nil
		})
		sut.Cleanup(func(context.Context, *cli.Command) error {
			calls = append(calls, "cleanup")

			return nil
		})

		Expect(runDemo(sut, "--title")).To(Succeed())
		Expect(calls).To(Equal([]string{"setup", "cleanup"}))
		Expect(out.String()).To(ContainSubstring("Some additional\nmultiline description\n"))
		Expect(out.String()).To(ContainSubstring("# description [1/3]:"))
		Expect(out.String()).To(ContainSubstring("> echo hello world\nhello world\n"))
		Expect(out.String()).To(ContainSubstring("without description\n"))
		Expect(out.String()).To(ContainSubstring("# Just a description without a command [3/3]"))
	})

	It("should run all runs in order with --all", func() {
		out := &strings.Builder{}
		run1 := demo.NewRun("Run 1")
		run2 := demo.NewRun("Run 2")

		Expect(run1.SetOutput(out)).To(Succeed())
		Expect(run2.SetOutput(out)).To(Succeed())

		sut := newTestDemo()
		sut.Add(run1, "run1", "first run")
		sut.Add(run2, "run2", "second run")

		Expect(runDemo(sut, allFlag)).To(Succeed())
		Expect(out.String()).To(MatchRegexp(`(?s)Run 1.*Run 2`))
	})

	DescribeTable("should run only the selected runs",
		func(flags []string, run1Matcher, run2Matcher types.GomegaMatcher) {
			run1, out1 := newTestRun("Run 1")
			run2, out2 := newTestRun("Run 2")

			sut := newTestDemo()
			sut.Add(run1, "run1", "first run")
			sut.Add(run2, "run2", "second run")

			Expect(sut.RunContext(context.Background(), append([]string{appName}, flags...))).
				To(Succeed())
			Expect(out1.String()).To(run1Matcher)
			Expect(out2.String()).To(run2Matcher)
		},
		Entry(
			"by flag",
			[]string{
				run2Flag,
				autoFlag,
				autoTimeoutFlag,
				immediateFlag,
			},
			BeEmpty(),
			HavePrefix("Run 2"),
		),
		Entry("with short options",
			[]string{"-a", "-i", "-t=0", run1Flag}, HavePrefix("Run 1"), BeEmpty()),
		Entry(
			"with all short flags",
			[]string{
				"-l",
				"-a",
				"-t=0",
				"-i",
				"-d",
				"-s=0",
			},
			HavePrefix("Run 1"),
			HavePrefix("Run 2"),
		),
	)

	It("should hide descriptions with the short flag", func() {
		run, out := newTestRun("Run 1", "hidden description")
		run.Step(demo.S("hidden step"), demo.S("echo $((40 + 2))"))

		sut := newTestDemo()
		sut.Add(run, "run1", "first run")

		Expect(runDemo(sut, "-d", run1Flag)).To(Succeed())
		Expect(out.String()).ToNot(ContainSubstring("hidden"))
		Expect(out.String()).To(ContainSubstring("> echo $((40 + 2))\n42\n"))
	})

	It("should run setup and cleanup for every run", func() {
		sut := newTestDemo()

		for _, name := range []string{"run1", "run2", "run3"} {
			run, _ := newTestRun(name)
			sut.Add(run, name, name)
		}

		setups, cleanups := 0, 0

		sut.Setup(func(context.Context, *cli.Command) error {
			setups++

			return nil
		})
		sut.Cleanup(func(context.Context, *cli.Command) error {
			cleanups++

			return nil
		})

		Expect(runDemo(sut, allFlag)).To(Succeed())
		Expect(setups).To(Equal(3))
		Expect(cleanups).To(Equal(3))
	})

	It("should repeat the runs in continuous mode until cancelled", func() {
		run, out := newTestRun("Run 1")
		ctx, cancel := context.WithCancel(context.Background())

		defer cancel()

		iterations := 0
		sut := newTestDemo()
		sut.Add(run, "run1", "first run")
		sut.Setup(func(context.Context, *cli.Command) error {
			iterations++
			if iterations == 3 {
				cancel()
			}

			return nil
		})

		err := sut.RunContext(ctx, []string{
			appName, "--continuously", run1Flag, autoFlag, autoTimeoutFlag, immediateFlag,
		})

		Expect(err).ToNot(HaveOccurred())
		Expect(iterations).To(Equal(3))
		Expect(strings.Count(out.String(), "Run 1")).To(Equal(2))
	})

	It("should stop the continuous mode on a failing run", func() {
		run, _ := newTestRun("Run 1")
		run.Step(nil, demo.S("false"))

		sut := newTestDemo()
		sut.Add(run, "run1", "first run")

		err := runDemo(sut, "--continuously", run1Flag)
		Expect(err).To(MatchError(ContainSubstring(`step command "false" failed`)))
	})

	It("should run with light theme", func() {
		expectTitleColor(blueTitle, "--theme=light")
	})

	It("should run with case-insensitive theme value", func() {
		expectTitleColor(blueTitle, "--theme=LIGHT")
	})

	DescribeTable("should detect only light ANSI backgrounds in auto theme",
		func(background, expectedColor string) {
			GinkgoT().Setenv("COLORFGBG", "15;"+background)
			expectTitleColor(expectedColor, "--theme=auto")
		},
		Entry("light gray background", "7", blueTitle),
		Entry("bright ANSI background", "15", blueTitle),
		Entry("bright black background", "8", cyanTitle),
		Entry("first extended palette index", "16", cyanTitle),
		Entry("near-black extended palette index", "234", cyanTitle),
	)

	It("should read run options after setup", func() {
		enableColor()

		run, out := newTestRun("Run 1")
		sut := newTestDemo()
		sut.Add(run, "run1", "first run")
		sut.Setup(func(_ context.Context, cmd *cli.Command) error {
			return cmd.Set(demo.FlagTheme, string(demo.ThemeLight))
		})

		Expect(runDemo(sut, run1Flag)).To(Succeed())
		Expect(out.String()).To(HavePrefix(blueTitle + "Run 1"))
	})

	It("should fail when theme is invalid", func() {
		run, out := newTestRun("Run 1")
		sut := newTestDemo()
		sut.Add(run, "run1", "first run")

		err := runDemo(sut, run1Flag, "--theme=ligth")
		Expect(err).To(MatchError(ContainSubstring(`invalid theme "ligth"`)))
		Expect(out.String()).To(BeEmpty())
	})

	It("should run with paired palettes configured via New options", func() {
		expectCustomPalette("--theme=light")
	})

	It("should use custom palettes for auto theme with no explicit theme flag", func() {
		GinkgoT().Setenv("COLORFGBG", "15;7")
		expectCustomPalette()
	})

	It("should run the cleanups in order when interrupted by a signal", func() {
		binary, err := gexec.Build("github.com/saschagrunert/demo/testdata/interrupt")
		Expect(err).ToNot(HaveOccurred())

		DeferCleanup(gexec.CleanupBuildArtifacts)

		cmd := exec.Command(binary, "--run", autoFlag, autoTimeoutFlag, immediateFlag)
		session, err := gexec.Start(cmd, GinkgoWriter, GinkgoWriter)
		Expect(err).ToNot(HaveOccurred())

		// A freshly built binary can take a while to start, for example on macOS.
		Eventually(session, 10*time.Second).Should(gbytes.Say("> sleep 10"))
		session.Terminate()

		Eventually(session, 10*time.Second).Should(gexec.Exit(0))
		Expect(session.Out).To(gbytes.Say("run cleanup"))
		Expect(session.Out).To(gbytes.Say("demo cleanup"))
	})

	It("should report a failing cleanup when interrupted by a signal", func() {
		binary, err := gexec.Build("github.com/saschagrunert/demo/testdata/interrupt")
		Expect(err).ToNot(HaveOccurred())

		DeferCleanup(gexec.CleanupBuildArtifacts)

		cmd := exec.Command(binary, "--run", autoFlag, autoTimeoutFlag, immediateFlag)

		cmd.Env = append(os.Environ(), "FAIL_RUN_CLEANUP=1")
		session, err := gexec.Start(cmd, GinkgoWriter, GinkgoWriter)
		Expect(err).ToNot(HaveOccurred())

		Eventually(session, 10*time.Second).Should(gbytes.Say("> sleep 10"))
		session.Terminate()

		Eventually(session, 10*time.Second).Should(gexec.Exit(1))
		Expect(session.Out).To(gbytes.Say("demo cleanup"))
		Expect(session.Err).To(gbytes.Say("run cleanup failed"))
	})

	It("should interrupt a running step and clean up when the context gets cancelled", func() {
		// The buffer is safe to read while the run writes to it.
		out := gbytes.NewBuffer()
		run := demo.NewRun("Run 1")
		Expect(run.SetOutput(out)).To(Succeed())
		run.Step(nil, demo.S("sleep 10"))

		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		go func() {
			defer GinkgoRecover()

			Eventually(out).Should(gbytes.Say("> sleep 10"))
			cancel()
		}()

		var cleanupErr error

		start := time.Now()
		sut := newTestDemo()
		sut.Add(run, "run1", "first run")
		sut.Cleanup(func(ctx context.Context, _ *cli.Command) error {
			cleanupErr = ctx.Err()

			return nil
		})

		err := sut.RunContext(
			ctx,
			[]string{appName, run1Flag, autoFlag, autoTimeoutFlag, immediateFlag},
		)

		Expect(err).ToNot(HaveOccurred())
		Expect(cleanupErr).ToNot(HaveOccurred())
		Expect(time.Since(start)).To(BeNumerically("<", 5*time.Second))
	})

	DescribeTable("should panic when a run name conflicts with a flag",
		func(name string) {
			sut := newTestDemo()

			Expect(func() { sut.Add(demo.NewRun("Run"), name, "conflict") }).
				To(PanicWith(ContainSubstring(name)))
		},
		Entry("built-in flag", "auto"),
		Entry("built-in alias", "a"),
		Entry("help flag", "help"),
	)

	It("should panic when a run name conflicts with the version flag", func() {
		sut := newTestDemo()
		sut.Version = "1.0.0"

		Expect(func() { sut.Add(demo.NewRun("Run"), "version", "conflict") }).
			To(PanicWith(ContainSubstring("version")))

		sut.HideVersion = true

		Expect(func() { sut.Add(demo.NewRun("Run"), "version", "no conflict") }).ToNot(Panic())
	})

	It("should run the demo cleanup when the setup fails", func() {
		run, out := newTestRun("Run 1")

		cleanedUp := false
		sut := newTestDemo()
		sut.Add(run, "run1", "first run")
		sut.Setup(func(context.Context, *cli.Command) error {
			return errSetupFailed
		})
		sut.Cleanup(func(context.Context, *cli.Command) error {
			cleanedUp = true

			return nil
		})

		Expect(runDemo(sut, run1Flag)).To(MatchError(errSetupFailed))
		Expect(cleanedUp).To(BeTrue())
		Expect(out.String()).To(BeEmpty())
	})
})
