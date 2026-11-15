package cmd

import (
	"github.com/spf13/cobra"

	"github.com/backbone81/bar-replay-analytics/internal/coaching"
	"github.com/backbone81/bar-replay-analytics/internal/utils"
)

// initCmd writes the workspace of an AI agent that coaches a player into the current folder.
var initCmd = &cobra.Command{
	Use:   "init",
	Short: "Set up the current folder as the workspace of an AI coaching agent",
	Long: `Writes the workspace of an AI coaching agent into the current folder: AGENTS.md (the agent's instructions),
the templates player.md and training-log.md, and work/ for intermediate output. Start the agent in this folder
afterwards.

Never overwrites a file: stops with an error at the first one that exists.`,
	Args: cobra.NoArgs,
	RunE: func(_ *cobra.Command, _ []string) error {
		return utils.WriteTemplate(log, coaching.Template, ".")
	},
}

func init() {
	rootCmd.AddCommand(initCmd)
}
