package demo_test

import (
	"testing"

	"github.com/fatih/color"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

// TestDemo runs the created specs.
func TestDemo(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "demo")
}

var _ = BeforeSuite(func() {
	// Do not depend on whether the output is a terminal, specs which check
	// colors enable them explicitly.
	color.NoColor = true
})
