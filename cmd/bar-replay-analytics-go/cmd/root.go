package cmd

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/spf13/cobra"
)

var barDataDir string

// rootCmd represents the base command when called without any subcommands.
var rootCmd = &cobra.Command{
	Use:   "bar-replay-analytics-go",
	Short: "Analyze replays of Beyond All Reason",
	Long: `Reads replays (demo files) of Beyond All Reason, shows the leaderboards of the BAR lobby server and downloads
the replays of top players from the replay database behind https://bar-rts.com.`,
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
		"data",
		"The BAR data directory which contains the folders demos/, engine/ and maps/.",
	)
}

// printRequest prints a line for a request to requestURL to stderr, aligned like the lines of printReplayLine. The
// commands pass it to barlive.WithOnRequest, so a run shows progress while its requests wait for the rate limit, and
// stdout holds only the results.
func printRequest(requestURL string) {
	_, _ = fmt.Fprintf(os.Stderr, "%-10s  %s\n", "GET", requestURL)
}
