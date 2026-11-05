package cmd

import (
	"github.com/spf13/cobra"

	"github.com/backbone81/bar-replay-analytics-go/internal/playback"
)

// installWidgetCmd installs the analytics widget into the BAR data directory.
var installWidgetCmd = &cobra.Command{
	Use:   "install-widget",
	Short: "Install the analytics widget into the BAR data directory",
	Long: `Installs the analytics widget (dbg_replay_analytics.lua), which logs a replay's playback, with a README and its
license into LuaUI/Widgets/ of the BAR data directory. The widget is built into this program.

If the widget is installed already, its folder is used, otherwise LuaUI/Widgets/bar-replay-analytics/ is created.
Existing files are never overwritten: only missing files are created, and existing ones are reported as an error.
To update the widget, delete its files first.

Playback does not need this: it uses the widget built into this program, which replaces an installed one. The widget
is active only in a headless playback and does nothing in your own games.`,
	Args: cobra.NoArgs,
	RunE: func(_ *cobra.Command, _ []string) error {
		_, created, err := playback.InstallAnalyticsWidget(barDataDir)
		printCreated(created)
		return err
	},
}

func init() {
	rootCmd.AddCommand(installWidgetCmd)
}
