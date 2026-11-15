package cmd

import (
	"context"
	"encoding/json"
	"errors"
	"io/fs"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"

	"github.com/spf13/cobra"

	"github.com/backbone81/bar-replay-analytics/internal/logger"
	"github.com/backbone81/bar-replay-analytics/internal/utils"
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
	Use:   "bar-replay-analytics",
	Short: "Analyze replays of Beyond All Reason",
	Long: `Analyzes replays of Beyond All Reason for coaching: reads them, plays them back headless and reports on the
game. Also downloads replays from https://bar-rts.com, such as the recent games of the best players on the
leaderboards of the BAR lobby server.`,
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

// replayPath returns the path of the replay named by arg: arg itself if that file exists, otherwise arg in the demos/
// folder of the BAR data directory.
func replayPath(arg string) (string, error) {
	_, err := os.Stat(arg)
	if err == nil {
		return arg, nil
	}
	if !errors.Is(err, fs.ErrNotExist) {
		return "", err
	}
	return filepath.Join(utils.DemosDir(barDataDir), arg), nil
}

// writeJSON writes value to stdout as indented JSON.
func writeJSON(value any) error {
	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")
	return encoder.Encode(value)
}
