package cmd

import (
	"github.com/spf13/cobra"

	"github.com/backbone81/bar-replay-analytics-go/internal/demo"
)

// replayInspectCmd prints everything the tool reads from a replay.
var replayInspectCmd = &cobra.Command{
	Use:         "inspect <replay>",
	Annotations: map[string]string{needsBarDataDir: "true"}, //nolint:goconst
	Short:       "Print the content of a replay as JSON",
	Long: `Prints the file header, start script, winning allyteams, player statistics and team statistics of a replay as
JSON. The recorded game itself is not included.

<replay> is a path to a .sdfz file, or the name of a file in the demos/ folder of the BAR data directory.`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		filePath, err := replayPath(args[0])
		if err != nil {
			return err
		}
		replay, err := demo.ReadFile(filePath)
		if err != nil {
			return err
		}
		return writeJSON(replay)
	},
}

func init() {
	replayCmd.AddCommand(replayInspectCmd)
}
