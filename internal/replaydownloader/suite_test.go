package replaydownloader_test

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestReplayDownloader(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Replay Downloader Suite")
}
