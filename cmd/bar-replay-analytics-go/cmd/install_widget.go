package cmd

import (
	"github.com/spf13/cobra"

	"github.com/backbone81/bar-replay-analytics-go/internal/playback"
)

// installWidgetCmd installs the analytics widget into the BAR data directory.
var installWidgetCmd = &cobra.Command{
	Use:   "install-widget",
	Short: "Install the analytics widget into the BAR data directory",
	Long: `Installs the analytics widget (dbg_replay_analytics.lua), which logs a replay's playback, and its license into
LuaUI/Widgets/ of the BAR data directory. The widget is built into this program; installing it again updates it.

If the widget is installed already, its folder is updated, otherwise LuaUI/Widgets/bar-replay-analytics/ is created.

The widget is active only in a headless playback and does nothing in your own games.`,
	Args: cobra.NoArgs,
	RunE: func(_ *cobra.Command, _ []string) error {
		widgetDir, err := playback.InstallAnalyticsWidget(barDataDir)
		if err != nil {
			return err
		}
		printField("installed", widgetDir)
		return nil
	},
}

func init() {
	rootCmd.AddCommand(installWidgetCmd)
}
