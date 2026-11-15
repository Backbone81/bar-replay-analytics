package cmd

import (
	"fmt"
	"time"

	"github.com/spf13/cobra"

	"github.com/backbone81/bar-replay-analytics/internal/report"
	"github.com/backbone81/bar-replay-analytics/internal/utils"
)

// The flags of replayReportCmd.
var (
	replayReportOpeningLength         time.Duration
	replayReportOpeningIntervalLength time.Duration
	replayReportIntervalLength        time.Duration
	replayReportFilterMaps            []string
	replayReportFilterPlayers         []string
)

// replayReportCmd writes the reports of replays.
var replayReportCmd = &cobra.Command{
	Use:         "report <replay>...",
	Annotations: map[string]string{needsBarDataDir: "true"}, //nolint:goconst
	Short:       "Write the reports of replays as JSON next to them",
	Long: `Writes each game aggregated for coaching to <replay name>_report.json next to its replay, replacing an
existing report. Needs a current <replay name>_events.json from 'replay play'. Each report holds its formatVersion and
the eventsFormatVersion of its events file.

A replay whose report is current is not reported again: the report has the format version of this program and was
prepared from an events file of the format version of the analytics widget built into this program. So the same
command, such as 'replay report leaderboard', reports only the new replays, and after an update of this program the
ones with outdated reports.

--filter-map and --filter-player select the replays to report; a replay must match both if both are given.

Stops at the first replay that fails, such as one without a current events file: play it back first.

<replay> is a path to a .sdfz file or to a folder; a relative path is relative to the demos/ folder of the BAR data
directory. A folder stands for the .sdfz files directly in it (not in its subfolders), such as "leaderboard" for
demos/leaderboard/.`,
	Args: cobra.MinimumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		start := time.Now()

		replayFilePaths, err := utils.ResolveReplayPaths(barDataDir, args)
		if err != nil {
			return err
		}

		filter := utils.Filter{
			MapNames:    replayReportFilterMaps,
			PlayerNames: replayReportFilterPlayers,
		}

		var reported, skipped int
		for _, filePath := range replayFilePaths {
			if err := cmd.Context().Err(); err != nil {
				return err
			}

			shortFilePath := utils.RelativePath(utils.DemosDir(barDataDir), filePath)
			replayInfo, err := utils.ReadReplayInfo(filePath)
			if err != nil {
				return fmt.Errorf("reading replay %s: %w", shortFilePath, err)
			}

			if !filter.Matches(replayInfo) {
				continue
			}

			date := replayInfo.StartTime.Format("2006.01.02")
			if report.ValidateReportFile(utils.DemoReportFilePath(filePath)) == nil {
				log.Info("Skipping replay. Report up to date.", "file-path", shortFilePath, "date", date)
				skipped++
				continue
			}

			log.Info("Writing report", "file-path", shortFilePath, "date", date)
			if err := report.WriteReport(filePath,
				report.WithOpeningLength(replayReportOpeningLength),
				report.WithOpeningIntervalLength(replayReportOpeningIntervalLength),
				report.WithIntervalLength(replayReportIntervalLength),
			); err != nil {
				return fmt.Errorf("writing the report of %s: %w", shortFilePath, err)
			}
			reported++
		}

		log.Info("Finished all replays",
			"total", reported+skipped,
			"reported", reported,
			"skipped", skipped,
			"duration", time.Since(start).Round(time.Second),
		)
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
	flags.StringArrayVar(
		&replayReportFilterMaps,
		"filter-map",
		nil,
		"Select the replays whose mapname contains this text, ignoring case, so \"argent strata\" selects every "+
			"version of Argent Strata. Repeat for several maps.",
	)
	flags.StringArrayVar(
		&replayReportFilterPlayers,
		"filter-player",
		nil,
		"Select the replays in which a player or an AI (not a spectator) has this name, ignoring case. Repeat for "+
			"several players.",
	)
}
