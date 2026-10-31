package cmd

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"

	"github.com/backbone81/bar-replay-analytics-go/internal/barlive"
)

// leaderboardCmd groups the commands which show the leaderboards.
var leaderboardCmd = &cobra.Command{
	Use:   "leaderboard",
	Short: "Show the leaderboards of the BAR lobby server",
}

func init() {
	rootCmd.AddCommand(leaderboardCmd)
}

// gameTypeFlagName returns the value of --game-type which selects gameType: its name in lower case with hyphens for
// spaces, such as "small-team" for "Small Team".
func gameTypeFlagName(gameType barlive.GameType) string {
	return strings.ToLower(strings.ReplaceAll(string(gameType), " ", "-"))
}

// gameTypeFlagNames returns the values of --game-type for the game types of season, separated by commas.
func gameTypeFlagNames(season barlive.LeaderboardSeason) string {
	names := make([]string, len(season.GameTypes))
	for i, gameType := range season.GameTypes {
		names[i] = gameTypeFlagName(gameType)
	}
	return strings.Join(names, ", ")
}

// findGameType returns the game type of season which flagName selects, see gameTypeFlagName.
func findGameType(season barlive.LeaderboardSeason, flagName string) (barlive.GameType, error) {
	for _, gameType := range season.GameTypes {
		if gameTypeFlagName(gameType) == flagName {
			return gameType, nil
		}
	}
	return "", fmt.Errorf("--game-type %q: season %d has %s", flagName, season.Season, gameTypeFlagNames(season))
}
