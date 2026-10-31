# BAR Replay Analytics Go

This repository provides a tool for analyzing replays of [Beyond All Reason](https://www.beyondallreason.info/).

## Usage

All commands take `--bar-data-dir` (default `data`), the BAR data directory with `demos/`.

```sh
# Read a replay: everything the tool reads (JSON), or only its setup script
bar-replay-analytics-go replay inspect <replay>
bar-replay-analytics-go replay script <replay>

# The report of a replay (JSON): metadata and every player's economy, over the game and in intervals of 5 minutes
bar-replay-analytics-go replay report <replay> [--interval-length 5m]

# Leaderboards of the BAR lobby server: seasons and game types, best players
bar-replay-analytics-go leaderboard seasons
bar-replay-analytics-go leaderboard show [--season N] [--game-type duel] [--top 20]

# Download the games of the 10 best duel players of the last 7 days into data/demos/leaderboard/
bar-replay-analytics-go replay download-top --game-type duel [--season N] [--top 10] [--days 7] [--dry-run]

# Download replays by ID (as in https://www.bar-rts.com/replays/<replay-id>)
bar-replay-analytics-go replay download <replay-id>...
```

`--game-type` is `duel`, `small-team`, `large-team` or `ffa`. `--season` 0 or omitted selects the active season.

`replay download-top` keeps only the games of the leaderboard's game type, by the size of their allyteams as the lobby
server decides it, without AIs and aborted games. Unrated games count too: the replay database does not tell them
apart. Running it again downloads only what is missing: `index.json` in the directory records which file holds which
replay. Files added or removed by hand are noticed on the next run. Replays are stored under the replay storage's file names in `demos/leaderboard/` (`--dir`), which the
BAR lobby's replay list does not show. `--parallel` sets the number of downloads at a time (default 4); requests to
the lobby server and the replay database are limited to five per second.
