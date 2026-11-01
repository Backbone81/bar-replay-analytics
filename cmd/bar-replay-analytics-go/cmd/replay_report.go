package cmd

import (
	"time"

	"github.com/spf13/cobra"

	"github.com/backbone81/bar-replay-analytics-go/internal/demo"
	"github.com/backbone81/bar-replay-analytics-go/internal/report"
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
	Long: `Prints the report of a replay as JSON: the data of the replay aggregated for analyzing and coaching the game,
so that it can be read as it is. So far it holds the metadata of the game and the economy of every player, over the
whole game and in intervals: of --opening-interval-length in the first --opening-length of the game, where economies
grow fastest and build orders differ most, and of --interval-length after that. The intervals are the same in every
report, so games can be compared interval by interval. Times are game time in minutes:seconds.

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
		result, err := report.NewReport(
			replay,
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

func init() {
	replayCmd.AddCommand(replayReportCmd)
	flags := replayReportCmd.Flags()
	flags.DurationVar(
		&replayReportOpeningLength,
		"opening-length",
		report.DefaultConfig.OpeningLength,
		"The length of the opening, which the report aggregates in intervals of --opening-interval-length. "+
			"0 aggregates the whole game in intervals of --interval-length.",
	)
	flags.DurationVar(
		&replayReportOpeningIntervalLength,
		"opening-interval-length",
		report.DefaultConfig.OpeningIntervalLength,
		"The length of the intervals in the opening. Rounded up to whole statistics periods of 15 s.",
	)
	flags.DurationVar(
		&replayReportIntervalLength,
		"interval-length",
		report.DefaultConfig.IntervalLength,
		"The length of the intervals after the opening, such as 1m or 5m. Rounded up to whole statistics periods "+
			"of 15 s.",
	)
}
