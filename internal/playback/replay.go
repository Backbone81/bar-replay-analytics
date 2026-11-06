package playback

import (
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"github.com/backbone81/bar-replay-analytics-go/internal/demo"
	"github.com/backbone81/bar-replay-analytics-go/internal/tdf"
)

// Replay is a replay which Prepare made ready to play back.
type Replay struct {
	// FilePath is the absolute path of the replay file.
	FilePath string

	// Name is the file name without .sdfz, which the analytics widget names its events file after.
	Name string

	// EngineVersion is the engine version which recorded the replay, such as 2026.07.04.
	EngineVersion string

	// GameType is the game version, the setup script's gametype, such as "Beyond All Reason test-31720-0f97aa5".
	GameType string

	// MapName is the map, the setup script's mapname, such as "Argent Strata 1.1".
	MapName string

	// GameTime is the length of the game in game time, from the replay's header. Is zero if the engine crashed while
	// recording, or the game did not end.
	GameTime time.Duration

	// Engine is the path of spring-headless at EngineVersion.
	Engine string

	// GamePackage is the path of the rapid package of GameType.
	GamePackage string

	// MapArchive is the path of the archive of MapName.
	MapArchive string
}

// EventsFileName returns the file name of the events file the analytics widget writes for the replay.
func (r *Replay) EventsFileName() string {
	return r.Name + "_events.json"
}

// EventsFilePath returns the path of the events file of the replay's playback: next to the replay file.
func (r *Replay) EventsFilePath() string {
	return EventsFilePath(r.FilePath)
}

// LogFilePath returns the path of the engine log of the replay's playback: next to the replay file.
func (r *Replay) LogFilePath() string {
	return LogFilePath(r.FilePath)
}

// EventsFilePath returns the path of the events file of the playback of the replay file replayFilePath:
// <replay name>_events.json next to it. BAR's replay browser lists only the .sdfz files in demos/, so it does not
// show the file.
func EventsFilePath(replayFilePath string) string {
	return siblingPath(replayFilePath, "_events.json")
}

// LogFilePath returns the path of the engine log of the playback of the replay file replayFilePath:
// <replay name>.log next to it.
func LogFilePath(replayFilePath string) string {
	return siblingPath(replayFilePath, ".log")
}

// siblingPath returns the path of the file next to the replay file replayFilePath that is named after the replay
// (its file name without extension) with suffix.
func siblingPath(replayFilePath string, suffix string) string {
	return strings.TrimSuffix(replayFilePath, filepath.Ext(replayFilePath)) + suffix
}

// readReplay reads the names of the engine, game and map from the replay file filePath, and checks them.
func readReplay(filePath string) (*Replay, error) {
	replayFile, err := demo.ReadFile(filePath)
	if err != nil {
		return nil, err
	}

	setupScript, err := tdf.Parse(replayFile.SetupScript)
	if err != nil {
		return nil, fmt.Errorf("setup script: %w", err)
	}

	gameType, _ := setupScript.Value(`game\gametype`)
	mapName, _ := setupScript.Value(`game\mapname`)

	replay := &Replay{
		FilePath:      filePath,
		Name:          strings.TrimSuffix(filepath.Base(filePath), filepath.Ext(filePath)),
		EngineVersion: replayFile.FileHeader.VersionStringAsString(),
		GameType:      gameType.String(),
		MapName:       mapName.String(),
		GameTime:      time.Duration(replayFile.FileHeader.GameTime) * time.Second,
	}
	if err := validateReplayName(replay.EngineVersion); err != nil {
		return nil, fmt.Errorf("the replay's engine version: %w", err)
	}
	if err := validateReplayName(replay.GameType); err != nil {
		return nil, fmt.Errorf("the replay's gametype: %w", err)
	}
	if err := validateReplayName(replay.MapName); err != nil {
		return nil, fmt.Errorf("the replay's mapname: %w", err)
	}
	return replay, nil
}
