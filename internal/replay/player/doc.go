// Package player plays BAR replays back in the headless engine to write their events files.
//
// ReadReplayInfo reads a replay's header and start script. A Player prepares the replay (makes sure its engine, map
// and game are in the data dir) and then runs spring-headless on it with a temporary write dir of its own. The
// embedded analytics widget writes the events file there; Run copies it next to the replay as
// <replay name>_events.json and the engine's output goes to <replay name>.log. The data dir is only read, apart from
// downloads, so it can be the player's own BAR install and several playbacks can share it.
//
// A BatchPlayer plays many replays with several Players in parallel. It selects the replays in file path order before
// anything is downloaded: it skips those whose events file is current and those its Filter does not match, and stops
// after MaxReplays. The Players share one AssetManager, which serializes downloads so no asset is downloaded twice.
package player
