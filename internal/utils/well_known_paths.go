package utils

import (
	"path/filepath"
	"strings"
)

// engineVersionDirPrefix is the prefix of the folder of an engine in engine/, followed by its version.
const engineVersionDirPrefix = "recoil_"

// DemosDir returns the folder in the data dir dataDir where BAR saves the replays.
func DemosDir(dataDir string) string {
	return filepath.Join(dataDir, "demos")
}

// EngineDir returns the folder in the data dir dataDir that holds the engines.
func EngineDir(dataDir string) string {
	return filepath.Join(dataDir, "engine")
}

// EngineVersionDir returns the folder of the engine engineVersion in the data dir dataDir, named like the BAR launcher
// names it.
func EngineVersionDir(dataDir string, engineVersion string) string {
	return filepath.Join(EngineDir(dataDir), engineVersionDirPrefix+engineVersion)
}

// PrDownloaderFilePath returns the pr-downloader of the engine engineVersion in the data dir dataDir.
func PrDownloaderFilePath(dataDir string, engineVersion string) string {
	return filepath.Join(EngineVersionDir(dataDir, engineVersion), "pr-downloader")
}

// SpringHeadlessFilePath returns the spring-headless of the engine engineVersion in the data dir dataDir.
func SpringHeadlessFilePath(dataDir string, engineVersion string) string {
	return filepath.Join(EngineVersionDir(dataDir, engineVersion), "spring-headless")
}

// MapsDir returns the folder of the map archives in the data dir dataDir.
func MapsDir(dataDir string) string {
	return filepath.Join(dataDir, "maps")
}

// RapidDir returns the folder of the rapid repos in the data dir dataDir.
func RapidDir(dataDir string) string {
	return filepath.Join(dataDir, "rapid")
}

// PackagesDir returns the folder of the rapid packages in the data dir dataDir.
func PackagesDir(dataDir string) string {
	return filepath.Join(dataDir, "packages")
}

// WidgetsDir returns the user widgets' folder in the data dir dataDir (WIDGET_DIRNAME in BAR's luaui/barwidgets.lua).
func WidgetsDir(dataDir string) string {
	return filepath.Join(dataDir, "LuaUI", "Widgets")
}

// AnalyticsWidgetDir returns the analytics widget's folder in the data dir dataDir.
func AnalyticsWidgetDir(dataDir string) string {
	return filepath.Join(WidgetsDir(dataDir), "bar-replay-analytics")
}

// WidgetHandlerConfigFilePath returns BAR's widget handler config in the data dir dataDir:
// LuaUI/Config/<Game.gameShortName>.lua (CONFIG_FILENAME in BAR's luaui/barwidgets.lua), where BAR's gameShortName is
// BYAR. Its order list says which widgets are enabled.
func WidgetHandlerConfigFilePath(dataDir string) string {
	return filepath.Join(dataDir, "LuaUI", "Config", "BYAR.lua")
}

// AnalyticsWidgetEventsDir returns the folder in the data dir dataDir where the analytics widget writes its events
// files.
func AnalyticsWidgetEventsDir(dataDir string) string {
	return filepath.Join(AnalyticsWidgetDir(dataDir), "events")
}

// LauncherLogsDir returns the folder in the data dir dataDir where the BAR launcher writes its logs. The launcher
// creates it on every start (spring-launcher src/spring_log.js).
func LauncherLogsDir(dataDir string) string {
	return filepath.Join(dataDir, "launcher-logs")
}

// CacheDir returns the engine's cache folder in the data dir dataDir.
func CacheDir(dataDir string) string {
	return filepath.Join(dataDir, "cache")
}

// SpringsettingsCfgFilePath returns the engine's config file in the data dir dataDir.
func SpringsettingsCfgFilePath(dataDir string) string {
	return filepath.Join(dataDir, "springsettings.cfg")
}

// DemoEventsFilePath returns the events file of the demo demoFilePath: <demo name>_events.json next to it, where the
// player copies the file the analytics widget wrote.
func DemoEventsFilePath(demoFilePath string) string {
	return siblingFilePath(demoFilePath, "_events.json")
}

// DemoReportFilePath returns the report of the demo demoFilePath: <demo name>_report.json next to it.
func DemoReportFilePath(demoFilePath string) string {
	return siblingFilePath(demoFilePath, "_report.json")
}

// DemoLogFilePath returns the engine's output of the playback of the demo demoFilePath: <demo name>.log next to it.
func DemoLogFilePath(demoFilePath string) string {
	return siblingFilePath(demoFilePath, ".log")
}

// siblingFilePath returns the file next to the demoFilePath that has suffix in place of the demo's extension.
func siblingFilePath(filePath string, suffix string) string {
	return strings.TrimSuffix(filePath, filepath.Ext(filePath)) + suffix
}
