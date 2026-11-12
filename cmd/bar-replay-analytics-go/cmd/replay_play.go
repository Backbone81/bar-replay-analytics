package cmd

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/backbone81/bar-replay-analytics-go/internal/replayplayer"
)

// The flags of replayPlayCmd.
var (
	replayPlayParallel      int
	replayPlayEngineThreads int
	replayPlayMaps          []string
	replayPlayPlayers       []string
	replayPlayMaxReplays    int
)

// replayPlayCmd plays replays back in a headless engine.
var replayPlayCmd = &cobra.Command{
	Use:         "play <replay>...",
	Annotations: map[string]string{needsBarDataDir: "true"}, //nolint:goconst
	Short:       "Play replays back in a headless engine at maximum speed",
	Long: `Plays replays back in spring-headless at maximum speed with the analytics widget built into this program,
which replaces one installed by install-widget. Writes <replay name>_events.json (the widget's events) and
<replay name>.log (the engine's output) next to each replay. Other widgets in LuaUI/Widgets/ run too, so their log
lines show what they would have done in the game, such as the [reminder] lines of bar-advisor.

Missing engines, games and maps are downloaded into the BAR data directory as the BAR lobby does. Each playback has a
temporary write dir, so the data directory's infolog.txt, settings and LuaUI/Config/ stay unchanged.

A replay whose events file is current is not played again: the file ends with the game over and has the format version
of the analytics widget built into this program, which increases whenever the events it writes change. So the same
command, such as 'replay play leaderboard', plays only the new replays, and after an update of this program the ones
with outdated events.

--map and --player select the replays to play back, before anything is downloaded; a replay must match both if both
are given. --max-replays stops after that many replays, in file path order, counting only those that are played.

<replay> is a path to a .sdfz file or to a folder; a relative path is relative to the demos/ folder of the BAR data
directory. A folder stands for the .sdfz files directly in it (not in its subfolders), such as "leaderboard" for
demos/leaderboard/.`,
	Args: cobra.MinimumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		if replayPlayParallel < 0 {
			return fmt.Errorf("--parallel: got %d, want >= 0", replayPlayParallel)
		}
		if replayPlayEngineThreads < 0 {
			return fmt.Errorf("--engine-threads: got %d, want >= 0", replayPlayEngineThreads)
		}
		if replayPlayMaxReplays < 0 {
			return fmt.Errorf("--max-replays: got %d, want >= 0", replayPlayMaxReplays)
		}

		assetManagerConfig := replayplayer.DefaultAssetManagerConfig
		assetManagerConfig.Logger = log
		assetManagerConfig.DataDir = barDataDir
		assetManager, err := replayplayer.NewAssetManager(assetManagerConfig)
		if err != nil {
			return err
		}

		batchPlayer, err := replayplayer.NewBatchPlayer(
			replayplayer.WithLogger(log),
			replayplayer.WithDataDir(barDataDir),
			replayplayer.WithAssetManager(assetManager),
			replayplayer.WithWorkerThreadCount(replayPlayEngineThreads),
			replayplayer.WithConcurrentReplays(replayPlayParallel),
			replayplayer.WithFilter(replayplayer.Filter{
				MapNames:    replayPlayMaps,
				PlayerNames: replayPlayPlayers,
			}),
			replayplayer.WithMaxReplays(replayPlayMaxReplays),
		)
		if err != nil {
			return err
		}
		return batchPlayer.Run(cmd.Context(), args...)
	},
}

func init() {
	replayCmd.AddCommand(replayPlayCmd)
	flags := replayPlayCmd.Flags()
	flags.IntVar(
		&replayPlayParallel,
		"parallel",
		0,
		"The number of replays to play back at the same time. 0 selects one per 2 CPUs, but at most one per 6 GB of "+
			"available memory: an engine needs about 5 GB.",
	)
	flags.IntVar(
		&replayPlayEngineThreads,
		"engine-threads",
		replayplayer.DefaultConfig.WorkerThreadCount,
		"The number of threads of each engine's thread pool (WorkerThreadCount), the main thread included. 0 leaves "+
			"the engine's default of one per CPU, which is slower.",
	)
	flags.StringArrayVar(
		&replayPlayMaps,
		"map",
		nil,
		"Select the replays whose mapname contains this text, ignoring case, so \"argent strata\" selects every "+
			"version of Argent Strata. Repeat for several maps.",
	)
	flags.StringArrayVar(
		&replayPlayPlayers,
		"player",
		nil,
		"Select the replays in which a player or an AI (not a spectator) has this name, ignoring case. Repeat for "+
			"several players.",
	)
	flags.IntVar(
		&replayPlayMaxReplays,
		"max-replays",
		0,
		"Play back at most this many replays: the first ones, in file path order, that are selected and whose events "+
			"file is not current. 0 means no limit.",
	)
}
