package cmd

import (
	"context"
	"fmt"
	"os"
	"strings"
	"text/tabwriter"
	"time"

	"github.com/spf13/cobra"

	"github.com/backbone81/bar-replay-analytics-go/internal/barlive"
)

// The flags of replayDownloadTopCmd.
var (
	replayDownloadTopSeason   int
	replayDownloadTopGameType string
	replayDownloadTopTop      int
	replayDownloadTopDays     int
	replayDownloadTopDryRun   bool
)

// replayDownloadTopCmd downloads the recent replays of the best players of a leaderboard.
var replayDownloadTopCmd = &cobra.Command{
	Use:   "download-top --game-type <game-type>",
	Short: "Download the recent replays of the best players of a leaderboard",
	Long: `Downloads the replays of the games which the best players of a leaderboard played in the last days, for the
game type of the leaderboard only. A game counts as that game type if bar-db's preset fits and its allyteams have the
size the lobby server decides the game type by: duel 1v1, small-team two allyteams of up to 5 players, large-team two
allyteams of more, ffa more than two allyteams of 1 player. Games with AIs and aborted games are left out; unrated
games are included, as bar-db does not tell them apart.

The replays are stored under the file names of the replay storage in demos/leaderboard/ of the BAR data directory
(see --dir). The BAR lobby lists only the replays directly in demos/, so these do not crowd its replay list.

Running it again downloads only the replays the directory does not hold yet. The file index.json in the directory
records which file holds which replay; a replay counts as present if it is recorded and its file exists. Otherwise its
file name is looked up in bar-db: a file of that name which holds the replay's game is recorded without a download,
any other replay is downloaded. So files added or removed by hand are noticed, and entries of removed files are
dropped. A download goes to the file name with .part appended first, which is renamed only once it is complete and
holds the replay's game, so an interrupted run leaves no partial replay under the replay's name.`,
	Args: cobra.NoArgs,
	RunE: func(cmd *cobra.Command, _ []string) error {
		if err := checkDownloadTopFlags(); err != nil {
			return err
		}

		client := barlive.NewClient()
		replays, err := findTopReplays(cmd.Context(), client)
		if err != nil {
			return err
		}

		fmt.Println()
		return downloadReplays(cmd.Context(), client, replays, replayDownloadTopDryRun)
	},
}

func init() {
	replayCmd.AddCommand(replayDownloadTopCmd)
	addReplayDownloadFlags(replayDownloadTopCmd)
	flags := replayDownloadTopCmd.Flags()
	flags.IntVar(
		&replayDownloadTopSeason,
		"season",
		0,
		"The season of the leaderboard, see 'leaderboard seasons'. 0 selects the active season.",
	)
	flags.StringVar(
		&replayDownloadTopGameType,
		"game-type",
		"duel",
		"The game type of the leaderboard and the games: duel, small-team, large-team or ffa.",
	)
	flags.IntVar(
		&replayDownloadTopTop,
		"top",
		10,
		"The number of best players, at most 100.",
	)
	flags.IntVar(
		&replayDownloadTopDays,
		"days",
		7,
		"Select the games which started in the last days.",
	)
	flags.BoolVar(
		&replayDownloadTopDryRun,
		"dry-run",
		false,
		"List the missing replays without downloading them.",
	)
	_ = replayDownloadTopCmd.MarkFlagRequired("game-type")
}

// checkDownloadTopFlags checks the flags of replayDownloadTopCmd which need no request.
func checkDownloadTopFlags() error {
	if replayDownloadTopTop < 1 || replayDownloadTopTop > 100 {
		return fmt.Errorf("--top: got %d, want 1 to 100", replayDownloadTopTop)
	}
	if replayDownloadTopDays < 1 {
		return fmt.Errorf("--days: got %d, want >= 1", replayDownloadTopDays)
	}
	return nil
}

// findTopReplays returns the recent replays of the best players the flags select, each once, and prints the players
// with their number of replays.
func findTopReplays(ctx context.Context, client *barlive.Client) ([]replayToDownload, error) {
	season, err := client.LeaderboardSeason(ctx, replayDownloadTopSeason)
	if err != nil {
		return nil, err
	}

	gameType, err := findGameType(season, replayDownloadTopGameType)
	if err != nil {
		return nil, err
	}

	if _, found := gameType.ReplayPreset(); !found {
		return nil, fmt.Errorf("--game-type %q: cannot select its replays", replayDownloadTopGameType)
	}

	leaderboard, err := client.Leaderboard(ctx, season.Season, gameType)
	if err != nil {
		return nil, err
	}

	players := leaderboard.Players[:min(replayDownloadTopTop, len(leaderboard.Players))]
	since := time.Now().Add(-time.Duration(replayDownloadTopDays) * 24 * time.Hour)

	printField("season", season.Season)
	printField("game type", gameTypeFlagName(gameType))
	printField("since", since.UTC().Format("2006-01-02 15:04 MST"))
	fmt.Println()

	writer := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	_, _ = fmt.Fprintln(writer, "rank\tplayer\treplays")

	var replays []replayToDownload
	seen := make(map[string]bool)
	for i, player := range players {
		count := 0
		for replay, err := range client.RecentReplays(ctx, player.Name, gameType, since) {
			if err != nil {
				_ = writer.Flush()
				return nil, fmt.Errorf("replays of %s: %w", player.Name, err)
			}
			count++
			if !seen[replay.ID] {
				seen[replay.ID] = true
				replays = append(replays, replayToDownload{ID: replay.ID, Description: describeReplay(&replay)})
			}
		}
		_, _ = fmt.Fprintf(writer, "%d\t%s\t%d\n", i+1, player.Name, count)
	}
	return replays, writer.Flush()
}

// describeReplay returns the start time, map and players of replay.
func describeReplay(replay *barlive.ReplaySummary) string {
	allyTeams := make([]string, 0, len(replay.AllyTeams))
	for _, allyTeam := range replay.AllyTeams {
		if len(allyTeam.Players) == 0 {
			continue
		}
		names := make([]string, len(allyTeam.Players))
		for i, player := range allyTeam.Players {
			names[i] = player.Name
		}
		allyTeams = append(allyTeams, strings.Join(names, ", "))
	}
	return fmt.Sprintf(
		"%s  %s  %s",
		replay.StartTime.UTC().Format("2006-01-02 15:04"),
		replay.Map.ScriptName,
		strings.Join(allyTeams, " vs "),
	)
}
