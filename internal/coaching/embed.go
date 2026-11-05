// Package coaching holds the template of the workspace of an AI agent that coaches a player of Beyond All Reason from
// the analytics of their replays: its instructions (AGENTS.md), and templates of the files and folders those tell it to
// keep. They are embedded into the binary, so a user who installed it with go install can write them into the folder
// they run their agent in.
package coaching

import (
	"embed"
	"io/fs"
)

// templateFiles holds the template under template/. The all: prefix includes work/.keep, which keeps the otherwise
// empty folder work/ in git and in the embedded files.
//
//go:embed all:template
var templateFiles embed.FS

// Template provides the files of the workspace template under their paths in it, e.g. AGENTS.md and work/.keep.
var Template = mustSub(templateFiles, "template")

// mustSub returns the subtree dir of fsys. It panics if dir is not a valid path, which the embed directive rules out.
func mustSub(fsys fs.FS, dir string) fs.FS {
	sub, err := fs.Sub(fsys, dir)
	if err != nil {
		panic(err)
	}
	return sub
}
