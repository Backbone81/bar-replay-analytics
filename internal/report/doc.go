// Package report prepares the analytics of a replay: NewReport turns a demo.Demo, its parsed setup script and the
// events file of its playback (events.File) into a Report. It only interprets the script, as the engine and BAR read
// it; parsing it is up to the caller (tdf.Parse).
//
// A Report is meant to be read as it is, by a person or an AI agent coaching a player: it aggregates the data, e.g.
// into intervals, finer in the opening of the game (see Config), so that its reader need not compute anything.
//
// A Report holds the replay's metadata from the DemoFileHeader, the winning teams and the setup script (map, game
// version and the game options as the script holds them), the teams, and per player its name, income multiplier and AI
// from the setup script, its economy from its TeamStatistics history (totals and averages per second over the game and
// per interval), and from the events the units it killed and lost with their metal (in total and per interval of the
// economy), the metal extractors it held at the end of each interval, the faction it played, its build order in the
// opening (the units and buildings it finished) and when it finished its first unit of each unit def over the game.
// NewReport first takes the faction chosen in the lobby from the setup script, then overwrites it with the one of the
// player's start unit in the events, so a disagreement shows when stepping through it.
//
// The game options (Metadata.ModOptions) are attached as they are, not interpreted: there are hundreds, and which of
// them make a game non-standard depends on what it is compared for. A game started from the local lobby holds only the
// options changed from their defaults in BAR's modoptions.lua, a hosted game holds every option.
//
// Names follow the BAR UI, as a coach talks to a player, not the engine: a player is what the engine calls a team (the
// units and resources of one player or AI), a team is what the engine calls an allyteam (the players on one side).
// Player and team IDs are the engine's team and allyteam IDs, as the statistics and the winning allyteams use them.
// The setup script may number its team and allyteam sections with gaps; the engine numbers them anew without gaps, in
// the order of the script numbers (CGameSetup::LoadTeams in rts/Game/GameSetup.cpp), and so does NewReport.
//
// Times are game time since the start of the game, see Duration and GameSpeed.
package report
