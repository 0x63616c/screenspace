package respond_test

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestRespond(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Respond Suite")
}
