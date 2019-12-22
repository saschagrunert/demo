package demo_test

import (
	"strings"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/saschagrunert/demo"
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
		Expect(err).To(BeNil())

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
		Expect(err).To(BeNil())
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
		Expect(err).To(BeNil())
		Expect(out).To(ContainSubstring(title))
		Expect(out).To(ContainSubstring(description[0]))
		Expect(out).To(ContainSubstring(description[1]))
		Expect(out).To(ContainSubstring(descriptionText))
		Expect(out).To(ContainSubstring(command))
	})

	It("should fail to set nil output", func() {
		// Given
		// When
		err := sut.SetOutput(nil)

		// Then
		Expect(err).NotTo(BeNil())
	})
})
