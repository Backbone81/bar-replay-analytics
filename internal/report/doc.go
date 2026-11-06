// Package report prepares the analytics of a replay: NewReport turns a demo.Demo, its parsed setup script (tdf.Parse)
// and the events file of its playback (events.File) into a Report.
//
// A Report is read as it is, by a person or an AI agent coaching a player, so it holds what its reader would otherwise
// compute: totals over the game and aggregates per interval, finer in the opening (see Config). It holds:
//   - Metadata: engine and game version, map, start time, length, winning teams, game and map options.
//   - MapGrid: the cells loss locations refer to, and where each player started.
//   - Teams, and Players with, per player: name, faction, AI and income multiplier from the setup script; economy
//     from the TeamStatistics history; and from the events the metal extractors, builders and their idle time, units
//     killed and lost (also by unit type, with the enemy unit types which killed them), where units were lost,
//     resurrections, the opening build order and the first finished unit of each unit def.
//
// Names follow the BAR UI, as a coach talks to a player, not the engine: a player is the engine's team (the units and
// resources of one player or AI), a team is the engine's allyteam. Player and team IDs are the engine's team and
// allyteam IDs, as in the statistics and the winning allyteams: the setup script may number its sections with gaps,
// the engine numbers them anew without gaps in the order of their numbers (CGameSetup::LoadTeams in
// rts/Game/GameSetup.cpp), and so does NewReport.
//
// Times are game time since the start of the game, see Duration and GameSpeed.
package report
