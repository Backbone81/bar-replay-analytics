package widget

import (
	"io/fs"
	"regexp"
	"strconv"
)

// FileName is the file name of the analytics widget in Template.
const FileName = "dbg_replay_analytics.lua"

// Name is the name the analytics widget gives in its GetInfo, by which BAR's widget handler enables it.
const Name = "Replay Analytics"

// FormatVersion is the version of the events file format the widget writes: the FORMAT_VERSION it sets, its only
// definition, which it writes into each events file as "formatVersion" (events.File.FormatVersion). An events file of
// a lower version is outdated.
var FormatVersion = readFormatVersion()

// formatVersionPattern matches the line of the widget that sets FORMAT_VERSION, with CRLF or LF line endings.
var formatVersionPattern = regexp.MustCompile(`(?m)^local FORMAT_VERSION = (\d+)\r?$`)

// readFormatVersion returns the FORMAT_VERSION the widget in Template sets. It panics if the widget does not set it
// exactly once to a number, so every test of a package that imports this one fails then.
func readFormatVersion() int {
	data, err := fs.ReadFile(Template, FileName)
	if err != nil {
		panic(err)
	}
	matches := formatVersionPattern.FindAllSubmatch(data, -1)
	if len(matches) != 1 {
		panic(FileName + ": FORMAT_VERSION must be set exactly once, to a number")
	}
	version, err := strconv.Atoi(string(matches[0][1]))
	if err != nil {
		panic(err)
	}
	return version
}
