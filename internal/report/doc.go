// Package report prepares the analytics of a replay: NewReport turns a demo.Demo into a Report.
//
// A Report is meant to be read as it is, by a person or an AI agent coaching a player: it aggregates the data, e.g.
// into intervals of Config.IntervalLength, so that its reader need not compute anything.
//
// So far a Report holds the replay's metadata from the DemoFileHeader and the winning teams, and per player the
// economy from its TeamStatistics history: totals and averages per second over the game and per interval. It does not
// read the setup script.
//
// Names follow the BAR UI, as a coach talks to a player, not the engine: a player is what the engine calls a team (the
// units and resources of one player or AI), a team is what the engine calls an allyteam (the players on one side).
// Player and team IDs are the engine's team and allyteam IDs, as the statistics and the winning allyteams use them.
// The setup script may number its team and allyteam sections with gaps; the engine numbers them anew without gaps, in
// the order of the script numbers (CGameSetup::LoadTeams in rts/Game/GameSetup.cpp).
//
// Times are game time since the start of the game, see Duration and GameSpeed.
package report
