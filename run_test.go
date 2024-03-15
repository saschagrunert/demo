package demo_test

import (
	"flag"
	"strings"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/saschagrunert/demo"
	"github.com/urfave/cli/v2"
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
		err := sut.RunWithOptions(opts)

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
		err := sut.RunWithOptions(opts)

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
		err := sut.RunWithOptions(opts)

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
})
