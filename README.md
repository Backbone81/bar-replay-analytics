# BAR Replay Analytics

Analyze your [Beyond All Reason](https://www.beyondallreason.info/) games from their replays, and let an AI agent
coach you.

`bar-replay-analytics-go` plays a replay back at maximum speed in the background and turns it into a report: your
economy minute by minute, metal extractors, idle constructors, what you killed and lost, your build order, side by
side with your opponent's. It also downloads the replays of the best players from the leaderboards, so you can compare
your games with theirs.

The report is written for an AI agent (Claude Code, Codex, …) to read and coach you from. The tool ships the
instructions for that agent.

## Quick start

You need [Go](https://go.dev/dl/) and [7-Zip](https://www.7-zip.org/). 7-Zip must be on your `PATH` as `7z`, `7zz` or
`7za`; on Windows, add `C:\Program Files\7-Zip` to it.

**1. Install the tool.**

```sh
go install github.com/backbone81/bar-replay-analytics-go/cmd/bar-replay-analytics-go@latest
```

It lands in `go/bin/` in your home folder. Add that folder to your `PATH` if `bar-replay-analytics-go` is not found.

**2. Find your BAR data folder.** It holds `demos/`, where BAR saves every game you play. Usually it is:

| Install | BAR data folder |
| --- | --- |
| Windows | `%LOCALAPPDATA%\Programs\Beyond-All-Reason\data` |
| Linux, AppImage | `~/.local/state/Beyond All Reason` |
| Linux, Flatpak | `~/.var/app/info.beyondallreason.bar/data` |

Below, `<BAR data>` stands for that folder.

**3. Get coached.** Make a folder for your coaching, set it up for the agent and start your AI agent there:

```sh
bar-replay-analytics-go init
```

This writes the agent's instructions (`AGENTS.md`), templates of the player profile and training log it keeps, and a
`work/` folder. It never overwrites a file: existing ones are reported as an error.

Ask the agent to analyze your last game. It asks you a few questions about yourself first, then plays your replays
back, reads their reports and keeps a training log, so later games are compared with earlier ones. Playing back a
one-hour game takes about one and a half minutes; the first playback also downloads the engine, game version and map the replay needs, as the BAR lobby does.

## Usage

Every command takes `--bar-data-dir "<BAR data>"`. A replay is the name of a file in `demos/` or a path to one. Every
command explains its options with `--help`.

```sh
# Analyze replays
bar-replay-analytics-go replay play <replay>...         # play back, several at a time
bar-replay-analytics-go replay report <replay>          # the report (needs the playback)
bar-replay-analytics-go replay script <replay>          # map, players, AIs and game options, no playback needed
bar-replay-analytics-go replay inspect <replay>         # everything the replay file holds

# Leaderboards and replays of strong players
bar-replay-analytics-go leaderboard show [--game-type duel] [--top 20]
bar-replay-analytics-go replay download-top [--game-type duel] [--top 10] [--days 7]
bar-replay-analytics-go replay download <replay-id>...  # the ID from https://www.bar-rts.com/replays/<replay-id>

# Set up the current folder for an AI coach
bar-replay-analytics-go init
```

Good to know:

- **Playback needs about 5 GB of memory per replay.** `replay play` plays several replays at once, as many as your
  CPU and memory allow; `--parallel` sets the number.
- **Playback leaves your BAR install alone.** It adds only what it downloads (engines, game versions, maps), like the
  lobby, and its results in `LuaUI/Widgets/bar-replay-analytics/events/`. Its analytics widget is built in and runs
  only in the playback. Your settings, widgets and `infolog.txt` stay untouched.
- **Games that crashed or did not end** (length 0:00 in the lobby) are skipped: their replay can run for hours.
  `--unended` plays them anyway.
- **Downloaded replays go to `demos/leaderboard/`**, so they don't crowd the replay list in the BAR lobby. Running
  `download-top` again fetches only what is new. `--game-type` is `duel`, `small-team`, `large-team` or `ffa`.

## Development

Needs Go and Docker (for the linter).

```sh
make build   # tidy, format, vet, lint, then build ./bar-replay-analytics-go
make test    # the same checks, then all tests
make fuzz    # run the fuzz tests until they stop finding new inputs (see scripts/fuzz.sh)
```

The template of the coaching agent's workspace that `init` writes is `internal/coaching/template/`, built into
the binary.
