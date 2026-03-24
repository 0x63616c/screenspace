package handler_test

import (
	"log/slog"
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestHandler(t *testing.T) {
	slog.SetDefault(slog.New(slog.DiscardHandler))
	RegisterFailHandler(Fail)
	RunSpecs(t, "Handler Suite")
}
