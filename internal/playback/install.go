package playback

import (
	"log/slog"
	"path/filepath"

	"github.com/backbone81/bar-replay-analytics-go/internal/playback/widget"
	"github.com/backbone81/bar-replay-analytics-go/internal/utils"
)

// InstallAnalyticsWidget writes the embedded template of the analytics widget's folder (widget.Template: the widget, a
// README and its license) into LuaUI/Widgets/ of the data dir dataDir. It returns the folder it wrote into.
//
// If the widget is installed already, InstallAnalyticsWidget uses that folder, so BAR does not load two copies.
// Otherwise it creates LuaUI/Widgets/bar-replay-analytics/. It never overwrites a file, see utils.WriteTemplate.
func InstallAnalyticsWidget(log *slog.Logger, dataDir string) (string, error) {
	widgets := filepath.Join(dataDir, widgetsDir)
	widgetDir, found, err := FindWidgetDir(widgets, AnalyticsWidgetFile)
	if err != nil {
		return "", err
	}
	if !found {
		widgetDir = filepath.Join(widgets, analyticsWidgetFolder)
	}
	err = utils.WriteTemplate(log, widget.Template, widgetDir)
	return widgetDir, err
}
