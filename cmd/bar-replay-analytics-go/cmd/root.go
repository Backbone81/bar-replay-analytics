package cmd

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/spf13/cobra"

	"github.com/backbone81/bar-replay-analytics-go/internal/datadir"
)

// barDataDir is the value of --bar-data-dir; use dataDir, which finds the BAR data dir if it is empty.
var barDataDir string

// rootCmd represents the base command when called without any subcommands.
var rootCmd = &cobra.Command{
	Use:   "bar-replay-analytics-go",
	Short: "Analyze replays of Beyond All Reason",
	Long: `Analyzes replays of Beyond All Reason for coaching: reads them, plays them back headless and reports on the
game. Also shows the leaderboards of the BAR lobby server and downloads replays from https://bar-rts.com.`,
	SilenceUsage: true,
}

// Execute adds all child commands to the root command and sets flags appropriately. Ctrl-C or SIGTERM cancel the
// context of the running command, so downloads stop and leave no partial files.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	err := rootCmd.ExecuteContext(ctx)
	stop()
	if err != nil {
		os.Exit(1)
	}
}

func init() {
	rootCmd.PersistentFlags().StringVar(
		&barDataDir,
		"bar-data-dir",
		"",
		"The BAR data directory which contains the folders demos/, engine/ and maps/. Empty looks for it where the "+
			"BAR installers put it (Windows: %LOCALAPPDATA%\\Programs\\Beyond-All-Reason\\data; Linux: "+
			"~/.var/app/info.beyondallreason.bar/data for the Flatpak, ~/.local/state/Beyond All Reason for the "+
			"AppImage).",
	)
}

// dataDir returns the BAR data directory: --bar-data-dir, or the one datadir.Find finds if that is empty. The
// commands call it only when they need the directory, so the others work without a BAR install.
func dataDir() (string, error) {
	if barDataDir != "" {
		return barDataDir, nil
	}
	found, err := datadir.Find()
	if err != nil {
		return "", err
	}
	barDataDir = found
	return found, nil
}

// printRequest prints a line for a request to requestURL to stderr, aligned like the lines of printReplayLine. The
// commands pass it to barlive.WithOnRequest, so a run shows progress while its requests wait for the rate limit, and
// stdout holds only the results.
func printRequest(requestURL string) {
	_, _ = fmt.Fprintf(os.Stderr, "%-10s  %s\n", "GET", requestURL)
}
