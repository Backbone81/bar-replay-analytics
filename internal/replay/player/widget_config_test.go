package player_test

import (
	"io/fs"
	"path/filepath"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	replayplayer "github.com/backbone81/bar-replay-analytics-go/internal/replay/player"
	"github.com/backbone81/bar-replay-analytics-go/internal/replay/player/widget"
)

var _ = Describe("WidgetConfig", func() {
	Context("ReadWidgetNameLiteral", func() {
		DescribeTable("reads the name from the widget's GetInfo as its literal, quotes included",
			func(source string, expected string) {
				literal, ok := replayplayer.ReadWidgetNameLiteral([]byte(source))

				Expect(ok).To(BeTrue())
				Expect(literal).To(Equal(expected))
			},
			Entry("in double quotes",
				"function widget:GetInfo()\n\treturn {\n\t\tname = \"Advisor\",\n\t}\nend\n", `"Advisor"`),
			Entry("in single quotes",
				"function widget:GetInfo()\n\treturn {\n\t\tname = 'Advisor',\n\t}\nend\n", `'Advisor'`),
			Entry("in single quotes with a double quote",
				"function widget:GetInfo()\n\treturn { name = 'Say \"hi\"' }\nend\n", `'Say "hi"'`),
			Entry("with bytes that are not ASCII",
				"function widget:GetInfo()\n\treturn { name = \"Ähre\" }\nend\n", `"Ähre"`),
			Entry("after other fields",
				"function widget:GetInfo()\n\treturn {\n\t\tdesc = \"x\",\n\t\tname=\"Advisor\",\n\t}\nend\n",
				`"Advisor"`),
			Entry("not a name before GetInfo",
				"local name = \"other\"\nfunction widget:GetInfo()\n\treturn { name = \"Advisor\" }\nend\n",
				`"Advisor"`),
		)

		It("reads the name of the analytics widget", func() {
			source, err := fs.ReadFile(widget.Template, widget.FileName)
			Expect(err).NotTo(HaveOccurred())

			literal, ok := replayplayer.ReadWidgetNameLiteral(source)

			Expect(ok).To(BeTrue())
			Expect(literal).To(Equal(`"` + widget.Name + `"`))
		})

		DescribeTable("finds no name",
			func(source string) {
				_, ok := replayplayer.ReadWidgetNameLiteral([]byte(source))

				Expect(ok).To(BeFalse())
			},
			Entry("without GetInfo", "local name = \"Advisor\"\n"),
			Entry("without a name in GetInfo", "function widget:GetInfo()\n\treturn {}\nend\n"),
			Entry("with an escape in the name", "function widget:GetInfo()\n\treturn { name = \"A\\\"B\" }\nend\n"),
			Entry("with a carriage return in the name",
				"function widget:GetInfo()\n\treturn { name = \"A\rB\" }\nend\n"),
		)
	})

	Context("UserWidgetNameLiterals", func() {
		It("returns the sorted names of the widgets directly in the folder and one subfolder deep", func() {
			widgetsDir := GinkgoT().TempDir()
			writeFile(filepath.Join(widgetsDir, "b.lua"), widgetSource("Bravo"))
			writeFile(filepath.Join(widgetsDir, "folder", "a.lua"), widgetSource("Alpha"))
			writeFile(filepath.Join(widgetsDir, "folder", "deeper", "c.lua"), widgetSource("Charlie"))
			writeFile(filepath.Join(widgetsDir, "folder", "d.txt"), widgetSource("Delta"))
			writeFile(filepath.Join(widgetsDir, "other", "e.lua"), widgetSource("Bravo"))
			writeFile(filepath.Join(widgetsDir, "unnamed.lua"), "return {}\n")

			Expect(replayplayer.UserWidgetNameLiterals(widgetsDir)).To(HaveExactElements(`"Alpha"`, `"Bravo"`))
		})

		It("returns no names for a missing folder", func() {
			Expect(replayplayer.UserWidgetNameLiterals(filepath.Join(GinkgoT().TempDir(), "missing"))).To(BeEmpty())
		})
	})

	Context("WidgetConfig", func() {
		It("enables each widget once, sorted, with its literal as it is", func() {
			Expect(string(replayplayer.WidgetConfig([]string{
				`"Replay Analytics"`, `'Say "hi"'`, `"Advisor"`,
				`"Advisor"`,
			}))).To(Equal(
				"-- Widget order written by bar-replay-analytics-go for the playback\n" +
					"return {\n" +
					"\torder = {\n" +
					"\t\t[\"Advisor\"] = 12345,\n" +
					"\t\t[\"Replay Analytics\"] = 12345,\n" +
					"\t\t['Say \"hi\"'] = 12345,\n" +
					"\t},\n" +
					"}\n",
			))
		})
	})
})

// widgetSource returns the source of a widget whose GetInfo gives name in double quotes.
func widgetSource(name string) string {
	return "function widget:GetInfo()\n\treturn {\n\t\tname = \"" + name + "\",\n\t}\nend\n"
}
