package barlive_test

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/backbone81/bar-replay-analytics-go/internal/barlive/barlivetest"
)

func TestBarlive(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Barlive Suite")
}

// newServer starts a barlivetest.Server which the spec closes when it ends.
func newServer() *barlivetest.Server {
	server := barlivetest.NewServer()
	DeferCleanup(server.Close)
	return server
}
