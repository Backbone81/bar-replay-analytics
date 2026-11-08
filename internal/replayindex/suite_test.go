package replayindex_test

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestReplayIndex(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Replay Index Suite")
}
