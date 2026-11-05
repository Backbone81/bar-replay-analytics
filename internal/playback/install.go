package playback

import (
	"io/fs"
	"os"
	"path/filepath"

	"github.com/backbone81/bar-replay-analytics-go/internal/playback/widget"
)

// analyticsWidgetFolder is the folder in LuaUI/Widgets/ InstallAnalyticsWidget installs the analytics widget into, if
// it is not installed yet: the name of the repository it comes from.
const analyticsWidgetFolder = "bar-replay-analytics"

// InstallAnalyticsWidget writes the embedded analytics widget and its license (widget.Files) into LuaUI/Widgets/ of the
// data dir dataDir, overwriting the files of an older version. It returns the folder it wrote them into.
//
// If the widget is installed already, InstallAnalyticsWidget updates that folder, so BAR does not load two copies.
// Otherwise it creates LuaUI/Widgets/bar-replay-analytics/.
func InstallAnalyticsWidget(dataDir string) (string, error) {
	widgets := filepath.Join(dataDir, widgetsDir)
	widgetDir, found, err := FindWidgetDir(widgets, AnalyticsWidgetFile)
	if err != nil {
		return "", err
	}
	if !found {
		widgetDir = filepath.Join(widgets, analyticsWidgetFolder)
	}

	if err := os.MkdirAll(widgetDir, dirPermissions); err != nil {
		return "", err
	}
	entries, err := fs.ReadDir(widget.Files, ".")
	if err != nil {
		return "", err
	}
	for _, entry := range entries {
		content, err := widget.Files.ReadFile(entry.Name())
		if err != nil {
			return "", err
		}
		if err := os.WriteFile(filepath.Join(widgetDir, entry.Name()), content, filePermissions); err != nil {
			return "", err
		}
	}
	return widgetDir, nil
}
