// Package events reads the events file the analytics widget of bar-replay-analytics (dbg_replay_analytics.lua) writes
// while a replay is played back in a headless engine: <replay name>_events.json.
//
// The file is one JSON object:
//
//	{"events":[
//	{"frame":0,"unitCreated":{...}},
//	{"frame":0,"unitFinished":{...}},
//	...
//	],"game":{...},"teams":[...],"teamStats":{"<teamID>":[...]},"unitDefs":{"<unitDefID>":{...}},
//	"end":{"frame":48227,"reason":"gameover"}}
//
// Each event holds its game frame (GameSpeed frames per second) and its fields under one key, the event's name. Events
// are named after the call-in or Spring function they come from, in lower camel case (unitCreated is
// widget:UnitCreated), and so are their fields (unitTeam, attackerDefID). Event has one pointer per event type, so a
// plain json.Unmarshal decodes them; the one that is set tells the type. An event of a type this package does not know
// yet has no pointer set, so a newer widget's file can still be read. For the same reason, unknown fields are ignored.
//
// All events come from call-ins, the widget samples nothing. What does not change during the game, or
// the engine already records, is not in the events but written once, after them: the game (Game), the teams (Team),
// each team's statistics history (TeamStats, every 15 s) and the unit defs the events refer to (File.UnitDefs, by
// unitDefID as UnitDefs in Lua).
//
// The widget writes End when it stops logging, with the reason. Read accepts only a file whose game ended
// (EndReasonGameOver): a file without End is incomplete, the engine crashed or was stopped, and a file with any other
// reason misses the rest of the game.
//
// Fields the widget leaves out because they have no value are pointers where the zero value is a valid value (e.g.
// team 0), and zero values otherwise.
package events
