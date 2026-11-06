package report

import (
	"math"

	"github.com/backbone81/bar-replay-analytics-go/internal/events"
)

// Combat is what a player killed and lost, from the units which died in the events.
//
// A unit is lost when it dies in any way: killed by an enemy, by an ally's or the player's own fire, self-destructed,
// crushed, decayed as an abandoned nanoframe. Its metal is gone either way. Units whose metal came back are not lost:
// those the player or an ally reclaimed, and metal extractors replaced by an upgrade, whose cost BAR refunds
// (unit_mex_upgrade_reclaimer.lua). Units are counted unfinished too, with the metal of their build progress, as
// partly built units cost what was spent on them. A reclaimed unit has build progress 0 when it dies, as the engine
// takes it to 0 before it kills the unit (CUnit::AddBuildPower in rts/Sim/Units/Unit.cpp), so a finished one counts
// its whole metal cost, an unfinished one 0: what it was built to before the reclaim is not known.
//
// A unit is killed by a player when one of the player's units killed it and it belonged to another team (or to Gaia),
// in the same ways a unit is lost, reclaiming included: an enemy unit reclaimed is one the enemy lost.
//
// Units removed because their player died (events.TeamDied: resigned, left or wiped out) are neither lost nor killed:
// BAR removes them, they did not die in a fight.
type Combat struct {
	// Total provides what the player killed and lost over the whole game.
	Total Casualties `json:"total"`

	// Intervals provides what the player killed and lost in the intervals of Economy.Intervals. Deaths at the end of
	// an interval count in it, deaths after the last one in the last one.
	// Is empty for a game of no length (Metadata.Duration).
	Intervals []CombatInterval `json:"intervals"`
}

// CombatInterval is what a player killed and lost in one interval of the game.
type CombatInterval struct {
	Casualties

	// Time is the span of the game this interval covers.
	Time Interval `json:"time"`
}

// Casualties provides the units a player killed and lost.
type Casualties struct {
	// Killed provides the units of other teams the player killed.
	Killed DeadUnits `json:"killed"`

	// Lost provides the player's units which died.
	Lost DeadUnits `json:"lost"`
}

// DeadUnits provides how many units died and what they were worth.
type DeadUnits struct {
	// Count is the number of units.
	Count int `json:"count"`

	// Metal is the metal spent on them, rounded to whole metal: each unit's metal cost times its build progress (for a
	// reclaimed unit see Combat).
	Metal float64 `json:"metal"`
}

// casualty is a unit which died, as a kill or a loss of a player.
type casualty struct {
	// frame is the frame the unit died at.
	frame int

	// isKill is true for a unit the player killed, false for one it lost.
	isKill bool

	// metal is the metal spent on the unit.
	metal float64
}

// combatHistory provides the casualties of a player, oldest first.
type combatHistory []casualty

// combatTracker follows the units which die through the events.
type combatTracker struct {
	// players provides the players, by player ID.
	players []Player

	// histories provides the combat history of each player, by player ID.
	histories []combatHistory

	// died provides the frames the dead teams died at, by team ID.
	died map[int]int

	// finished provides the unit IDs of the units alive and finished. Unit IDs are reused once a unit died, so each
	// UnitCreated takes its ID out.
	finished map[int]bool
}

// newCombatHistories prepares the combat history of each player, by player ID, from the units which died in the
// events. A player's units are allied with those of the players in its team (Player.TeamID).
func newCombatHistories(eventsFile *events.File, players []Player) []combatHistory {
	tracker := combatTracker{
		players:   players,
		histories: make([]combatHistory, len(players)),
		died:      make(map[int]int),
		finished:  make(map[int]bool),
	}
	for _, event := range eventsFile.Events {
		switch {
		case event.UnitCreated != nil:
			delete(tracker.finished, event.UnitCreated.UnitID)
		case event.UnitFinished != nil:
			tracker.finished[event.UnitFinished.UnitID] = true
		case event.TeamDied != nil:
			if _, ok := tracker.died[event.TeamDied.TeamID]; !ok {
				tracker.died[event.TeamDied.TeamID] = event.Frame
			}
		case event.UnitDestroyed != nil:
			destroyed := event.UnitDestroyed
			tracker.addDeath(event.Frame, destroyed, eventsFile.UnitDefs[destroyed.UnitDefID])
		}
	}
	return tracker.histories
}

// addDeath adds the destroyed unit of unitDef, which died at frame, as a kill to the player whose unit killed it,
// unless that player is its own or an ally, and as a loss to its player, unless its metal came back (isRefunded). A
// unit of a dead team is neither. A reclaimed unit which was finished counts with build progress 1 (see Combat).
func (t *combatTracker) addDeath(frame int, destroyed *events.UnitDestroyed, unitDef events.UnitDef) {
	if diedAt, ok := t.died[destroyed.UnitTeam]; ok && frame >= diedAt {
		return
	}
	buildProgress := destroyed.BuildProgress
	if destroyed.WeaponDefID == events.EnvDamageReclaimed && t.finished[destroyed.UnitID] {
		buildProgress = 1
	}
	metal := unitDef.MetalCost * buildProgress
	attacker := -1
	if destroyed.AttackerTeam != nil && t.isPlayer(*destroyed.AttackerTeam) {
		attacker = *destroyed.AttackerTeam
	}
	isAllied := t.isPlayer(destroyed.UnitTeam) &&
		attacker >= 0 &&
		t.players[attacker].TeamID == t.players[destroyed.UnitTeam].TeamID
	if attacker >= 0 && !isAllied {
		t.histories[attacker] = append(t.histories[attacker], casualty{frame: frame, isKill: true, metal: metal})
	}
	if !t.isPlayer(destroyed.UnitTeam) || isRefunded(destroyed, unitDef, isAllied) {
		return
	}
	t.histories[destroyed.UnitTeam] = append(t.histories[destroyed.UnitTeam], casualty{frame: frame, metal: metal})
}

// isPlayer returns true if the team with teamID is a player, not Gaia.
func (t *combatTracker) isPlayer(teamID int) bool {
	return teamID >= 0 && teamID < len(t.players)
}

// isRefunded returns true if the metal of the destroyed unit of unitDef came back to its team or an ally: if the unit
// was reclaimed by its own or an allied team (isAllied), or is a metal extractor a gadget killed without attacker,
// which BAR does when an upgrade replaces it.
func isRefunded(destroyed *events.UnitDestroyed, unitDef events.UnitDef, isAllied bool) bool {
	switch destroyed.WeaponDefID {
	case events.EnvDamageReclaimed:
		return isAllied
	case events.EnvDamageKilledByLua:
		return unitDef.ExtractsMetal > 0 && destroyed.AttackerID == nil
	}
	return false
}

// newCombat prepares the combat from a player's combat history, in the intervals of spans (newIntervalSpans).
func newCombat(casualties combatHistory, spans []frameSpan) Combat {
	combat := Combat{Intervals: []CombatInterval{}}
	for _, c := range casualties {
		combat.Total.add(c)
	}
	combat.Total = combat.Total.rounded()
	next := 0
	for i, span := range spans {
		interval := CombatInterval{Time: span.interval()}
		isLast := i == len(spans)-1
		for ; next < len(casualties) && (isLast || casualties[next].frame <= span.end); next++ {
			interval.add(casualties[next])
		}
		interval.Casualties = interval.rounded()
		combat.Intervals = append(combat.Intervals, interval)
	}
	return combat
}

// add adds a casualty to the units killed or lost.
func (c *Casualties) add(casualty casualty) {
	dead := &c.Lost
	if casualty.isKill {
		dead = &c.Killed
	}
	dead.Count++
	dead.Metal += casualty.metal
}

// rounded returns c with the metal rounded to whole metal.
func (c Casualties) rounded() Casualties {
	c.Killed.Metal = math.Round(c.Killed.Metal)
	c.Lost.Metal = math.Round(c.Lost.Metal)
	return c
}
