package cmd

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/spf13/cobra"

	"github.com/backbone81/bar-replay-analytics-go/internal/demo"
	"github.com/backbone81/bar-replay-analytics-go/internal/playback"
)

// advisorWidgetFile is the file name of the bar-advisor widget, whose [reminder] lines in the engine log show what it
// would have said in the game.
const advisorWidgetFile = "snd_advisor.lua"

// The flags of replayPlayCmd.
var (
	replayPlayParallel         int
	replayPlayProgressInterval time.Duration
	replayPlayEngineThreads    int
	replayPlayForce            bool
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

A replay whose events file is current is not played again unless --force is given: the file ends with the game over
and has the format version of the analytics widget built into this program, which increases whenever the events it
writes change. So the same command, such as 'replay play leaderboard', plays only the new replays, and after an
update of this program the ones with outdated events.

A playback is stopped if its output has not grown for a minute, and fails unless its events file ends with the game
over. A replay without a game length in its header (0:00 in the lobby: the engine crashed or the game was not ended)
is skipped.

` + replayArgsHelp,
	Args: cobra.MinimumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		if replayPlayParallel < 0 {
			return fmt.Errorf("--parallel: got %d, want >= 0", replayPlayParallel)
		}
		if replayPlayParallel == 0 {
			replayPlayParallel = playback.DefaultParallel()
		}
		if replayPlayEngineThreads < 0 {
			return fmt.Errorf("--engine-threads: got %d, want >= 0", replayPlayEngineThreads)
		}
		if replayPlayProgressInterval <= 0 {
			return fmt.Errorf("--progress-interval: got %v, want > 0", replayPlayProgressInterval)
		}

		filePaths, err := replayPaths(args)
		if err != nil {
			return err
		}

		player, err := playback.NewPlayer(
			barDataDir,
			playback.WithProgressInterval(replayPlayProgressInterval),
			playback.WithWorkerThreadCount(replayPlayEngineThreads),
			playback.WithOnProgress(printProgress),
			playback.WithOnDownload(printDownload),
		)
		if err != nil {
			return err
		}
		printField("data dir", player.DataDir())
		printField("analytics", "built in")
		advisorDir, found, err := playback.FindWidgetDir(filepath.Join(player.DataDir(), "LuaUI", "Widgets"),
			advisorWidgetFile)
		if err != nil {
			return err
		}
		if !found {
			advisorDir = "not installed (no " + advisorWidgetFile + " in a folder of LuaUI/Widgets/)"
		}
		printField("bar-advisor", advisorDir)
		printField("parallel", replayPlayParallel)
		fmt.Println()

		counts := playReplays(cmd.Context(), player, filePaths)

		counts.print(statusPlayed, statusCurrent, statusSkipped, statusFailed)
		if err := cmd.Context().Err(); err != nil {
			return err
		}
		if counts[statusFailed] != 0 {
			return fmt.Errorf("%d of %d replays failed", counts[statusFailed], len(filePaths))
		}
		return nil
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
	flags.DurationVar(
		&replayPlayProgressInterval,
		"progress-interval",
		playback.DefaultConfig.ProgressInterval,
		"How often to print the progress of each running playback.",
	)
	flags.IntVar(
		&replayPlayEngineThreads,
		"engine-threads",
		playback.DefaultConfig.WorkerThreadCount,
		"The number of threads of each engine's thread pool (WorkerThreadCount), the main thread included. 0 leaves "+
			"the engine's default of one per CPU, which is slower.",
	)
	flags.BoolVar(
		&replayPlayForce,
		"force",
		false,
		"Play replays back also if their events file is current.",
	)
}

// playReplays plays the replays filePaths back with --parallel at a time, prints a line for each and returns how
// many ended with each status (see playReplay).
func playReplays(ctx context.Context, player *playback.Player, filePaths []string) statusCounts {
	queue := make(chan string)
	var mutex sync.Mutex
	counts := statusCounts{}

	var waitGroup sync.WaitGroup
	for range min(replayPlayParallel, len(filePaths)) {
		waitGroup.Go(func() {
			for filePath := range queue {
				status, description := playReplay(ctx, player, filePath)

				mutex.Lock()
				counts[status]++
				printReplayLine(status, filepath.Base(filePath), description)
				mutex.Unlock()
			}
		})
	}
	for _, filePath := range filePaths {
		queue <- filePath
	}
	close(queue)
	waitGroup.Wait()
	return counts
}

// playReplay prepares and plays the replay filePath back, and returns its status and a description of the outcome.
// The status is "played", "current" (its events file is current and --force was not given), "skipped" (no game
// length) or "failed".
func playReplay(ctx context.Context, player *playback.Player, filePath string) (string, string) {
	if err := ctx.Err(); err != nil {
		return statusFailed, err.Error()
	}
	if !replayPlayForce {
		status, err := playback.CheckEventsFile(filePath)
		if err != nil {
			return statusFailed, err.Error()
		}
		if status.Current {
			return statusCurrent, status.Reason
		}
	}
	// Before Prepare, so nothing is downloaded for a skipped replay.
	demoFile, err := demo.ReadFile(filePath, demo.ReadUntilHeader)
	if err != nil {
		return statusFailed, err.Error()
	}
	if demoFile.FileHeader.GameTime == 0 {
		return statusSkipped, "no game length: the engine crashed or the game was not ended"
	}
	replay, err := player.Prepare(ctx, filePath)
	if err != nil {
		return statusFailed, err.Error()
	}
	length := "unknown length"
	if replay.GameTime > 0 {
		length = formatGameTime(replay.GameTime)
	}
	printStatus("starting", replay.Name, fmt.Sprintf("engine %s, %s, %s, %s", replay.EngineVersion, replay.GameType,
		replay.MapName, length))

	result, err := player.Play(ctx, log, replay)
	if err != nil {
		return statusFailed, err.Error()
	}
	description := describeResult(result)
	if !result.GameOver() {
		return statusFailed, description
	}
	return statusPlayed, description
}

// describeResult returns how a playback ended, its number of events, how long it took and where its files are.
func describeResult(result *playback.Result) string {
	var parts []string
	switch result.End {
	case nil:
		parts = append(parts, "events file incomplete")
	default:
		parts = append(parts, fmt.Sprintf("%s at %s, %d events", result.End.Reason,
			formatGameTime(playback.Progress{Frame: result.End.Frame}.GameTime()), result.Events))
	}
	if result.StopReason != "" {
		parts = append(parts, "stopped: "+result.StopReason)
	}
	if result.ExitErr != nil {
		parts = append(parts, "engine: "+result.ExitErr.Error())
	}
	parts = append(parts, fmt.Sprintf("%.0f s", result.Duration.Seconds()))
	if !result.GameOver() {
		parts = append(parts, "log "+result.LogFile)
	}
	return strings.Join(parts, ", ")
}

// printProgress prints the progress of a playback to stderr; it is the Config.OnProgress of the player.
func printProgress(progress playback.Progress) {
	elapsed := fmt.Sprintf("(%.0f s)", progress.Elapsed.Seconds())
	if progress.Loading {
		printStatus("loading", progress.Replay.Name, elapsed)
		return
	}
	gameTime := formatGameTime(progress.GameTime())
	if percent, known := progress.Percent(); known {
		printStatus("playing", progress.Replay.Name, fmt.Sprintf("%3.0f%%  %s of %s  %s", percent, gameTime,
			formatGameTime(progress.Replay.GameTime), elapsed))
		return
	}
	printStatus("playing", progress.Replay.Name, fmt.Sprintf("game time %s  %s", gameTime, elapsed))
}

// printDownload prints a line for a download to stderr; it is the Config.OnDownload of the player.
func printDownload(description string) {
	printStatus("download", description, "")
}

// printReplayLine prints the outcome for one replay to stdout.
func printReplayLine(status string, name string, description string) {
	fmt.Println(strings.TrimRight(fmt.Sprintf("%-10s  %s  %s", status, name, description), " "))
}

// printStatus prints a line about a running replay or download to stderr, aligned like the lines of
// printReplayLine, so stdout holds only the results.
func printStatus(status string, name string, description string) {
	_, _ = fmt.Fprintln(os.Stderr, strings.TrimRight(fmt.Sprintf("%-10s  %s  %s", status, name, description), " "))
}

// formatGameTime returns a game time as minutes:seconds.
func formatGameTime(gameTime time.Duration) string {
	seconds := int(gameTime.Seconds())
	return fmt.Sprintf("%d:%02d", seconds/60, seconds%60)
}
