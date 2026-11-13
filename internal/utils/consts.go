package utils

import "path/filepath"

// WidgetsDir is the user widgets' folder in a data dir or write dir (WIDGET_DIRNAME in BAR's luaui/barwidgets.lua).
var WidgetsDir = filepath.Join("LuaUI", "Widgets")

var CacheDir = "cache"

// AnalyticsWidgetFolder is the analytics widget's folder in LuaUI/Widgets/.
const AnalyticsWidgetFolder = "bar-replay-analytics"

// WidgetConfigFile is BAR's widget config file in a data dir or write dir: LuaUI/Config/<Game.gameShortName>.lua
// (CONFIG_FILENAME in BAR's luaui/barwidgets.lua), where BAR's gameShortName is BYAR. Its order list says which widgets
// are enabled.
var WidgetConfigFile = filepath.Join("LuaUI", "Config", "BYAR.lua")
