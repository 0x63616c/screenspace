package service_test

import (
	"log/slog"
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestService(t *testing.T) {
	slog.SetDefault(slog.New(slog.DiscardHandler))
	RegisterFailHandler(Fail)
	RunSpecs(t, "Service Suite")
}
