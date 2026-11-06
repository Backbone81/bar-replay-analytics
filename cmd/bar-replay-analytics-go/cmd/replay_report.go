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

// replayReportCmd writes the report of a replay.
var replayReportCmd = &cobra.Command{
	Use:         "report <replay>",
	Annotations: map[string]string{needsBarDataDir: "true"}, //nolint:goconst
	Short:       "Write the report of a replay as JSON next to the replay",
	Long: `Writes the game aggregated for coaching to <replay name>_report.json next to the replay, replacing an existing
report. Needs <replay name>_events.json from 'replay play'.

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
		eventsFile, err := events.ReadFile(playback.EventsFilePath(filePath))
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
		reportFilePath := reportPath(filePath)
		if err := writeJSONFile(reportFilePath, result); err != nil {
			return err
		}
		printField("report", reportFilePath)
		return nil
	},
}

// reportPath returns the path of the report of the replay file filePath: <replay name>_report.json next to it, like
// the events file of its playback (see playback.EventsFilePath).
func reportPath(filePath string) string {
	return strings.TrimSuffix(filePath, filepath.Ext(filePath)) + "_report.json"
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
