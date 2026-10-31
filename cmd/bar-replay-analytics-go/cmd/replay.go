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
	return filepath.Join(barDataDir, "demos", arg), nil
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
