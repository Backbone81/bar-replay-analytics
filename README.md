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

You need [Go](https://go.dev/dl/).

**1. Install the tool.**

```sh
go install github.com/backbone81/bar-replay-analytics-go/cmd/bar-replay-analytics-go@latest
```

It lands in `go/bin/` in your home folder. Add that folder to your `PATH` if `bar-replay-analytics-go` is not found.

**2. Check that it finds BAR.** The tool works in your BAR data folder, which holds `demos/`, where BAR saves every
game you play. It looks for it where the BAR installers put it:

| Install | BAR data folder |
| --- | --- |
| Windows | `%LOCALAPPDATA%\Programs\Beyond-All-Reason\data` |
| Linux, AppImage | `~/.local/state/Beyond All Reason` (`$XDG_STATE_HOME/Beyond All Reason`) |
| Linux, Flatpak | `~/.var/app/info.beyondallreason.bar/data` |

It uses the first of these that holds BAR. If BAR is somewhere else, the commands say so; then pass the folder with
`--bar-data-dir "<BAR data>"` to every command. Do the same if you have several BAR installs and want another one
than the first. `replay play` prints the folder it uses.

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

A replay is the name of a file in `demos/` or a path to one. `replay play` and `replay report` also take folders, such
as `leaderboard` for `demos/leaderboard/`: every `.sdfz` file directly in it. Every command explains its options with
`--help`.

```sh
# Analyze replays
bar-replay-analytics-go replay play <replay>...         # play back, several at a time: <replay>_events.json
bar-replay-analytics-go replay report <replay>...       # the reports (need the playback): <replay>_report.json
bar-replay-analytics-go replay script <replay>          # map, players, AIs and game options, no playback needed
bar-replay-analytics-go replay inspect <replay>         # everything the replay file holds

# Leaderboards and replays of strong players
bar-replay-analytics-go replay download-top [--game-type duel] [--top 10] [--days 7] [--map "<mapname>"] [--dry-run]

# Set up the current folder for an AI coach
bar-replay-analytics-go init
```

Good to know:

- **Playback needs about 5 GB of memory per replay.** `replay play` plays several replays at once, as many as your
  CPU and memory allow; `--parallel` sets the number.
- **Results go next to the replay:** `<replay>_events.json` and `<replay>.log` (the engine's output) from the
  playback, `<replay>_report.json` from the report. BAR's replay browser lists only the `.sdfz` files, so it does not
  show them.
- **Playback leaves your BAR install alone.** It adds only what it downloads (engines, game versions, maps), like the
  lobby, and its results next to the replays. Its analytics widget is built in and runs only in the playback. Your
  settings, widgets and `infolog.txt` stay untouched.
- **Only new or outdated replays are played back.** `replay play` skips a replay whose events file is complete and
  of the analytics widget's current format version (`formatVersion` in the file), so `replay play leaderboard` after
  a download plays only the new games, and after an update of the tool the ones its widget would log differently.
  `--force` plays them all again. Each report records its own `formatVersion` and the `eventsFormatVersion` it was
  made from; reports take seconds, so `replay report leaderboard` simply writes them all again.
- **Games that crashed or did not end** (length 0:00 in the lobby) are skipped: their replay can run for hours.
- **Downloaded replays go to `demos/leaderboard/`**, so they don't crowd the replay list in the BAR lobby. Running
  `download-top` again fetches only what is new; `--dry-run` logs the missing games of the players of the active
  season's leaderboard without downloading them. `--game-type` is `duel`, `small-team`, `large-team` or `ffa`;
  `--map` takes the exact mapname from `replay script`, version included, and can be repeated.

## Development

Needs Go and Docker (for the linter).

```sh
make build   # tidy, format, vet, lint, then build ./bar-replay-analytics-go
make test    # the same checks, then all tests
make fuzz    # run the fuzz tests until they stop finding new inputs (see scripts/fuzz.sh)
```

Two tests make sure the format versions are increased when the output changes:

- **Events:** `internal/replay/player` holds a hash of the widget's `.lua` files. When the widget
  changes, it fails and prints the new hash: if the events it writes change, increase `FORMAT_VERSION` in
  `dbg_replay_analytics.lua` (the only definition; Go reads it from there), then set the new hash.
- **Report:** `internal/report/testdata/report*.json` are the reports of a test game. When a report changes, the test
  fails: if that is intended, increase `report.FormatVersion` and update the files with
  `go test ./internal/report -update`.

The template of the coaching agent's workspace that `init` writes is `internal/coaching/template/`, built into
the binary.
