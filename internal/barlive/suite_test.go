package barlive_test

import (
	"context"
	"errors"
	"io"
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/backbone81/bar-replay-analytics/internal/barlive"
	"github.com/backbone81/bar-replay-analytics/internal/barlive/barlivetest"
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

// downloadDemo downloads the demo file demoFileName with client and returns its content.
func downloadDemo(ctx context.Context, client *barlive.Client, demoFileName string) ([]byte, error) {
	body, err := client.DownloadDemo(ctx, demoFileName)
	if err != nil {
		return nil, err
	}
	data, err := io.ReadAll(body)
	if err := errors.Join(err, body.Close()); err != nil {
		return nil, err
	}
	return data, nil
}
