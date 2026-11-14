package cmd

import (
	"github.com/spf13/cobra"

	"github.com/backbone81/bar-replay-analytics-go/internal/replay/player/widget"
	"github.com/backbone81/bar-replay-analytics-go/internal/utils"
)

// installWidgetCmd installs the analytics widget into the BAR data directory.
var installWidgetCmd = &cobra.Command{
	Use:         "install-widget",
	Annotations: map[string]string{needsBarDataDir: "true"}, //nolint:goconst
	Short:       "Install the analytics widget into the BAR data directory",
	Long: `Installs the analytics widget (dbg_replay_analytics.lua) with its README and license into
LuaUI/Widgets/bar-replay-analytics/ of the BAR data directory. Never overwrites a file: stops with an error at the
first one that exists, so delete the installed files first to update.

'replay play' does not need this: it uses the widget built into this program. The widget is active only in a headless
playback.`,
	Args: cobra.NoArgs,
	RunE: func(_ *cobra.Command, _ []string) error {
		return utils.WriteTemplate(
			log,
			widget.Template,
			utils.AnalyticsWidgetDir(barDataDir),
		)
	},
}

func init() {
	rootCmd.AddCommand(installWidgetCmd)
}
