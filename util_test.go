package demo_test

import (
	"context"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/saschagrunert/demo"
)

var _ = t.Describe("Util", func() {
	It("should succeed to Ensure", func() {
		// Given
		// When
		err := demo.Ensure("echo hi")

		// Then
		Expect(err).ToNot(HaveOccurred())
	})

	It("should succeed to Ensure with multiple commands", func() {
		// Given
		// When
		err := demo.Ensure("echo first", "echo second", "echo third")

		// Then
		Expect(err).ToNot(HaveOccurred())
	})

	It("should fail to Ensure with invalid command", func() {
		// Given
		// When
		err := demo.Ensure("commandthatdoesnotexist12345")

		// Then
		Expect(err).To(HaveOccurred())
	})

	It("should succeed to EnsureWithContext", func() {
		// Given
		ctx := context.Background()

		// When
		err := demo.EnsureWithContext(ctx, "echo test")

		// Then
		Expect(err).ToNot(HaveOccurred())
	})

	It("should fail to EnsureWithContext with cancelled context", func() {
		// Given
		ctx, cancel := context.WithCancel(context.Background())
		cancel()

		// When
		err := demo.EnsureWithContext(ctx, "sleep 10")

		// Then
		Expect(err).To(HaveOccurred())
	})

	It("should panic with MustEnsure on error", func() {
		// Given
		// When/Then
		Expect(func() {
			demo.MustEnsure("commandthatdoesnotexist12345")
		}).To(Panic())
	})

	It("should succeed with MustEnsure on success", func() {
		// Given
		// When/Then
		Expect(func() {
			demo.MustEnsure("echo test")
		}).ToNot(Panic())
	})
})
