package replayplayer

import "time"

// ReplayInfo is what Player.Prepare found out about a replay and what Player.Run needs to play it back.
type ReplayInfo struct {
	// FilePath is the absolute path of the replay file.
	FilePath string

	// EngineVersion is the engine tag from the replay's header, e.g. 2026.07.04.
	EngineVersion string

	// EngineDir is the folder of that engine in the data dir.
	EngineDir string

	// MapName is the start script's mapname.
	MapName string

	// GameType is the start script's gametype, the game's rapid name, e.g. "Beyond All Reason test-31720-0f97aa5".
	GameType string

	// Duration is the game time the header records.
	Duration time.Duration
}
