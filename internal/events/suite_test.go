package events_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/backbone81/bar-replay-analytics/internal/events"
)

func TestEvents(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Events Suite")
}

// writeFile writes content to filePath, creating its folder.
func writeFile(filePath string, content string) {
	Expect(os.MkdirAll(filepath.Dir(filePath), 0o750)).To(Succeed())
	Expect(os.WriteFile(filePath, []byte(content), 0o600)).To(Succeed())
}

// eventsFileContent returns an events file of formatVersion that ended for reason, or has no end if reason is "".
func eventsFileContent(formatVersion int, reason events.EndReason) string {
	file := map[string]any{
		"formatVersion": formatVersion,
	}
	if reason != "" {
		file["end"] = events.End{Frame: 48210, Reason: reason}
	}
	data, err := json.Marshal(file)
	Expect(err).NotTo(HaveOccurred())
	return string(data)
}
