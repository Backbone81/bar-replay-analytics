package cmd

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/backbone81/bar-replay-analytics-go/internal/demo"
	"github.com/backbone81/bar-replay-analytics-go/internal/tdf"
)

// replayScriptCmd prints the setup script of a replay.
var replayScriptCmd = &cobra.Command{
	Use:         "script <replay>",
	Annotations: map[string]string{needsBarDataDir: "true"}, //nolint:goconst
	Short:       "Print the start script of a replay as JSON",
	Long: `Prints the start script (script.txt) of a replay as JSON: map, game version, players, teams, allyteams, AIs
and game options. Names are lowercased, values are strings.

<replay> is a path to a .sdfz file, or the name of a file in the demos/ folder of the BAR data directory.`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		filePath, err := replayPath(args[0])
		if err != nil {
			return err
		}

		replay, err := demo.ReadFile(filePath, demo.ReadUntilEnd)
		if err != nil {
			return err
		}

		setupScript, err := tdf.Parse(replay.SetupScript)
		if err != nil {
			return fmt.Errorf("setup script: %w", err)
		}

		return writeJSON(setupScript)
	},
}

func init() {
	replayCmd.AddCommand(replayScriptCmd)
}
