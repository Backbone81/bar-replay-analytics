// Package coaching holds the instructions for an AI agent that coaches a player of Beyond All Reason from the analytics
// of their replays. They are embedded into the binary, so a user who installed it with go install can write them into
// the folder they run their agent in.
package coaching

import (
	_ "embed"
)

// AgentsMD is the content of AGENTS.md: the instructions for the coaching agent.
//
//go:embed AGENTS.md
var AgentsMD string
