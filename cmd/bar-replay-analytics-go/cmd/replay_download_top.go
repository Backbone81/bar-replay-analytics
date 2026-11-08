package cmd

import (
	"path/filepath"

	"github.com/spf13/cobra"

	"github.com/backbone81/bar-replay-analytics-go/internal/barlive"
	"github.com/backbone81/bar-replay-analytics-go/internal/replaydownloader"
)

// The flags of replayDownloadTopCmd.
var (
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
	Long: `Downloads the replays of the games the --top players of a leaderboard of the active season played in about the
last --days (bar-db selects whole days), of the leaderboard's game type only: bar-db's preset must fit, and the
allyteams must have the shape the lobby server decides the game type by (duel 1v1, small-team two allyteams of up to 5
players, large-team two of more, ffa more than two of 1 player). Games with AIs and aborted games are left out; unrated
ones are included, as bar-db does not tell them apart. An error stops the downloads of that player only; the command
goes on with the next one and fails at the end.

--map takes the exact mapname of the start script ('replay script' shows it), version included, such as "Argent Strata
1.1". Repeat it for several maps.

The replays go to demos/leaderboard/ in the BAR data directory, which the BAR lobby's replay list does not show, under
the file names of the replay storage. index.json there records which file holds which replay, so a new run downloads
only the missing ones and notices files added or removed by hand. A download is written to a .part file first, so an
interrupted run leaves no partial replay. --dry-run logs the replays it would download.`,
	Args: cobra.NoArgs,
	RunE: func(cmd *cobra.Command, _ []string) error {
		options := []replaydownloader.Option{
			replaydownloader.WithLogger(log),
			replaydownloader.WithDir(filepath.Join(barDataDir, "demos", "leaderboard")),
			replaydownloader.WithPlayers(replayDownloadTopTop),
			replaydownloader.WithDays(replayDownloadTopDays),
			replaydownloader.WithGameType(toGameType(replayDownloadTopGameType)),
		}
		for _, mapName := range replayDownloadTopMaps {
			options = append(options, replaydownloader.WithMap(mapName))
		}
		if replayDownloadTopDryRun {
			options = append(options, replaydownloader.WithDryRun())
		}
		downloader, err := replaydownloader.New(options...)
		if err != nil {
			return err
		}
		defer func() {
			if err := downloader.Close(); err != nil {
				log.Error(err.Error())
			}
		}()

		if err := downloader.Run(cmd.Context()); err != nil {
			return err
		}
		return nil
	},
}

func init() {
	replayCmd.AddCommand(replayDownloadTopCmd)
	flags := replayDownloadTopCmd.Flags()
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
		"Log the missing replays without downloading them.",
	)
}

func toGameType(gameType string) barlive.GameType {
	switch gameType {
	case "duel":
		return barlive.GameTypeDuel
	case "small-team":
		return barlive.GameTypeSmallTeam
	case "large-team":
		return barlive.GameTypeLargeTeam
	case "ffa":
		return barlive.GameTypeFFA
	default:
		return barlive.GameType(gameType)
	}
}
