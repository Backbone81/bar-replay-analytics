# Replay Analytics widget

`dbg_replay_analytics.lua` is a Beyond All Reason user widget from
[bar-replay-analytics-go](https://github.com/backbone81/bar-replay-analytics-go), a tool that analyzes replays.

The widget is active only when a replay is played back in a headless engine (`spring-headless`). It then plays the
replay at maximum speed, logs unit, economy and builder events to `events/<replay name>_events.json` in this folder,
and quits the engine at the end. In a normal game and when watching a replay, it removes itself and does nothing.

The tool writes the widget out for each playback by itself. This folder is only needed to inspect the widget; you can
delete it.

The files in this folder are licensed under GPL-2.0-or-later (see `LICENSE`).
