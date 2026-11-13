package cmd

import (
	"fmt"
	"path/filepath"
	"time"

	"github.com/spf13/cobra"

	"github.com/backbone81/bar-replay-analytics-go/internal/report"
)

// The flags of replayReportCmd.
var (
	replayReportOpeningLength         time.Duration
	replayReportOpeningIntervalLength time.Duration
	replayReportIntervalLength        time.Duration
)

// replayReportCmd writes the reports of replays.
var replayReportCmd = &cobra.Command{
	Use:         "report <replay>...",
	Annotations: map[string]string{needsBarDataDir: "true"}, //nolint:goconst
	Short:       "Write the reports of replays as JSON next to them",
	Long: `Writes each game aggregated for coaching to <replay name>_report.json next to its replay, replacing an
existing report. Needs <replay name>_events.json from 'replay play'; a replay without one is skipped. A report of an
events file older than the analytics widget built into this program is written, with a note to play the replay
again. Each report holds its formatVersion and the eventsFormatVersion of its events file.

<replay> is a path to a .sdfz file or to a folder, or the name of a file or folder in the
demos/ folder of the BAR data directory. A folder stands for the .sdfz files directly in it (not in its subfolders),
such as "leaderboard" for demos/leaderboard/.`,
	Args: cobra.MinimumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		filePaths, err := replayPaths(args)
		if err != nil {
			return err
		}
		counts := statusCounts{}
		for _, filePath := range filePaths {
			if err := cmd.Context().Err(); err != nil {
				return err
			}
			status, description := reportReplay(filePath)
			counts[status]++
			printReplayLine(status, filepath.Base(filePath), description)
		}
		if len(filePaths) > 1 {
			counts.print(statusReported, statusSkipped, statusFailed)
		}
		if counts[statusFailed] != 0 {
			return fmt.Errorf("%d of %d replays failed", counts[statusFailed], len(filePaths))
		}
		return nil
	},
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
