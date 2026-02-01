package smartio_test

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestSmartio(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Smartio Suite")
}
