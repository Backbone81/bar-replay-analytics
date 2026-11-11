package replayplayer

import "time"

type ReplayInfo struct {
	FilePath      string
	EngineVersion string
	EngineDir     string
	MapName       string
	GameType      string
	Duration      time.Duration
}
