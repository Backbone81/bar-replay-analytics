package cmd

import (
	"errors"
	"fmt"
	"os"
	"text/tabwriter"

	"github.com/spf13/cobra"

	"github.com/backbone81/bar-replay-analytics-go/internal/barlive"
)

// The flags of leaderboardShowCmd.
var (
	leaderboardShowSeason   int
	leaderboardShowGameType string
	leaderboardShowTop      int
)

// leaderboardShowCmd prints the best players of a season.
var leaderboardShowCmd = &cobra.Command{
	Use:   "show",
	Short: "Print the best players of the leaderboards of a season",
	Long: `Prints the best players of each leaderboard of a season, or of one with --game-type. The lobby server lists at
most 100 players, and only those whose rating changed in the 35 days before now, or before the end of an earlier
season.`,
	Args: cobra.NoArgs,
	RunE: func(cmd *cobra.Command, _ []string) error {
		if leaderboardShowTop < 1 {
			return fmt.Errorf("--top: got %d, want >= 1", leaderboardShowTop)
		}

		client := barlive.NewClient(barlive.WithOnRequest(printRequest))
		season, err := client.LeaderboardSeason(cmd.Context(), leaderboardShowSeason)
		if err != nil {
			return err
		}
		gameTypes := season.GameTypes
		if leaderboardShowGameType != "" {
			gameType, err := findGameType(season, leaderboardShowGameType)
			if err != nil {
				return err
			}
			gameTypes = []barlive.GameType{gameType}
		}

		leaderboards, err := client.Leaderboards(cmd.Context(), season.Season)
		if err != nil {
			return err
		}

		printField("season", season.Season)
		var errs []error
		for _, gameType := range gameTypes {
			leaderboard, found := barlive.FindLeaderboard(leaderboards, gameType)
			if !found {
				errs = append(errs, fmt.Errorf("season %d has no leaderboard for game type %q", season.Season, gameType))
				continue
			}
			fmt.Println()
			errs = append(errs, printLeaderboard(leaderboard, leaderboardShowTop))
		}
		return errors.Join(errs...)
	},
}

// printLeaderboard prints the top players of leaderboard.
func printLeaderboard(leaderboard barlive.Leaderboard, top int) error {
	printField("game type", gameTypeFlagName(leaderboard.Name))
	writer := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	_, _ = fmt.Fprintln(writer, "rank\trating\tid\tname")
	for i, player := range leaderboard.Players[:min(top, len(leaderboard.Players))] {
		_, _ = fmt.Fprintf(writer, "%d\t%.2f\t%d\t%s\n", i+1, player.Rating, player.ID, player.Name)
	}
	return writer.Flush()
}

func init() {
	leaderboardCmd.AddCommand(leaderboardShowCmd)
	leaderboardShowCmd.Flags().IntVar(
		&leaderboardShowSeason,
		"season",
		0,
		"The season, see 'leaderboard seasons'. 0 selects the active season.",
	)
	leaderboardShowCmd.Flags().StringVar(
		&leaderboardShowGameType,
		"game-type",
		"",
		"The game type, such as duel, small-team, large-team or ffa, see 'leaderboard seasons'. Empty selects all.",
	)
	leaderboardShowCmd.Flags().IntVar(
		&leaderboardShowTop,
		"top",
		20,
		"The number of players to print per game type.",
	)
}
