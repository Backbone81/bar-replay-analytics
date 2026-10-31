package cmd

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"path/filepath"
	"strings"
	"sync"

	"github.com/spf13/cobra"

	"github.com/backbone81/bar-replay-analytics-go/internal/barlive"
)

// The flags shared by replayDownloadCmd and replayDownloadTopCmd.
var (
	replayDownloadDir      string
	replayDownloadParallel int
)

// replayDownloadCmd downloads replays by their IDs.
var replayDownloadCmd = &cobra.Command{
	Use:   "download <replay-id>...",
	Short: "Download replays by their IDs",
	Long: `Downloads the replays with the given IDs from the replay storage behind https://bar-rts.com, unless the
download directory already holds them. A replay ID is the game ID in hex, as in the URL of the replay's page:
https://www.bar-rts.com/replays/<replay-id>.

See 'replay download-top' for where the replays are stored and how existing ones are found.`,
	Args: cobra.MinimumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		replays := make([]replayToDownload, len(args))
		for i, id := range args {
			replays[i] = replayToDownload{ID: id}
		}
		return downloadReplays(cmd.Context(), barlive.NewClient(barlive.WithOnRequest(printRequest)), replays, false)
	},
}

func init() {
	replayCmd.AddCommand(replayDownloadCmd)
	addReplayDownloadFlags(replayDownloadCmd)
}

// addReplayDownloadFlags adds the flags shared by the download commands to cmd.
func addReplayDownloadFlags(cmd *cobra.Command) {
	cmd.Flags().StringVar(
		&replayDownloadDir,
		"dir",
		"",
		"The directory to store the replays in. Empty selects demos/leaderboard/ in the BAR data directory.",
	)
	cmd.Flags().IntVar(
		&replayDownloadParallel,
		"parallel",
		4,
		"The number of replays to download at the same time.",
	)
}

// replayToDownload is a replay a download command was asked for.
type replayToDownload struct {
	// ID is the replay ID, see barlive.Replay.ID.
	ID string

	// Description describes the game for the output, or is empty.
	Description string
}

// downloadDir returns the directory --dir selects.
func downloadDir() string {
	if replayDownloadDir != "" {
		return replayDownloadDir
	}
	return filepath.Join(barDataDir, "demos", "leaderboard")
}

// downloadReplays downloads the replays the download directory does not hold yet, and prints a line for each and a
// summary. With dryRun, it only lists them. The directory's barlive.ReplayIndex tells which it holds; it is saved
// even if downloads fail or the context is cancelled. It returns an error if any replay failed.
func downloadReplays(ctx context.Context, client *barlive.Client, replays []replayToDownload, dryRun bool) (err error) {
	if replayDownloadParallel < 1 {
		return fmt.Errorf("--parallel: got %d, want >= 1", replayDownloadParallel)
	}

	dir := downloadDir()
	replayIndex, err := barlive.OpenReplayIndex(dir)
	if err != nil {
		return err
	}

	var missing []replayToDownload
	for _, replay := range replays {
		if !replayIndex.Has(replay.ID) {
			missing = append(missing, replay)
		}
	}

	failed := 0
	if dryRun {
		for _, replay := range missing {
			printReplayLine("missing", replay.ID, replay.Description)
		}
	} else {
		defer func() {
			err = errors.Join(err, replayIndex.Save())
		}()
		failed = fetchMissing(ctx, client, replayIndex, missing)
	}

	fmt.Println()
	printField("dir", dir)
	printField("replays", len(replays))
	printField("present", len(replays)-len(missing))
	if dryRun {
		printField("missing", len(missing))
		return nil
	}
	printField("fetched", len(missing)-failed)
	printField("failed", failed)
	if failed != 0 {
		return fmt.Errorf("%d of %d replays failed", failed, len(missing))
	}
	return nil
}

// fetchMissing fetches replays into the directory of index with --parallel at a time, prints a line for each and
// returns the number of failed ones.
func fetchMissing(
	ctx context.Context, client *barlive.Client, index *barlive.ReplayIndex, replays []replayToDownload,
) int {
	queue := make(chan replayToDownload)
	var mutex sync.Mutex
	failed := 0

	var waitGroup sync.WaitGroup
	for range min(replayDownloadParallel, len(replays)) {
		waitGroup.Go(func() {
			for replay := range queue {
				result, err := index.Fetch(ctx, client, replay.ID)

				mutex.Lock()
				switch {
				case errors.Is(err, fs.ErrExist):
					failed++
					printReplayLine("failed", replay.ID, "another game's replay has the same file name: "+err.Error())
				case err != nil:
					failed++
					printReplayLine("failed", replay.ID, err.Error())
				case result.Downloaded:
					printReplayLine("downloaded", result.FileName, replay.Description)
				default:
					// The file was there, but not in the index.
					printReplayLine("indexed", result.FileName, replay.Description)
				}
				mutex.Unlock()
			}
		})
	}
	for _, replay := range replays {
		queue <- replay
	}
	close(queue)
	waitGroup.Wait()
	return failed
}

// printReplayLine prints the outcome for one replay.
func printReplayLine(status string, name string, description string) {
	fmt.Println(strings.TrimRight(fmt.Sprintf("%-10s  %s  %s", status, name, description), " "))
}
