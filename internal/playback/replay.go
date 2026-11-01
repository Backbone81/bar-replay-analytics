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

// LogFileName returns the file name of the engine log of the replay's playback.
func (r *Replay) LogFileName() string {
	return r.Name + ".log"
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
