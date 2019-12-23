package demo_test

import (
	"os"

	. "github.com/onsi/ginkgo"

	"github.com/saschagrunert/demo"
)

var _ = t.Describe("Demo", func() {
	It("should succeed to run", func() {
		// Given
		sut := demo.New()
		os.Args = []string{
			"demo", "--auto", "--immediate",
		}

		// When
		sut.Run()
	})
})
