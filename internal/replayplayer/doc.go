// Package replayplayer plays BAR replays back in the headless engine to write their events files.
//
// A Player prepares a replay (reads its header and start script, makes sure its engine, map and game are in the data
// dir) and then runs spring-headless on it with a temporary write dir of its own. The embedded analytics widget writes
// the events file there; Run copies it next to the replay as <replay name>_events.json and the engine's output goes
// to <replay name>.log. The data dir is only read, apart from downloads, so it can be the player's own BAR install and
// several playbacks can share it.
//
// A BatchPlayer plays many replays with several Players in parallel and skips replays whose events file is current.
// The Players share one AssetManager, which serializes downloads so no asset is downloaded twice.
package replayplayer
