package cmd

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/backbone81/bar-replay-analytics-go/internal/demo"
	"github.com/backbone81/bar-replay-analytics-go/internal/events"
	"github.com/backbone81/bar-replay-analytics-go/internal/playback"
	"github.com/backbone81/bar-replay-analytics-go/internal/replayplayer/widget"
	"github.com/backbone81/bar-replay-analytics-go/internal/report"
	"github.com/backbone81/bar-replay-analytics-go/internal/tdf"
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

` + replayArgsHelp,
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

// reportReplay writes the report of the replay filePath, and returns its status, "reported", "skipped" (no events
// file) or "failed", and a description of the outcome.
func reportReplay(filePath string) (string, string) {
	eventsFilePath := playback.EventsFilePath(filePath)
	if _, err := os.Stat(eventsFilePath); errors.Is(err, fs.ErrNotExist) {
		return statusSkipped, "no events file: play the replay back first (replay play)"
	}
	replay, err := demo.ReadFile(filePath, demo.ReadUntilEnd)
	if err != nil {
		return statusFailed, err.Error()
	}
	setupScript, err := tdf.Parse(replay.SetupScript)
	if err != nil {
		return statusFailed, fmt.Sprintf("setup script: %v", err)
	}
	eventsFile, err := events.ReadFile(eventsFilePath)
	if err != nil {
		return statusFailed, fmt.Sprintf("events file of the replay's playback: %v", err)
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
		return statusFailed, err.Error()
	}
	reportFilePath := reportPath(filePath)
	if err := writeJSONFile(reportFilePath, result); err != nil {
		return statusFailed, err.Error()
	}
	description := reportFilePath
	if eventsFile.FormatVersion < widget.FormatVersion {
		description += fmt.Sprintf(" (events file outdated: format %d, the widget writes %d; play the replay "+
			"again with replay play)", eventsFile.FormatVersion, widget.FormatVersion)
	}
	return statusReported, description
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
