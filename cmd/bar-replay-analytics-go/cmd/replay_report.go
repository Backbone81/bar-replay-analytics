package cmd

import (
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/backbone81/bar-replay-analytics-go/internal/demo"
	"github.com/backbone81/bar-replay-analytics-go/internal/events"
	"github.com/backbone81/bar-replay-analytics-go/internal/playback"
	"github.com/backbone81/bar-replay-analytics-go/internal/report"
	"github.com/backbone81/bar-replay-analytics-go/internal/tdf"
)

// The flags of replayReportCmd.
var (
	replayReportOpeningLength         time.Duration
	replayReportOpeningIntervalLength time.Duration
	replayReportIntervalLength        time.Duration
)

// replayReportCmd prints the report of a replay.
var replayReportCmd = &cobra.Command{
	Use:   "report <replay>",
	Short: "Print the report of a replay as JSON",
	Long: `Prints the report of a replay as JSON: the game aggregated for coaching, readable as it is. It needs the events
file of the replay's playback (see "replay play").

<replay> is a path to a .sdfz file, or the name of a file in the demos/ folder of the BAR data directory.`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		filePath, err := replayPath(args[0])
		if err != nil {
			return err
		}
		replay, err := demo.ReadFile(filePath)
		if err != nil {
			return err
		}
		setupScript, err := tdf.Parse(replay.SetupScript)
		if err != nil {
			return fmt.Errorf("setup script: %w", err)
		}
		eventsFilePath, err := replayEventsPath(filePath)
		if err != nil {
			return err
		}
		eventsFile, err := events.ReadFile(eventsFilePath)
		if err != nil {
			return fmt.Errorf("events file of the replay's playback: %w", err)
		}
		result, err := report.NewReport(
			replay,
			setupScript,
			eventsFile,
			report.WithOpeningLength(replayReportOpeningLength),
			report.WithOpeningIntervalLength(replayReportOpeningIntervalLength),
			report.WithIntervalLength(replayReportIntervalLength),
		)
		if err != nil {
			return err
		}
		return writeJSON(result)
	},
}

// replayEventsPath returns the path of the events file the playback of the replay file filePath wrote, in the events
// folder in the BAR data directory (see playback.Player.EventsDir).
func replayEventsPath(filePath string) (string, error) {
	player, err := playback.NewPlayer(barDataDir)
	if err != nil {
		return "", err
	}
	replay := playback.Replay{
		Name: strings.TrimSuffix(filepath.Base(filePath), filepath.Ext(filePath)),
	}
	return filepath.Join(player.EventsDir(), replay.EventsFileName()), nil
}

func init() {
	replayCmd.AddCommand(replayReportCmd)
	flags := replayReportCmd.Flags()
	flags.DurationVar(
		&replayReportOpeningLength,
		"opening-length",
		report.DefaultConfig.OpeningLength,
		"The length of the opening: the first important minutes of the game. 0 for none.",
	)
	flags.DurationVar(
		&replayReportOpeningIntervalLength,
		"opening-interval-length",
		report.DefaultConfig.OpeningIntervalLength,
		"The length of the intervals in the opening.",
	)
	flags.DurationVar(
		&replayReportIntervalLength,
		"interval-length",
		report.DefaultConfig.IntervalLength,
		"The length of the intervals after the opening.",
	)
}
