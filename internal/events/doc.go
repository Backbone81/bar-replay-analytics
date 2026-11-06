// Package events reads the events file, <replay name>_events.json, that the analytics widget
// (internal/playback/widget/template/dbg_replay_analytics.lua) writes while a replay plays back headless.
//
// The file is one JSON object:
//
//	{"events":[
//	{"frame":0,"unitCreated":{...}},
//	{"frame":0,"unitFinished":{...}},
//	...
//	],"game":{...},"sides":[...],"teams":[...],"teamStats":{"<teamID>":[...]},"unitDefs":{"<unitDefID>":{...}},
//	"end":{"frame":48227,"reason":"gameover"}}
//
// Each event has its game frame (GameSpeed per second) and one key, its name, that holds its fields. Events are named
// after the call-in they come from in lower camel case (unitCreated is widget:UnitCreated), fields after the call-in's
// parameters (unitTeam, attackerDefID). Event has one pointer per event type, so json.Unmarshal decodes it as is.
// Unknown event types and fields are ignored, so a newer widget's file can still be read.
//
// Every event comes from a call-in; the widget samples nothing. What does not change during the game, or what the
// engine records itself, is written once after the events: Game, Sides, Teams, TeamStats (every 15 s) and the UnitDefs
// the events and teams refer to.
//
// End says why the widget stopped logging. Read accepts only EndReasonGameOver: a file without End is incomplete (the
// engine crashed or was stopped), and any other reason means the rest of the game is missing.
//
// A field the widget leaves out is a pointer if its zero value is valid (e.g. team 0), else it reads as its zero value.
package events
