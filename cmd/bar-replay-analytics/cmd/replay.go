package cmd

import (
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
