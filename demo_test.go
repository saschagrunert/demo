package demo_test

import (
	"context"
	"os"
	"os/exec"
	"strings"
	"sync"
	"time"

	"github.com/fatih/color"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gbytes"
	"github.com/onsi/gomega/gexec"
	"github.com/saschagrunert/demo"
	"github.com/urfave/cli/v3"
)

// osArgsMu protects os.Args from concurrent test access.
//
//nolint:gochecknoglobals // required for test synchronization
var osArgsMu sync.Mutex

const (
	appName         = "demo"
	autoFlag        = "--auto"
	autoTimeoutFlag = "--auto-timeout=0"
	immediateFlag   = "--immediate"
	run1Flag        = "--run1"
)

func withArgs(args []string, fn func()) {
	osArgsMu.Lock()

	defer osArgsMu.Unlock()

	orig := os.Args

	defer func() { os.Args = orig }()

	os.Args = args

	fn()
}

// withColorFGBG sets the COLORFGBG environment variable for the current spec.
func withColorFGBG(value string, fn func()) {
	GinkgoT().Setenv("COLORFGBG", value)
	fn()
}

func runWithCustomPalettes(args []string) {
	withArgs(args, func() {
		origNoColor := color.NoColor
		color.NoColor = false

		defer func() { color.NoColor = origNoColor }()

		run1 := demo.NewRun("Run 1", "desc")
		out := &strings.Builder{}
		Expect(run1.SetOutput(out)).To(Succeed())

		sut := demo.New(
			demo.WithPalettes(
				demo.Palette{
					Title:       []color.Attribute{color.FgRed},
					Description: []color.Attribute{color.FgRed},
					Command:     []color.Attribute{color.FgRed},
				},
				demo.Palette{
					Title:       []color.Attribute{color.FgMagenta},
					Description: []color.Attribute{color.FgMagenta},
					Command:     []color.Attribute{color.FgMagenta},
				},
			),
		)
		sut.Add(run1, "run1", "first run")

		Expect(sut.RunE()).To(Succeed())
		Expect(out.String()).To(ContainSubstring("\x1b[35m"))
	})
}

func runWithExpectedTitleColor(args []string, expectedColorCode string) {
	withArgs(args, func() {
		origNoColor := color.NoColor
		color.NoColor = false

		defer func() { color.NoColor = origNoColor }()

		run1 := demo.NewRun("Run 1")
		out := &strings.Builder{}
		Expect(run1.SetOutput(out)).To(Succeed())

		sut := demo.New()
		sut.Add(run1, "run1", "first run")

		Expect(sut.RunE()).To(Succeed())
		Expect(out.String()).To(ContainSubstring(expectedColorCode))
	})
}

var _ = Describe("Demo", func() {
	It("should succeed to run", func() {
		withArgs([]string{appName}, func() {
			sut := demo.New()
			Expect(sut.RunE()).To(Succeed())
		})
	})

	It("should succeed to run with example", func() {
		withArgs([]string{
			appName, autoFlag, autoTimeoutFlag, immediateFlag, "--title",
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
			appName, "--all", autoFlag, autoTimeoutFlag, immediateFlag,
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
			appName, "--run2", autoFlag, autoTimeoutFlag, immediateFlag,
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
			appName, run1Flag, autoFlag, autoTimeoutFlag, immediateFlag,
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
			appName, "-a", "-i", "--run1",
		}, func() {
			run1 := demo.NewRun("Run 1")
			sut := demo.New()
			sut.Add(run1, "run1", "first run")

			Expect(sut.RunE()).To(Succeed())
		})
	})

	It("should run with all short flags", func() {
		withArgs([]string{
			appName, "-l", "-a", "-t=0", "-i", "-d", "-s=0",
		}, func() {
			run1 := demo.NewRun("Run 1")
			run2 := demo.NewRun("Run 2")

			sut := demo.New()
			sut.Add(run1, "run1", "first run")
			sut.Add(run2, "run2", "second run")

			Expect(sut.RunE()).To(Succeed())
		})
	})

	It("should run with light theme", func() {
		runWithExpectedTitleColor([]string{
			appName, run1Flag, "--theme=light", autoFlag, autoTimeoutFlag, immediateFlag,
		}, "\x1b[34m")
	})

	It("should run with auto theme", func() {
		withColorFGBG("15;7", func() {
			runWithExpectedTitleColor([]string{
				appName, run1Flag, "--theme=auto", autoFlag, autoTimeoutFlag, immediateFlag,
			}, "\x1b[34m")
		})
	})

	DescribeTable("should detect only light ANSI backgrounds",
		func(background, expectedColor string) {
			withColorFGBG("15;"+background, func() {
				runWithExpectedTitleColor([]string{
					appName, run1Flag, "--theme=auto", autoFlag, autoTimeoutFlag, immediateFlag,
				}, expectedColor)
			})
		},
		Entry("bright ANSI background", "15", "\x1b[34m"),
		Entry("first extended palette index", "16", "\x1b[36m"),
		Entry("near-black extended palette index", "234", "\x1b[36m"),
	)

	It("should read run options after setup", func() {
		withArgs([]string{
			appName, run1Flag, autoFlag, autoTimeoutFlag, immediateFlag,
		}, func() {
			origNoColor := color.NoColor
			color.NoColor = false

			defer func() { color.NoColor = origNoColor }()

			run := demo.NewRun("Run 1")
			out := &strings.Builder{}
			Expect(run.SetOutput(out)).To(Succeed())

			sut := demo.New()
			sut.Add(run, "run1", "first run")
			sut.Setup(func(_ context.Context, cmd *cli.Command) error {
				return cmd.Set(demo.FlagTheme, string(demo.ThemeLight))
			})

			Expect(sut.RunE()).To(Succeed())
			Expect(out.String()).To(ContainSubstring("\x1b[34m"))
		})
	})

	It("should fail when theme is invalid", func() {
		withArgs([]string{
			appName, run1Flag, "--theme=ligth", autoFlag, autoTimeoutFlag, immediateFlag,
		}, func() {
			run1 := demo.NewRun("Run 1")
			sut := demo.New()
			sut.Add(run1, "run1", "first run")

			err := sut.RunE()
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("invalid theme"))
		})
	})

	It("should run with case-insensitive theme value", func() {
		runWithExpectedTitleColor([]string{
			appName, run1Flag, "--theme=LIGHT", autoFlag, autoTimeoutFlag, immediateFlag,
		}, "\x1b[34m")
	})

	It("should run with paired palettes configured via New options", func() {
		runWithCustomPalettes([]string{
			appName, run1Flag, "--theme=light", autoFlag, autoTimeoutFlag, immediateFlag,
		})
	})

	It("should use custom palettes for auto theme with no explicit theme flag", func() {
		withColorFGBG("15;7", func() {
			runWithCustomPalettes([]string{
				appName, run1Flag, autoFlag, autoTimeoutFlag, immediateFlag,
			})
		})
	})

	It("should handle multiple runs with setup and cleanup", func() {
		withArgs([]string{
			appName, "--all", autoFlag, autoTimeoutFlag, immediateFlag,
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

	It("should show the help if no run is selected", func() {
		withArgs([]string{appName, "--continuously"}, func() {
			out := &strings.Builder{}
			sut := demo.New()
			sut.Writer = out
			sut.Add(demo.NewRun("Run 1"), "run1", "first run")

			Expect(sut.RunE()).To(Succeed())
			Expect(out.String()).To(ContainSubstring("GLOBAL OPTIONS"))
		})
	})

	It("should run the cleanups in order when interrupted by a signal", func() {
		binary, err := gexec.Build("github.com/saschagrunert/demo/testdata/interrupt")
		Expect(err).ToNot(HaveOccurred())

		DeferCleanup(gexec.CleanupBuildArtifacts)

		cmd := exec.Command(binary, "--run", autoFlag, autoTimeoutFlag, immediateFlag)
		session, err := gexec.Start(cmd, GinkgoWriter, GinkgoWriter)
		Expect(err).ToNot(HaveOccurred())

		Eventually(session).Should(gbytes.Say("> sleep 10"))
		session.Terminate()

		Eventually(session).Should(gexec.Exit(0))
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

		Eventually(session).Should(gbytes.Say("> sleep 10"))
		session.Terminate()

		Eventually(session).Should(gexec.Exit(1))
		Expect(session.Out).To(gbytes.Say("demo cleanup"))
		Expect(session.Err).To(gbytes.Say("run cleanup failed"))
	})

	It("should use the auto theme for empty options", func() {
		withColorFGBG("15;7", func() {
			origNoColor := color.NoColor
			color.NoColor = false

			defer func() { color.NoColor = origNoColor }()

			run := demo.NewRun("Run 1")
			out := &strings.Builder{}
			Expect(run.SetOutput(out)).To(Succeed())

			Expect(run.RunWithOptions(&demo.Options{Auto: true, Immediate: true})).To(Succeed())
			Expect(out.String()).To(ContainSubstring("\x1b[34m"))
		})
	})

	It("should run with a context and arguments", func() {
		out := &strings.Builder{}
		run := demo.NewRun("Run 1")
		Expect(run.SetOutput(out)).To(Succeed())

		sut := demo.New()
		sut.Add(run, "run1", "first run")

		err := sut.RunContext(context.Background(), []string{
			appName, run1Flag, autoFlag, autoTimeoutFlag, immediateFlag,
		})

		Expect(err).ToNot(HaveOccurred())
		Expect(out.String()).To(ContainSubstring("Run 1"))
	})

	It("should clean up and return no error when the context gets cancelled", func() {
		run := demo.NewRun("Run 1")
		Expect(run.SetOutput(&strings.Builder{})).To(Succeed())
		run.Step(nil, demo.S("sleep 10"))

		cleanedUp := false
		sut := demo.New()
		sut.Add(run, "run1", "first run")
		sut.Cleanup(func(ctx context.Context, _ *cli.Command) error {
			cleanedUp = ctx.Err() == nil

			return nil
		})

		ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
		defer cancel()

		err := sut.RunContext(
			ctx,
			[]string{appName, run1Flag, autoFlag, autoTimeoutFlag, immediateFlag},
		)

		Expect(err).ToNot(HaveOccurred())
		Expect(cleanedUp).To(BeTrue())
	})

	DescribeTable("should panic when a run name conflicts with a flag",
		func(name string) {
			sut := demo.New()

			Expect(func() { sut.Add(demo.NewRun("Run"), name, "conflict") }).
				To(PanicWith(ContainSubstring(name)))
		},
		Entry("built-in flag", "auto"),
		Entry("built-in alias", "a"),
		Entry("help flag", "help"),
	)

	It("should panic when a run name conflicts with the version flag", func() {
		sut := demo.New()
		sut.Version = "1.0.0"

		Expect(func() { sut.Add(demo.NewRun("Run"), "version", "conflict") }).
			To(PanicWith(ContainSubstring("version")))

		sut.HideVersion = true

		Expect(func() { sut.Add(demo.NewRun("Run"), "version", "no conflict") }).ToNot(Panic())
	})

	It("should run the demo cleanup when the setup fails", func() {
		run := demo.NewRun("Run 1")
		out := &strings.Builder{}
		Expect(run.SetOutput(out)).To(Succeed())

		cleanedUp := false
		sut := demo.New()
		sut.Add(run, "run1", "first run")
		sut.Setup(func(context.Context, *cli.Command) error {
			return errSetupFailed
		})
		sut.Cleanup(func(context.Context, *cli.Command) error {
			cleanedUp = true

			return nil
		})

		err := sut.RunContext(
			context.Background(),
			[]string{appName, run1Flag, autoFlag, autoTimeoutFlag, immediateFlag},
		)

		Expect(err).To(MatchError(errSetupFailed))
		Expect(cleanedUp).To(BeTrue())
		Expect(out.String()).To(BeEmpty())
	})
})
