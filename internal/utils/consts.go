package utils

import "path/filepath"

// WidgetConfigFile is BAR's widget config file in a data dir or write dir: LuaUI/Config/<Game.gameShortName>.lua
// (CONFIG_FILENAME in BAR's luaui/barwidgets.lua), where BAR's gameShortName is BYAR. Its order list says which widgets
// are enabled.
var WidgetConfigFile = filepath.Join("LuaUI", "Config", "BYAR.lua")
