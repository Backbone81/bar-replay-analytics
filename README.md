# BAR Replay Analytics Go

This repository provides a tool for analyzing replays of [Beyond All Reason](https://www.beyondallreason.info/).

## Usage

All commands take `--bar-data-dir` (default `data`), the BAR data directory with `demos/`.

```sh
# Read a replay: everything the tool reads (JSON), or only its setup script
bar-replay-analytics-go replay inspect <replay>
bar-replay-analytics-go replay script <replay>

# The report of a replay (JSON): metadata and game options, teams, every player (name, faction, income bonus, AI)
# and its economy, over the game and in intervals of 1 minute for the first 10 minutes and of 5 minutes after that
bar-replay-analytics-go replay report <replay> \
    [--opening-length 10m] [--opening-interval-length 1m] [--interval-length 5m]

# Play replays back in a headless engine at maximum speed, several at a time, with the analytics widget of
# bar-replay-analytics logging each game to events/<replay name>_events.json in the widget's folder
bar-replay-analytics-go replay play <replay>... \
    [--parallel N] [--engine-threads 2] [--progress-interval 15s] [--unended]

# Leaderboards of the BAR lobby server: seasons and game types, best players
bar-replay-analytics-go leaderboard seasons
bar-replay-analytics-go leaderboard show [--season N] [--game-type duel] [--top 20]

# Download the games of the 10 best duel players of the last 7 days into data/demos/leaderboard/
bar-replay-analytics-go replay download-top [--game-type duel] [--season N] [--top 10] [--days 7] [--dry-run]

# Download replays by ID (as in https://www.bar-rts.com/replays/<replay-id>)
bar-replay-analytics-go replay download <replay-id>...
```

`replay play` needs the analytics widget (`dbg_replay_analytics.lua`) installed in `LuaUI/Widgets/` of the BAR data
directory, by cloning or linking [bar-replay-analytics](https://github.com/backbone81/bar-replay-analytics) there. It
downloads a missing engine (needs 7-Zip: `7z`, `7zz` or `7za` on `PATH`), game and map into the data directory, as the
BAR lobby does. Each playback runs in a temporary write dir of its own with its own UDP port, so parallel playbacks do
not get in each other's way and the data directory's `infolog.txt`, settings and `LuaUI/Config/` stay as they are.
Every `--progress-interval`, stderr shows how far each playback has come in percent of the game. An engine needs about
5 GB of memory and, with `--engine-threads 2` (the engine's `WorkerThreadCount`), about 1.5 CPU cores, so `--parallel`
defaults to one playback per 2 CPUs, but at most one per 6 GB of available memory. The engine's own default of one
thread per CPU is slower and burns about 5 cores per playback on idle threads. Replays without a game length in their
header (the engine crashed or the game was not ended) are skipped unless `--unended` is given, as their stream can go
on for hours.

`--game-type` is `duel`, `small-team`, `large-team` or `ffa`. `--season` 0 or omitted selects the active season.

`replay download-top` keeps only the games of the leaderboard's game type, by the size of their allyteams as the lobby
server decides it, without AIs and aborted games. Unrated games count too: the replay database does not tell them
apart. Running it again downloads only what is missing: `index.json` in the directory records which file holds which
replay. Files added or removed by hand are noticed on the next run. Replays are stored under the replay storage's file names in `demos/leaderboard/` (`--dir`), which the
BAR lobby's replay list does not show. It goes through the players best first and downloads each player's missing
replays one after the other, as the replay database lists them; requests to the lobby server and the replay database
are limited to five per second.
