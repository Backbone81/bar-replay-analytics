// Package playback plays replays back in a headless engine (spring-headless) at maximum speed, so that the analytics
// widget of bar-replay-analytics (dbg_replay_analytics.lua) logs the game to an events file (see package events).
//
// A Player works in a BAR data dir, the folder with demos/, engine/, maps/, the game packages (rapid/, packages/,
// pool/) and LuaUI/Widgets/. The analytics widget is embedded (package widget) and written into each playback's write
// dir, so it need not be installed in the data dir; if it is, the embedded one replaces it in the playback. The widget
// does the work inside the engine: it is active only in a headless replay, raises the speed limit, skips to the end of
// the replay, and quits the engine when the game is over or the replay stops advancing. It writes
// events/<replay name>_events.json in its own folder, which is linked to events/ in the installed widget's folder in
// the data dir, or in LuaUI/Widgets/bar-replay-analytics/ if none is installed; the engine output goes next to it as
// <replay name>.log.
//
// Prepare reads the replay's engine version, game version (the setup script's gametype) and map, and downloads what
// the data dir is missing, as the BAR lobby does: the engine's release build from BAR's CDN (GitHub releases of
// RecoilEngine as a fallback), unpacked into engine/<version folder> in Go (no 7-Zip needed), and the game and map
// with the engine's pr-downloader and the BAR launcher's settings. Prepare calls of one Player run one at a time, so
// replays needing the same engine, game or map download it once.
//
// Play runs the engine. Several Play calls may run at the same time: each playback gets its own write dir, a
// temporary folder, and reads the data dir as an additional data dir (SPRING_DATADIR). The engine writes its
// infolog.txt, _script.txt, settings, archive cache and the widgets' LuaUI/Config/ there, so parallel playbacks do
// not overwrite each other's files, and the data dir is not changed except for the events and log files. Its
// settings let the system pick the UDP port of the local server which plays the replay. The write dir holds the
// analytics widget with its events/ folder linked to the data dir's, so the events file lands there.
// The write dir starts with a copy of the data dir's archive cache, so the engine does not scan every map and game
// again, and is removed afterwards.
//
// Play reports the progress every Config.ProgressInterval: the game frame of the last event, which the replay's
// header turns into a percentage of the game. It stops the engine if the playback hangs: the events file (or, while
// the engine loads, its log) has not grown for Config.StallTimeout, or the widget reported that it cannot write its
// events file, as the replay would then play in real time.
//
// Names read from the replay (engine version, gametype, map name) end up in paths and command lines, so Prepare
// rejects any that is not a plain name.
package playback
