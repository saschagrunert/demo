package demo_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/saschagrunert/demo"
)

var _ = t.Describe("Util", func() {
	It("should succeed to Ensure", func() {
		// Given
		// When
		err := demo.Ensure("echo hi")

		// When
		Expect(err).ToNot(HaveOccurred())
	})
})
