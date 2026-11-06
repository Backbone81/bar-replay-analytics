// Package playback plays replays back in spring-headless at maximum speed, so that the analytics widget
// (dbg_replay_analytics.lua, embedded from package widget) logs each game to an events file (see package events).
//
// A Player works in a BAR data dir: demos/, engine/, maps/, the game packages (rapid/, packages/, pool/) and
// LuaUI/Widgets/.
//
// Prepare reads the replay's engine version, gametype and mapname and downloads what the data dir lacks, as the BAR
// lobby does: the engine's release build from BAR's CDN (RecoilEngine's GitHub releases as fallback), unpacked in Go
// into engine/<EngineFolderName>, and the game and map with the engine's pr-downloader and the BAR launcher's
// settings. Prepare calls run one at a time, so a download shared by several replays runs once. The names come from
// an untrusted file and end up in paths and command lines, so Prepare rejects any that is not a plain name.
//
// Play runs the engine with a temporary write dir of its own (see MakeWriteDir) and the data dir as additional data
// dir (SPRING_DATADIR), so playbacks can run in parallel and leave the data dir alone. The widget writes the events
// file into the write dir, and Play copies it next to the replay file before it removes the write dir:
// <replay name>_events.json (see EventsFilePath); the engine's output goes there directly as <replay name>.log. The
// widget is active only in a headless replay; it raises the speed limit, skips to the end and quits the engine when
// the game is over or the replay stops advancing.
//
// Play reports progress every Config.ProgressInterval as the frame of the last event. It stops the engine if the
// events file (while loading: the log) has not grown for Config.StallTimeout, or the widget cannot write its events
// file, as the replay would then play in real time.
package playback
