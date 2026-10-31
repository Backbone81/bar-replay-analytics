package cmd

import (
	"fmt"
	"os"
	"text/tabwriter"

	"github.com/spf13/cobra"

	"github.com/backbone81/bar-replay-analytics-go/internal/barlive"
)

// leaderboardSeasonsCmd lists the seasons.
var leaderboardSeasonsCmd = &cobra.Command{
	Use:   "seasons",
	Short: "List the seasons and their game types",
	Long: `Lists the seasons for which the BAR lobby server has leaderboards, with the game types as --game-type names
them. The last season is the active one.`,
	Args: cobra.NoArgs,
	RunE: func(cmd *cobra.Command, _ []string) error {
		seasons, err := barlive.NewClient().LeaderboardSeasons(cmd.Context())
		if err != nil {
			return err
		}

		writer := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
		_, _ = fmt.Fprintln(writer, "season\tstate\tgame types")
		for i, season := range seasons {
			state := "ended"
			if i == len(seasons)-1 {
				state = "active"
			}
			_, _ = fmt.Fprintf(writer, "%d\t%s\t%s\n", season.Season, state, gameTypeFlagNames(season))
		}
		return writer.Flush()
	},
}

func init() {
	leaderboardCmd.AddCommand(leaderboardSeasonsCmd)
}
