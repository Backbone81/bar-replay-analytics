// Package widget holds the analytics widget, the BAR user widget which logs a replay's playback to an events file (see
// package events), and its license. Both are embedded into the binary, so a playback needs no widget installed in the
// BAR data directory.
//
// The widget is licensed under GPL-2.0-or-later, like BAR's own widgets, as it runs inside the game. Whoever gets the
// widget gets its license too, so the two are written out together.
package widget

import (
	"embed"
)

// Files provides the analytics widget and its license under their file names, so os.CopyFS writes both into a widget
// folder.
//
//go:embed dbg_replay_analytics.lua LICENSE
var Files embed.FS
