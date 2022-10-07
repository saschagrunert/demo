package demo_test

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	. "github.com/saschagrunert/demo/test/framework"
)

// TestDemo runs the created specs.
func TestDemo(t *testing.T) {
	t.Parallel()
	RegisterFailHandler(Fail)
	RunFrameworkSpecs(t, "demo")
}

//nolint:gochecknoglobals // the framework has to be global
var t *TestFramework

var _ = BeforeSuite(func() {
	t = NewTestFramework(NilFunc, NilFunc)
	t.Setup()
})

var _ = AfterSuite(func() {
	t.Teardown()
})
