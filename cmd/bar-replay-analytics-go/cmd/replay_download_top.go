package cmd

import (
	"context"
	"errors"
	"fmt"
	"strings"
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
	replayDownloadTopMaps     []string
	replayDownloadTopDryRun   bool
)

// replayDownloadTopCmd downloads the recent replays of the best players of a leaderboard.
var replayDownloadTopCmd = &cobra.Command{
	Use:         "download-top",
	Annotations: map[string]string{needsBarDataDir: "true"}, //nolint:goconst
	Short:       "Download the recent replays of the best players of a leaderboard",
	Long: `Downloads the replays of the games the --top players of a leaderboard played in the last --days, of the
leaderboard's game type only: bar-db's preset must fit, and the allyteams must have the shape the lobby server decides
the game type by (duel 1v1, small-team two allyteams of up to 5 players, large-team two of more, ffa more than two of
1 player). Games with AIs and aborted games are left out; unrated ones are included, as bar-db does not tell them
apart.

--map takes the exact mapname of the start script ('replay script' shows it), version included, such as "Argent Strata
1.1". Repeat it for several maps.

The replays keep the file names of the replay storage. The default --dir, demos/leaderboard/, is not shown in the BAR
lobby's replay list. index.json there records which file holds which replay, so a new run downloads only the missing
ones and notices files added or removed by hand. A download is written to a .part file first, so an interrupted run
leaves no partial replay.`,
	Args: cobra.NoArgs,
	RunE: func(cmd *cobra.Command, _ []string) error {
		if err := checkDownloadTopFlags(); err != nil {
			return err
		}

		client := barlive.NewClient(barlive.WithOnRequest(printRequest))
		selection, err := selectTopPlayers(cmd.Context(), client)
		if err != nil {
			return err
		}

		downloader, err := newReplayDownloader(client, replayDownloadTopDryRun)
		if err != nil {
			return err
		}

		for i, player := range selection.players {
			err = selection.downloadPlayerReplays(cmd.Context(), downloader, i+1, player)
			if err != nil {
				break
			}
		}
		return downloader.finish(err)
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
	flags.StringArrayVar(
		&replayDownloadTopMaps,
		"map",
		nil,
		"Select the games on this map, by its exact mapname, such as \"Argent Strata 1.1\". Repeat for several maps.",
	)
	flags.BoolVar(
		&replayDownloadTopDryRun,
		"dry-run",
		false,
		"List the missing replays without downloading them.",
	)
}

// checkDownloadTopFlags checks the flags of replayDownloadTopCmd which need no request.
func checkDownloadTopFlags() error {
	if replayDownloadTopTop < 1 || replayDownloadTopTop > 100 {
		return fmt.Errorf("--top: got %d, want 1 to 100", replayDownloadTopTop)
	}
	if replayDownloadTopDays < 1 {
		return fmt.Errorf("--days: got %d, want >= 1", replayDownloadTopDays)
	}
	for _, mapName := range replayDownloadTopMaps {
		if strings.TrimSpace(mapName) == "" {
			return errors.New("--map: got an empty map name")
		}
	}
	return nil
}

// topPlayers is what the flags of replayDownloadTopCmd select: the best players of a leaderboard, and which of their
// games to download.
type topPlayers struct {
	// gameType is the game type of the leaderboard and the games.
	gameType barlive.GameType

	// since is the time from which on games are downloaded.
	since time.Time

	// maps are the maps of the games, by mapname, or nil for any map.
	maps []string

	// players are the best players, best first.
	players []barlive.LeaderboardPlayer
}

// selectTopPlayers returns the best players the flags select, and prints the selection.
func selectTopPlayers(ctx context.Context, client *barlive.Client) (*topPlayers, error) {
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

	selection := &topPlayers{
		gameType: gameType,
		since:    time.Now().Add(-time.Duration(replayDownloadTopDays) * 24 * time.Hour),
		maps:     replayDownloadTopMaps,
		players:  leaderboard.Players[:min(replayDownloadTopTop, len(leaderboard.Players))],
	}

	printField("season", season.Season)
	printField("game type", gameTypeFlagName(gameType))
	printField("since", selection.since.UTC().Format("2006-01-02 15:04 MST"))
	if len(selection.maps) != 0 {
		printField("maps", strings.Join(selection.maps, ", "))
	}
	printField("players", len(selection.players))
	return selection, nil
}

// downloadPlayerReplays has downloader download the selected games of player, who has rank on the leaderboard, one
// after the other as bar-db returns them, and prints the player and their number of games.
func (t *topPlayers) downloadPlayerReplays(
	ctx context.Context, downloader *replayDownloader, rank int, player barlive.LeaderboardPlayer,
) error {
	fmt.Println()
	printField("player", fmt.Sprintf("%d. %s", rank, player.Name))

	count := 0
	for replay, err := range downloader.client.RecentReplays(ctx, player.Name, t.gameType, t.since, t.maps) {
		if err != nil {
			return fmt.Errorf("replays of %s: %w", player.Name, err)
		}
		count++
		downloader.download(ctx, replayToDownload{ID: replay.ID, Description: describeReplay(&replay)})
	}
	printField("replays", count)
	return nil
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
