package demo_test

import (
	"testing"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	. "github.com/saschagrunert/demo/test/framework"
)

// TestDemo runs the created specs.
func TestDemo(t *testing.T) {
	t.Parallel()
	RegisterFailHandler(Fail)
	RunFrameworkSpecs(t, "demo")
}

// nolint: gochecknoglobals
var t *TestFramework

var _ = BeforeSuite(func() {
	t = NewTestFramework(NilFunc, NilFunc)
	t.Setup()
})

var _ = AfterSuite(func() {
	t.Teardown()
})
