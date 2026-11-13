package cmd

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"

	"github.com/spf13/cobra"

	"github.com/backbone81/bar-replay-analytics-go/internal/demo"
	"github.com/backbone81/bar-replay-analytics-go/internal/events"
	"github.com/backbone81/bar-replay-analytics-go/internal/logger"
	replayplayer "github.com/backbone81/bar-replay-analytics-go/internal/replay/player"
	"github.com/backbone81/bar-replay-analytics-go/internal/replay/player/widget"
	"github.com/backbone81/bar-replay-analytics-go/internal/report"
	"github.com/backbone81/bar-replay-analytics-go/internal/tdf"
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
	return filepath.Join(barDataDir, "demos", arg), nil
}

// replayFileExtension is the extension of BAR's replay files.
const replayFileExtension = ".sdfz"

// replayPaths returns the paths of the replays args name: each arg is resolved by replayPath; a folder stands for the
// .sdfz files directly in it, sorted by name. A replay named twice is returned once. Returns an error if a folder
// holds no replay.
func replayPaths(args []string) ([]string, error) {
	var filePaths []string
	seen := make(map[string]bool)
	add := func(filePath string) error {
		absoluteFilePath, err := filepath.Abs(filePath)
		if err != nil {
			return err
		}
		if !seen[absoluteFilePath] {
			seen[absoluteFilePath] = true
			filePaths = append(filePaths, filePath)
		}
		return nil
	}
	for _, arg := range args {
		filePath, err := replayPath(arg)
		if err != nil {
			return nil, err
		}
		info, err := os.Stat(filePath)
		if err != nil || !info.IsDir() {
			// A missing file fails when its command reads it, with the replay's name in the error.
			if err := add(filePath); err != nil {
				return nil, err
			}
			continue
		}
		entries, err := os.ReadDir(filePath)
		if err != nil {
			return nil, err
		}
		found := false
		for _, entry := range entries { // sorted by name
			if !entry.IsDir() && filepath.Ext(entry.Name()) == replayFileExtension {
				found = true
				if err := add(filepath.Join(filePath, entry.Name())); err != nil {
					return nil, err
				}
			}
		}
		if !found {
			return nil, fmt.Errorf("%s: no %s files in the folder", arg, replayFileExtension)
		}
	}
	return filePaths, nil
}

// The statuses a command that handles several replays prints for each of them, and counts.
const (
	statusReported = "reported"
	statusSkipped  = "skipped"
	statusFailed   = "failed"
)

// statusCounts provides how many replays a command ended with each status, such as "played" or "failed", by status.
type statusCounts map[string]int

// print prints, after an empty line, the number of replays and the count of each of statuses, in that order.
func (c statusCounts) print(statuses ...string) {
	total := 0
	for _, count := range c {
		total += count
	}
	fmt.Println()
	printField("replays", total)
	for _, status := range statuses {
		printField(status, c[status])
	}
}

// printReplayLine prints the outcome for one replay to stdout.
func printReplayLine(status string, name string, description string) {
	fmt.Println(strings.TrimRight(fmt.Sprintf("%-10s  %s  %s", status, name, description), " "))
}

// printField prints one "label: value" line to stdout, with the values of all such lines aligned.
func printField(label string, value any) {
	fmt.Printf("%-12s %v\n", label+":", value)
}

// writeJSON writes value to stdout as indented JSON.
func writeJSON(value any) error {
	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")
	return encoder.Encode(value)
}

// writeJSONFile writes value as indented JSON to filePath, replacing it. It writes a temporary file next to it and
// renames that, so filePath is never incomplete.
func writeJSONFile(filePath string, value any) error {
	data, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return err
	}
	tempFile, err := os.CreateTemp(filepath.Dir(filePath), filepath.Base(filePath)+".tmp-*")
	if err != nil {
		return err
	}
	_, err = tempFile.Write(append(data, '\n'))
	err = errors.Join(err, tempFile.Close())
	if err == nil {
		err = os.Chmod(tempFile.Name(), 0o644) //nolint:gosec // This file does not contain sensitive data.
	}
	if err == nil {
		err = os.Rename(tempFile.Name(), filePath)
	}
	if err != nil {
		return errors.Join(err, os.Remove(tempFile.Name()))
	}
	return nil
}

// reportReplay writes the report of the replay filePath, and returns its status, "reported", "skipped" (no events
// file) or "failed", and a description of the outcome.
func reportReplay(filePath string) (string, string) {
	eventsFilePath := replayplayer.EventsFilePath(filePath)
	if _, err := os.Stat(eventsFilePath); errors.Is(err, fs.ErrNotExist) {
		return statusSkipped, "no events file: play the replay back first (replay play)"
	}
	replay, err := demo.ReadFile(filePath, demo.ReadUntilEnd)
	if err != nil {
		return statusFailed, err.Error()
	}
	setupScript, err := tdf.Parse(replay.SetupScript)
	if err != nil {
		return statusFailed, fmt.Sprintf("setup script: %v", err)
	}
	eventsFile, err := events.ReadFile(eventsFilePath)
	if err != nil {
		return statusFailed, fmt.Sprintf("events file of the replay's playback: %v", err)
	}
	result, err := report.NewReport(
		replay,
		setupScript,
		eventsFile,
		report.WithOpeningLength(replayReportOpeningLength),
		report.WithOpeningIntervalLength(replayReportOpeningIntervalLength),
		report.WithIntervalLength(replayReportIntervalLength),
	)
	if err != nil {
		return statusFailed, err.Error()
	}
	reportFilePath := reportPath(filePath)
	if err := writeJSONFile(reportFilePath, result); err != nil {
		return statusFailed, err.Error()
	}
	description := reportFilePath
	if eventsFile.FormatVersion < widget.FormatVersion {
		description += fmt.Sprintf(" (events file outdated: format %d, the widget writes %d; play the replay "+
			"again with replay play)", eventsFile.FormatVersion, widget.FormatVersion)
	}
	return statusReported, description
}

// reportPath returns the path of the report of the replay file filePath: <replay name>_report.json next to it, like
// the events file of its playback (see player.EventsFilePath).
func reportPath(filePath string) string {
	return strings.TrimSuffix(filePath, filepath.Ext(filePath)) + "_report.json"
}
