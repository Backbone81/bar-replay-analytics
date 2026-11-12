// Package widget embeds the template of the analytics widget's folder: the BAR user widget that logs a playback to an
// events file (see package events), a README and the license. It is written into each playback's write dir, and by
// install-widget into the data dir's LuaUI/Widgets/.
//
// The widget is GPL-2.0-or-later like BAR's widgets, as it runs inside the game; the LICENSE is written with it.
package widget

import (
	"embed"
	"io/fs"
)

// templateFiles holds template/. all: includes the dot files (.stylua.toml, .gitattributes).
//
//go:embed all:template
var templateFiles embed.FS

// Template holds the files of the widget folder: dbg_replay_analytics.lua, README.md, LICENSE and the dot files.
var Template = mustSub(templateFiles, "template")

// mustSub returns the subtree dir of fsys. It panics if dir is not a valid path, which the embed directive rules out.
func mustSub(fsys fs.FS, dir string) fs.FS {
	sub, err := fs.Sub(fsys, dir)
	if err != nil {
		panic(err)
	}
	return sub
}
