package cmd

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
)

// replayCmd groups the commands which read or download replays.
var replayCmd = &cobra.Command{
	Use:   "replay",
	Short: "Read, play back, report and download replays",
}

func init() {
	rootCmd.AddCommand(replayCmd)
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

// replayArgsHelp describes the <replay> arguments of the commands that take several replays (see replayPaths).
const replayArgsHelp = `<replay> is a path to a .sdfz file or to a folder, or the name of a file or folder in the
demos/ folder of the BAR data directory. A folder stands for the .sdfz files directly in it (not in its subfolders),
such as "leaderboard" for demos/leaderboard/.`

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

// outputFilePermissions are the permissions of the files the commands write next to a replay; none is confidential.
const outputFilePermissions = 0o644

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
		err = os.Chmod(tempFile.Name(), outputFilePermissions)
	}
	if err == nil {
		err = os.Rename(tempFile.Name(), filePath)
	}
	if err != nil {
		return errors.Join(err, os.Remove(tempFile.Name()))
	}
	return nil
}
