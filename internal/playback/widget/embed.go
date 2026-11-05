// Package widget holds the template of the analytics widget's folder: the BAR user widget which logs a replay's
// playback to an events file (see package events), a README explaining it to whoever finds the folder, and its
// license. It is embedded into the binary, and written into each playback's write dir, or by
// bar-replay-analytics-go install-widget into LuaUI/Widgets/ of the BAR data directory.
//
// The widget is licensed under GPL-2.0-or-later, like BAR's own widgets, as it runs inside the game. Whoever gets the
// widget gets its license too, so the two are written out together.
package widget

import (
	"embed"
	"io/fs"
)

// templateFiles holds the template under template/. The all: prefix includes files whose names start with a dot.
//
//go:embed all:template
var templateFiles embed.FS

// Template provides the files of the widget folder's template under their paths in it, e.g. dbg_replay_analytics.lua,
// README.md and LICENSE.
var Template = mustSub(templateFiles, "template")

// mustSub returns the subtree dir of fsys. It panics if dir is not a valid path, which the embed directive rules out.
func mustSub(fsys fs.FS, dir string) fs.FS {
	sub, err := fs.Sub(fsys, dir)
	if err != nil {
		panic(err)
	}
	return sub
}
