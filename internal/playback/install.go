package playback

import (
	"path/filepath"

	"github.com/backbone81/bar-replay-analytics-go/internal/newfiles"
	"github.com/backbone81/bar-replay-analytics-go/internal/playback/widget"
)

// InstallAnalyticsWidget writes the embedded template of the analytics widget's folder (widget.Template: the widget, a
// README and its license) into LuaUI/Widgets/ of the data dir dataDir. It returns the folder it wrote into and the
// files it created.
//
// If the widget is installed already, InstallAnalyticsWidget uses that folder, so BAR does not load two copies.
// Otherwise it creates LuaUI/Widgets/bar-replay-analytics/. It never overwrites a file: it creates the missing ones and
// returns a *newfiles.ExistError listing those that exist already.
func InstallAnalyticsWidget(dataDir string) (string, []string, error) {
	widgets := filepath.Join(dataDir, widgetsDir)
	widgetDir, found, err := FindWidgetDir(widgets, AnalyticsWidgetFile)
	if err != nil {
		return "", nil, err
	}
	if !found {
		widgetDir = filepath.Join(widgets, analyticsWidgetFolder)
	}
	created, err := newfiles.Write(widget.Template, widgetDir)
	return widgetDir, created, err
}
