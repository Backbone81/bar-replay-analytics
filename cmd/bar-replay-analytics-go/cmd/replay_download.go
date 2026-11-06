package cmd

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"

	"github.com/backbone81/bar-replay-analytics-go/internal/barlive"
)

// replayDownloadDir is the --dir flag shared by replayDownloadCmd and replayDownloadTopCmd.
var replayDownloadDir string

// replayDownloadCmd downloads replays by their IDs.
var replayDownloadCmd = &cobra.Command{
	Use:         "download <replay-id>...",
	Annotations: map[string]string{needsBarDataDir: "true"}, //nolint:goconst
	Short:       "Download replays by their IDs",
	Long: `Downloads replays from the replay storage behind https://bar-rts.com into --dir, skipping those it holds
already. A replay ID is the game ID in hex, as in https://www.bar-rts.com/replays/<replay-id>.

Uses the same directory and index.json as 'replay download-top'.`,
	Args: cobra.MinimumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		downloader, err := newReplayDownloader(barlive.NewClient(barlive.WithOnRequest(printRequest)), false)
		if err != nil {
			return err
		}
		for _, id := range args {
			downloader.download(cmd.Context(), replayToDownload{ID: id})
		}
		return downloader.finish(nil)
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

// replayDownloader downloads replays into the download directory one after the other, as a download command is asked
// for them, prints a line for each it does not hold yet, and counts them for the summary. The directory's
// barlive.ReplayIndex tells which replays it holds.
type replayDownloader struct {
	client *barlive.Client
	index  *barlive.ReplayIndex

	// dir is the download directory, see downloadDir.
	dir string

	// dryRun makes it only list the missing replays.
	dryRun bool

	// seen holds the IDs of the replays it was asked for, so a replay asked for twice (a game of two of the best
	// players) counts once.
	seen map[string]bool

	// present counts the replays the directory held already, missing those it did not, failed those of them that
	// failed to download.
	present int
	missing int
	failed  int
}

// newReplayDownloader returns a replayDownloader for the directory --dir selects. With dryRun, it only lists the
// missing replays.
func newReplayDownloader(client *barlive.Client, dryRun bool) (*replayDownloader, error) {
	dir := downloadDir()
	index, err := barlive.OpenReplayIndex(dir)
	if err != nil {
		return nil, err
	}
	return &replayDownloader{
		client: client,
		index:  index,
		dir:    dir,
		dryRun: dryRun,
		seen:   make(map[string]bool),
	}, nil
}

// download downloads replay unless it was asked for before or the directory holds it, and prints a line for it if
// the directory did not hold it. With dryRun, it only prints the line.
func (d *replayDownloader) download(ctx context.Context, replay replayToDownload) {
	if d.seen[replay.ID] {
		return
	}
	d.seen[replay.ID] = true

	if d.index.Has(replay.ID) {
		d.present++
		return
	}
	d.missing++

	if d.dryRun {
		printReplayLine("missing", replay.ID, replay.Description)
		return
	}

	result, err := d.index.Fetch(ctx, d.client, replay.ID)
	switch {
	case errors.Is(err, fs.ErrExist):
		d.failed++
		printReplayLine("failed", replay.ID, "another game's replay has the same file name: "+err.Error())
	case err != nil:
		d.failed++
		printReplayLine("failed", replay.ID, err.Error())
	case result.Downloaded:
		printReplayLine("downloaded", result.FileName, replay.Description)
	default:
		// The file was there, but not in the index.
		printReplayLine("indexed", result.FileName, replay.Description)
	}
}

// finish saves the index unless dryRun and prints the summary, also if err, the error which stopped the command
// before it asked for every replay, is not nil. It returns err, joined with an error if the index cannot be saved or
// any replay failed.
func (d *replayDownloader) finish(err error) error {
	if !d.dryRun {
		err = errors.Join(err, d.index.Save())
	}

	fmt.Println()
	printField("dir", d.dir)
	printField("replays", len(d.seen))
	printField("present", d.present)
	if d.dryRun {
		printField("missing", d.missing)
		return err
	}
	printField("fetched", d.missing-d.failed)
	printField("failed", d.failed)
	if d.failed != 0 {
		err = errors.Join(err, fmt.Errorf("%d of %d replays failed", d.failed, d.missing))
	}
	return err
}

// printReplayLine prints the outcome for one replay.
func printReplayLine(status string, name string, description string) {
	fmt.Println(strings.TrimRight(fmt.Sprintf("%-10s  %s  %s", status, name, description), " "))
}
