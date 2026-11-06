package cmd

import (
	"github.com/spf13/cobra"

	"github.com/backbone81/bar-replay-analytics-go/internal/playback"
)

// installWidgetCmd installs the analytics widget into the BAR data directory.
var installWidgetCmd = &cobra.Command{
	Use:   "install-widget",
	Short: "Install the analytics widget into the BAR data directory",
	Long: `Installs the analytics widget (dbg_replay_analytics.lua) with its README and license into the folder of an
installed copy, or else into LuaUI/Widgets/bar-replay-analytics/ of the BAR data directory. Only missing files are
created; existing ones are left unchanged and reported as errors, so delete them first to update.

'replay play' does not need this: it uses the widget built into this program. The widget is active only in a headless
playback.`,
	Args: cobra.NoArgs,
	RunE: func(_ *cobra.Command, _ []string) error {
		dir, err := dataDir()
		if err != nil {
			return err
		}
		_, created, err := playback.InstallAnalyticsWidget(dir)
		printCreated(created)
		return err
	},
}

func init() {
	rootCmd.AddCommand(installWidgetCmd)
}
