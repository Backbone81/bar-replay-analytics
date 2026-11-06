package cmd

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/spf13/cobra"

	"github.com/backbone81/bar-replay-analytics-go/internal/logger"
	"github.com/backbone81/bar-replay-analytics-go/internal/utils"
)

// needsBarDataDir is the annotation of the commands that use barDataDir. The root command's PersistentPreRunE finds the
// BAR data dir only for them.
const needsBarDataDir = "needsBarDataDir"

var (
	// barDataDir is the BAR data dir: --bar-data-dir, or the one utils.BarDataDir finds if that is empty.
	barDataDir string

	log = logger.New(os.Stderr)
)

// rootCmd represents the base command when called without any subcommands.
var rootCmd = &cobra.Command{
	Use:   "bar-replay-analytics-go",
	Short: "Analyze replays of Beyond All Reason",
	Long: `Analyzes replays of Beyond All Reason for coaching: reads them, plays them back headless and reports on the
game. Also shows the leaderboards of the BAR lobby server and downloads replays from https://bar-rts.com.`,
	SilenceUsage: true,
	PersistentPreRunE: func(cmd *cobra.Command, _ []string) error {
		if barDataDir != "" || cmd.Annotations[needsBarDataDir] == "" {
			return nil
		}

		dataDir, err := utils.BarDataDir()
		if err != nil {
			return err
		}

		barDataDir = dataDir
		return nil
	},
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
		"The BAR data directory which contains the folders demos/, engine/ and maps/. Empty autodetects the "+
			"location.",
	)
}

// printRequest prints a line for a request to requestURL to stderr, aligned like the lines of printReplayLine. The
// commands pass it to barlive.WithOnRequest, so a run shows progress while its requests wait for the rate limit, and
// stdout holds only the results.
func printRequest(requestURL string) {
	_, _ = fmt.Fprintf(os.Stderr, "%-10s  %s\n", "GET", requestURL)
}
