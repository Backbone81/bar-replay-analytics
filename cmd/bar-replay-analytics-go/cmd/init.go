package cmd

import (
	"github.com/spf13/cobra"

	"github.com/backbone81/bar-replay-analytics-go/internal/coaching"
	"github.com/backbone81/bar-replay-analytics-go/internal/newfiles"
)

// initCmd writes the workspace of an AI agent that coaches a player into the current folder.
var initCmd = &cobra.Command{
	Use:   "init",
	Short: "Set up the current folder as the workspace of an AI coaching agent",
	Long: `Writes the workspace of an AI agent that coaches a player from the analytics of their replays into the current
folder: AGENTS.md with the agent's instructions, the templates player.md and training-log.md the agent fills in, and
the folder work/ for its intermediate output. Start your agent in this folder afterwards.

Existing files are never overwritten: only missing files are created, and existing ones are reported as an error.`,
	Args: cobra.NoArgs,
	RunE: func(_ *cobra.Command, _ []string) error {
		created, err := newfiles.Write(coaching.Template, ".")
		printCreated(created)
		return err
	},
}

func init() {
	rootCmd.AddCommand(initCmd)
}

// printCreated prints a "created" line for each of the files in filePaths.
func printCreated(filePaths []string) {
	for _, filePath := range filePaths {
		printField("created", filePath)
	}
}
