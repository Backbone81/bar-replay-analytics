package cmd

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/backbone81/bar-replay-analytics-go/internal/coaching"
)

// agentsMDCmd prints the instructions for an AI agent that coaches a player.
var agentsMDCmd = &cobra.Command{
	Use:   "agents-md",
	Short: "Print the AGENTS.md for an AI agent that coaches a player",
	Long: `Prints the AGENTS.md for an AI agent that coaches a player from the analytics of their replays. Write it into
the folder the agent runs in:

  bar-replay-analytics-go agents-md > AGENTS.md`,
	Args: cobra.NoArgs,
	RunE: func(cmd *cobra.Command, _ []string) error {
		_, err := fmt.Fprint(cmd.OutOrStdout(), coaching.AgentsMD)
		return err
	},
}

func init() {
	rootCmd.AddCommand(agentsMDCmd)
}
