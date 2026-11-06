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
	Long: `Writes the workspace of an AI coaching agent into the current folder: AGENTS.md (the agent's instructions),
the templates player.md and training-log.md, and work/ for intermediate output. Start the agent in this folder
afterwards.

Only missing files are created; existing ones are left unchanged and reported as errors.`,
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
