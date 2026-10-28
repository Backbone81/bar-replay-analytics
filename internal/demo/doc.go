// Package demo reads BAR demo files (.sdfz, called "replays" in the UI) as the Recoil engine writes them, see
// rts/System/LoadSave/DemoRecorder.cpp.
//
// A demo file is gzip-compressed. Decompressed, it holds these sections in this order:
//
//   - DemoFileHeader: the format version and the size of every following section.
//   - Start script (scriptSize bytes): map, game, players, teams, allyteams and AIs.
//   - Demo stream (demoStreamSize bytes): the recorded network messages. This package skips it.
//   - Winning allyteams (winningAllyTeamsSize bytes): one allyteam ID per byte.
//   - Player statistics: numPlayers PlayerStatistics, indexed by player ID, spectators included.
//   - Team statistics: numTeams entry counts (uint32), then the TeamStatistics history of each team in team order.
//     Gaia is not included.
//
// The layout comment in rts/System/LoadSave/demofile.h leaves out the winning allyteams; the order above is the
// writer's. All numbers are in ByteOrder.
//
// A demoStreamSize of 0 marks a demo whose engine crashed while recording: the demo stream runs to the end of the file
// and there are no statistics. Current engines write the whole file when the game ends, so a crash leaves an empty
// file instead.
//
// A team's statistics are recorded every teamStatPeriod seconds and hold totals since the start of the game. The last
// entry holds the totals at the end of the game, see TeamStatistics.Frame for its frame.
//
// Read validates every field and rejects the whole file on the first invalid one.
package demo
