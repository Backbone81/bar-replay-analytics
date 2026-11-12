package playback_test

import (
	"fmt"
	"path/filepath"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/backbone81/bar-replay-analytics-go/internal/playback"
	"github.com/backbone81/bar-replay-analytics-go/internal/replayplayer/widget"
)

var _ = Describe("CheckEventsFile", func() {
	var replayFilePath string

	BeforeEach(func() {
		replayFilePath = filepath.Join(GinkgoT().TempDir(), "game.sdfz")
	})

	It("reports a complete file of the widget's format version as current", func() {
		writeEventsFile(replayFilePath, new(widget.FormatVersion))

		Expect(playback.CheckEventsFile(replayFilePath)).To(Equal(playback.EventsFileStatus{
			Current:       true,
			FormatVersion: widget.FormatVersion,
			Reason:        fmt.Sprintf("events file current (format %d)", widget.FormatVersion),
		}))
	})

	It("reports a file of a newer widget as current", func() {
		writeEventsFile(replayFilePath, new(widget.FormatVersion+1))

		Expect(playback.CheckEventsFile(replayFilePath)).To(HaveField("Current", BeTrue()))
	})

	It("reports a file of an older format version as outdated", func() {
		writeEventsFile(replayFilePath, new(widget.FormatVersion-1))

		Expect(playback.CheckEventsFile(replayFilePath)).To(Equal(playback.EventsFileStatus{
			FormatVersion: widget.FormatVersion - 1,
			Reason: fmt.Sprintf("events file outdated (format %d, the widget writes %d)", widget.FormatVersion-1,
				widget.FormatVersion),
		}))
	})

	It("reports a file without format version as outdated", func() {
		writeEventsFile(replayFilePath, nil)

		Expect(playback.CheckEventsFile(replayFilePath)).To(And(
			HaveField("Current", BeFalse()), HaveField("FormatVersion", 0)))
	})

	DescribeTable("reports a file as not current",
		func(content string, reason string) {
			writeFile(playback.EventsFilePath(replayFilePath), []byte(content))

			status, err := playback.CheckEventsFile(replayFilePath)

			Expect(err).NotTo(HaveOccurred())
			Expect(status.Current).To(BeFalse())
			Expect(status.Reason).To(HavePrefix(reason))
		},
		Entry("without end", `{"formatVersion":99,"events":[]}`, "events file incomplete"),
		Entry("that ended before the game over", `{"formatVersion":99,"events":[],"end":{"frame":5,"reason":"stall"}}`,
			"events file incomplete"),
		Entry("cut off", `{"formatVersion":99,"events":[`, "events file unreadable"),
	)

	It("reports a missing file as not current", func() {
		Expect(playback.CheckEventsFile(replayFilePath)).To(Equal(playback.EventsFileStatus{Reason: "no events file"}))
	})
})

// writeEventsFile writes a complete events file of the playback of the replay file replayFilePath, with one event,
// the game over as its end and the format version formatVersion (none if nil).
func writeEventsFile(replayFilePath string, formatVersion *int) {
	formatVersionMember := ""
	if formatVersion != nil {
		formatVersionMember = fmt.Sprintf(`"formatVersion":%d,`, *formatVersion)
	}
	writeFile(playback.EventsFilePath(replayFilePath), fmt.Appendf(nil, `{%s"events":[
{"frame":0,"unitIdle":{"unitID":1,"unitDefID":1,"unitTeam":0}}
],"end":{"frame":100,"reason":"gameover"}}`, formatVersionMember))
}
