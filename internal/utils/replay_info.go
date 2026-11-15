package utils

import (
	"errors"
	"fmt"
	"path/filepath"
	"time"

	"github.com/backbone81/bar-replay-analytics/internal/demo"
	"github.com/backbone81/bar-replay-analytics/internal/tdf"
)

// ReplayInfo is what ReadReplayInfo found out about a replay.
type ReplayInfo struct {
	// FilePath is the absolute path of the replay file.
	FilePath string

	// EngineVersion is the engine tag from the replay's header, e.g. 2026.07.04.
	EngineVersion string

	// MapName is the start script's mapname.
	MapName string

	// GameType is the start script's gametype, the game's rapid name, e.g. "Beyond All Reason test-31720-0f97aa5".
	GameType string

	// PlayerNames are the names of the players and AIs of the start script that play. An AI without a name has its
	// short name, as CGameSetup::LoadSkirmishAIs names it.
	PlayerNames []string

	// StartTime is when the game was started, in UTC, from the header. BAR's replay file names start with it too.
	StartTime time.Time

	// Duration is the game time the header records.
	Duration time.Duration
}

// ReadReplayInfo reads the header and the start script of the replay filePath. It fails for a replay without a
// recorded stream (the game crashed), which the engine cannot play back, and for one without a game time (0:00 in the
// lobby: the engine crashed or the game was not ended).
func ReadReplayInfo(filePath string) (*ReplayInfo, error) {
	filePath, err := filepath.Abs(filePath)
	if err != nil {
		return nil, err
	}

	demoFile, err := demo.ReadFile(filePath, demo.ReadUntilSetupScript)
	if err != nil {
		return nil, err
	}

	if demoFile.FileHeader.DemoStreamSize <= 0 {
		return nil, errors.New("the demo does not have a positive stream size")
	}
	if demoFile.FileHeader.GameTime <= 0 {
		return nil, errors.New("the demo does not have a positive game time")
	}

	setupScript, err := tdf.Parse(demoFile.SetupScript)
	if err != nil {
		return nil, err
	}

	gameType, ok := setupScript.Value(`game\gametype`)
	if !ok {
		return nil, errors.New("the setup script has no game\\gametype")
	}

	mapName, ok := setupScript.Value(`game\mapname`)
	if !ok {
		return nil, errors.New("the setup script has no game\\mapname")
	}

	playerNames, err := GetPlayerNames(setupScript.Section("game"))
	if err != nil {
		return nil, err
	}

	return &ReplayInfo{
		FilePath:      filePath,
		EngineVersion: demoFile.FileHeader.VersionStringAsString(),
		MapName:       mapName.String(),
		GameType:      gameType.String(),
		PlayerNames:   playerNames,
		Duration:      demoFile.FileHeader.GameTimeAsDuration(),
		StartTime:     time.Unix(int64(demoFile.FileHeader.UnixTime), 0).UTC(), //nolint:gosec // no overflow
	}, nil
}

// GetPlayerNames returns the names of the playing players and AIs of the start script's game section.
func GetPlayerNames(game *tdf.Section) ([]string, error) {
	var result []string
	for number, section := range game.Indexed("player") {
		spectator, err := isSpectator(section)
		if err != nil {
			return nil, fmt.Errorf("player%d: %w", number, err)
		}
		if spectator {
			continue
		}
		name, _ := section.Value("name")
		result = append(result, name.String())
	}
	for _, section := range game.Indexed("ai") {
		name, ok := section.Value("name")
		if !ok {
			name, _ = section.Value("shortname")
		}
		result = append(result, name.String())
	}
	return result, nil
}

// isSpectator returns whether a player section is a spectator's.
func isSpectator(section *tdf.Section) (bool, error) {
	value, ok := section.Value("spectator")
	if !ok {
		return false, nil
	}

	spectator, err := value.AsBool()
	if err != nil {
		return false, fmt.Errorf("spectator: %w", err)
	}
	return spectator, nil
}
