package report

import (
	"encoding/json"
	"fmt"
	"os"
	"slices"

	"github.com/backbone81/bar-replay-analytics/internal/demo"
	"github.com/backbone81/bar-replay-analytics/internal/events"
	"github.com/backbone81/bar-replay-analytics/internal/replay/player/widget"
	"github.com/backbone81/bar-replay-analytics/internal/tdf"
	"github.com/backbone81/bar-replay-analytics/internal/utils"
)

// FormatVersion is the version of the report's format. Increase it whenever the report of the same replay and events
// file changes (a field or value added, removed or computed differently), so that an outdated report can be told
// apart. The golden test (testdata/report.json) fails on every such change to remind of it.
const FormatVersion = 1

// Report is the prepared analytics of a replay.
type Report struct {
	// FormatVersion is the version of the report's format, see FormatVersion.
	FormatVersion int `json:"formatVersion"`

	// EventsFormatVersion is the format version of the events file the report was prepared from
	// (events.File.FormatVersion): a report of outdated events stays outdated until the replay is played back again.
	EventsFormatVersion int `json:"eventsFormatVersion"`

	// Metadata describes the game.
	Metadata Metadata `json:"metadata"`

	// MapGrid divides the map into the cells Player.LossLocations refers to, and tells where each player started.
	MapGrid MapGrid `json:"mapGrid"`

	// Teams provides every team but Gaia's, indexed by team ID.
	Teams []Team `json:"teams"`

	// Players provides every player but Gaia, indexed by player ID.
	Players []Player `json:"players"`
}

// NewReport prepares the report of a replay from the replay, its parsed setup script and the events file of its
// playback, configured by DefaultConfig changed by options.
// Returns an error if the configuration is invalid, if the setup script is one the engine refuses or holds an invalid
// value the report reads, if it disagrees with the statistics on the number of players, or if the events file is
// invalid or not of this game.
//
//nolint:funlen // One part of the report after the other reads best in one function.
func NewReport(
	replay *demo.Demo,
	setupScript *tdf.Section,
	eventsFile *events.File,
	options ...Option,
) (*Report, error) {
	config := DefaultConfig
	for _, option := range options {
		option(&config)
	}
	if err := config.Validate(); err != nil {
		return nil, err
	}

	game := setupScript.Section("game")
	metadata := newMetadata(replay, setupScript)
	teams, teamIDs := newTeams(game, metadata.WinningTeams)
	players, err := newPlayers(game, teamIDs)
	if err != nil {
		return nil, fmt.Errorf("setup script: %w", err)
	}
	// Some replays hold empty histories for fewer teams than there are, which is as good as no statistics.
	hasStats := slices.ContainsFunc(replay.TeamStats, func(history []demo.TeamStatistics) bool {
		return len(history) > 0
	})
	if hasStats && len(replay.TeamStats) != len(players) {
		return nil, fmt.Errorf(
			"setup script: got %d teams, want the %d of the statistics",
			len(players),
			len(replay.TeamStats),
		)
	}
	if err := eventsFile.Validate(); err != nil {
		return nil, err
	}
	if err := validateEventsGame(eventsFile, metadata, len(players)); err != nil {
		return nil, fmt.Errorf("events: %w", err)
	}
	buildOrders, err := newOpeningBuildOrders(eventsFile, len(players), config)
	if err != nil {
		return nil, fmt.Errorf("events: %w", err)
	}
	firstFinished := newFirstFinished(eventsFile, len(players))
	metalExtractors, err := newMetalExtractorHistories(eventsFile, len(players))
	if err != nil {
		return nil, fmt.Errorf("events: %w", err)
	}
	builderActivities, err := newBuilderActivities(eventsFile, len(players))
	if err != nil {
		return nil, fmt.Errorf("events: %w", err)
	}
	combatHistories := newCombatHistories(eventsFile, players)
	mapGrid := newMapGrid(eventsFile, len(players), config)
	resurrectionHistories := newResurrectionHistories(eventsFile, len(players))
	endFrame := int(replay.FileHeader.GameTime) * GameSpeed
	spans := newIntervalSpans(endFrame, int(replay.FileHeader.TeamStatPeriod), config)
	lossSpans := lossLocationSpans(spans, config)
	for i := range players {
		players[i].setFactionFromStartUnit(eventsFile)
		players[i].Economy = newEconomy(
			economyHistory(replay, eventsFile, i),
			metalExtractors[i],
			builderActivities[i],
			spans,
			endFrame,
		)
		players[i].Combat = newCombat(combatHistories[i], spans)
		players[i].LossLocations = newLossLocations(combatHistories[i], mapGrid, lossSpans)
		players[i].Resurrection = newResurrection(resurrectionHistories[i], spans)
		players[i].OpeningBuildOrder = buildOrders[i]
		players[i].FirstFinished = firstFinished[i]
		players[i].Builders = builderActivities[i].builders(eventsFile.UnitDefs, config)
	}
	for _, player := range players {
		teams[player.TeamID].PlayerIDs = append(teams[player.TeamID].PlayerIDs, player.PlayerID)
	}

	return &Report{
		FormatVersion:       FormatVersion,
		EventsFormatVersion: eventsFile.FormatVersion,
		Metadata:            metadata,
		MapGrid:             mapGrid,
		Teams:               teams,
		Players:             players,
	}, nil
}

// validateEventsGame returns an error if the events file is not of the game of metadata with numPlayers players: if its
// map, game version or number of teams differ.
func validateEventsGame(eventsFile *events.File, metadata Metadata, numPlayers int) error {
	game := eventsFile.Game
	if game.MapName != metadata.Map {
		return fmt.Errorf("map: got %q, want %q", game.MapName, metadata.Map)
	}
	if gameVersion := game.GameName + " " + game.GameVersion; gameVersion != metadata.GameVersion {
		return fmt.Errorf("game version: got %q, want %q", gameVersion, metadata.GameVersion)
	}
	if len(eventsFile.Teams) != numPlayers {
		return fmt.Errorf("got %d teams, want the %d of the setup script", len(eventsFile.Teams), numPlayers)
	}
	return nil
}

// WriteReport prepares the report of the replay filePath (see NewReport) from the replay and the events file of its
// playback next to it, configured by DefaultConfig changed by options, and writes it as indented JSON to
// <replay name>_report.json next to the replay, replacing an existing one. Returns an error if the events file is
// missing, invalid or older than the embedded widget's format version, or if NewReport fails.
func WriteReport(filePath string, options ...Option) error {
	eventsFile, err := events.ReadFile(utils.DemoEventsFilePath(filePath))
	if err != nil {
		return fmt.Errorf("events file of the replay's playback: %w", err)
	}
	if eventsFile.FormatVersion < widget.FormatVersion {
		return fmt.Errorf("the events file has format version %d, the widget writes %d: play the replay back again",
			eventsFile.FormatVersion, widget.FormatVersion)
	}

	replay, err := demo.ReadFile(filePath, demo.ReadUntilEnd)
	if err != nil {
		return err
	}

	setupScript, err := tdf.Parse(replay.SetupScript)
	if err != nil {
		return fmt.Errorf("setup script: %w", err)
	}

	report, err := NewReport(
		replay,
		setupScript,
		eventsFile,
		options...,
	)
	if err != nil {
		return err
	}

	data, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		return err
	}
	return utils.WriteFile(utils.DemoReportFilePath(filePath), data, 0o644)
}

// ValidateReportFile returns an error if the report file filePath is missing or unreadable, if its format version is
// older than FormatVersion, or if it was prepared from an events file older than the embedded widget's format version.
// Writing such a report again would give the same report; a missing, unreadable or outdated one is written again.
func ValidateReportFile(filePath string) error {
	data, err := os.ReadFile(filePath) //nolint:gosec // Reading the file the caller names is the purpose.
	if err != nil {
		return err
	}
	var versions struct {
		FormatVersion       int `json:"formatVersion"`
		EventsFormatVersion int `json:"eventsFormatVersion"`
	}
	if err := json.Unmarshal(data, &versions); err != nil {
		return err
	}
	if versions.FormatVersion < FormatVersion {
		return fmt.Errorf("the report has format version %d, this program writes %d", versions.FormatVersion,
			FormatVersion)
	}
	if versions.EventsFormatVersion < widget.FormatVersion {
		return fmt.Errorf("the report is of an events file with format version %d, the widget writes %d",
			versions.EventsFormatVersion, widget.FormatVersion)
	}
	return nil
}
