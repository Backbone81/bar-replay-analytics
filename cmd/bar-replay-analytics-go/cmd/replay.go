package cmd

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
)

// replayCmd groups the commands which read or download replays.
var replayCmd = &cobra.Command{
	Use:   "replay",
	Short: "Work with replays",
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
	dir, err := dataDir()
	if err != nil {
		return "", fmt.Errorf("%s is no file, and the demos/ folder to look for it in is unknown: %w", arg, err)
	}
	return filepath.Join(dir, "demos", arg), nil
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
