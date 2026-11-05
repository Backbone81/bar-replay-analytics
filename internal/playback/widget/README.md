# Widget

This folder contains the template of the analytics widget's folder: `template/` holds a Beyond All Reason user widget
for detailed log output during replays, and its license. The template is embedded into the binary, and
`bar-replay-analytics-go install-widget` writes all of it into `LuaUI/Widgets/` of the BAR data directory.

The files in `template/` are licensed under GPL-2.0-or-later (see `template/LICENSE`), like BAR's own widgets, as the
widget runs inside the game.
