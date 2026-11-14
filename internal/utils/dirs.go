package utils

import "path/filepath"

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
	return filepath.Join(EngineDir(dataDir), "recoil_"+engineVersion)
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

// CacheDir returns the engine's cache folder in the data dir dataDir.
func CacheDir(dataDir string) string {
	return filepath.Join(dataDir, "cache")
}

// SpringsettingsCfgFilePath returns the engine's config file in the data dir dataDir.
func SpringsettingsCfgFilePath(dataDir string) string {
	return filepath.Join(dataDir, "springsettings.cfg")
}
