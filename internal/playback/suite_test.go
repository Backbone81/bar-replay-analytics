package playback_test

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestPlayback(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Playback Suite")
}
