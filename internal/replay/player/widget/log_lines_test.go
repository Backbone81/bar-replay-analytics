package widget_test

import (
	"io/fs"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/backbone81/bar-replay-analytics/internal/replay/player/widget"
)

var _ = Describe("Log lines", func() {
	It("are the lines the widget logs", func() {
		data, err := fs.ReadFile(widget.Template, widget.FileName)
		Expect(err).ToNot(HaveOccurred())

		Expect(string(data)).To(ContainSubstring(`spEcho("` + widget.ActiveLogLine + `")`))
		Expect(string(data)).To(ContainSubstring(`"` + widget.HeartbeatLogLine + ` (frame %d)"`))
	})
})
