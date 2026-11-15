package report

import (
	"cmp"
	"math"
	"slices"

	"github.com/backbone81/bar-replay-analytics/internal/events"
)

// killedByLength is the most killers UnitTypeCasualties.KilledBy lists: the top few show what counters a unit type.
const killedByLength = 3

// Combat is what a player killed and lost, from the units which died in the events.
//
// A unit is lost when it dies in any way (enemy or friendly fire, self-destructed, crushed, an abandoned nanoframe
// decayed), unless its metal came back: reclaimed by the player or an ally, or a metal extractor replaced by an
// upgrade, which BAR refunds (unit_mex_upgrade_reclaimer.lua). Unfinished units count with the metal of their build
// progress. The engine takes a reclaimed unit to build progress 0 before it dies (CUnit::AddBuildPower in
// rts/Sim/Units/Unit.cpp), so a finished one counts its whole metal cost, an unfinished one 0.
//
// A unit is killed by a player when one of the player's units killed it and it was not the player's or an ally's
// (Gaia's counts), in the same ways, reclaiming included.
//
// Units of a player who died (events.TeamDied: resigned, left or wiped out) count neither way from then on: BAR
// removes them.
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

	// UnitTypes provides Killed and Lost by the player's unit types, those which killed or lost a unit, the most metal
	// killed and lost first. Their counts add up to Killed and Lost, their metal too, but for rounding.
	UnitTypes []UnitTypeCasualties `json:"unitTypes"`
}

// UnitTypeCasualties provides what the units of one unit type of a player killed and lost. Metal killed against metal
// lost tells how well the unit type traded.
type UnitTypeCasualties struct {
	// Name is the name the BAR UI shows, like "Thug". Unit defs of the same name, like the Metal Extractor of Armada
	// and Cortex, count as one.
	Name string `json:"name"`

	// Killed provides the units of other teams the units of this type killed.
	Killed DeadUnits `json:"killed"`

	// Lost provides the units of this type which died.
	Lost DeadUnits `json:"lost"`

	// KilledBy provides the names of the unit types of other teams which killed most of Lost, at most killedByLength,
	// the most metal first, over all enemy players. Units which died without such an attacker, e.g. self-destructed,
	// are left out.
	KilledBy []string `json:"killedBy"`
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

	// name is the UI name of the player's unit type: of the unit lost, or of the unit which killed.
	name string

	// killer is the UI name of the unit type of another team which killed the unit lost, empty if there is none.
	killer string

	// killerIsBuilding is true if killer is a building (see isBuilding), such as a static defence.
	killerIsBuilding bool

	// isBuilding is true if the unit lost is a building (see isBuilding).
	isBuilding bool

	// x and z are the position the unit lost died at, in elmos.
	x, z int
}

// combatHistory provides the casualties of a player, oldest first.
type combatHistory []casualty

// combatTracker follows the units which die through the events.
type combatTracker struct {
	// players provides the players, by player ID.
	players []Player

	// unitDefs provides the unit defs of the events file.
	unitDefs map[int]events.UnitDef

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
		unitDefs:  eventsFile.UnitDefs,
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
			tracker.addDeath(event.Frame, destroyed)
		}
	}
	return tracker.histories
}

// addDeath adds the destroyed unit, which died at frame, as a kill to the player whose unit killed it, unless that
// player is its own or an ally, and as a loss to its player, unless its metal came back (isRefunded). A unit of a dead
// team is neither. A reclaimed unit which was finished counts with build progress 1 (see Combat).
func (t *combatTracker) addDeath(frame int, destroyed *events.UnitDestroyed) {
	if diedAt, ok := t.died[destroyed.UnitTeam]; ok && frame >= diedAt {
		return
	}
	unitDef := t.unitDefs[destroyed.UnitDefID]
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
		kill := casualty{frame: frame, isKill: true, metal: metal}
		if destroyed.AttackerDefID != nil {
			kill.name = t.unitDefs[*destroyed.AttackerDefID].TranslatedHumanName
		}
		t.histories[attacker] = append(t.histories[attacker], kill)
	}
	if !t.isPlayer(destroyed.UnitTeam) || isRefunded(destroyed, unitDef, isAllied) {
		return
	}
	t.histories[destroyed.UnitTeam] = append(t.histories[destroyed.UnitTeam], t.loss(frame, destroyed, metal, isAllied))
}

// loss returns the casualty of the destroyed unit, which died at frame with metal spent on it, as a loss of its player.
// isAllied is true if its own player or an ally killed it: such an attacker is no killer to learn from.
func (t *combatTracker) loss(frame int, destroyed *events.UnitDestroyed, metal float64, isAllied bool) casualty {
	unitDef := t.unitDefs[destroyed.UnitDefID]
	loss := casualty{
		frame:      frame,
		metal:      metal,
		name:       unitDef.TranslatedHumanName,
		isBuilding: isBuilding(unitDef),
		x:          destroyed.X,
		z:          destroyed.Z,
	}
	if destroyed.AttackerDefID != nil && !isAllied {
		attackerDef := t.unitDefs[*destroyed.AttackerDefID]
		loss.killer = attackerDef.TranslatedHumanName
		loss.killerIsBuilding = isBuilding(attackerDef)
	}
	return loss
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

// isBuilding returns true if unitDef is a building: a unit which cannot move, or a factory, which BAR lets move so it
// can give its units move orders.
func isBuilding(unitDef events.UnitDef) bool {
	return !unitDef.CanMove || unitDef.IsFactory
}

// newCombat prepares the combat from a player's combat history, in the intervals of spans (newIntervalSpans).
func newCombat(casualties combatHistory, spans []frameSpan) Combat {
	combat := Combat{
		Total:     newCasualties(casualties),
		Intervals: []CombatInterval{},
	}
	next := 0
	for i, span := range spans {
		first := next
		isLast := i == len(spans)-1
		for next < len(casualties) && (isLast || casualties[next].frame <= span.end) {
			next++
		}
		combat.Intervals = append(combat.Intervals, CombatInterval{
			Time:       span.interval(),
			Casualties: newCasualties(casualties[first:next]),
		})
	}
	return combat
}

// unitTypeTally adds up the casualties of one unit type of a player.
type unitTypeTally struct {
	// casualties provides what the unit type killed and lost, with the metal not rounded.
	casualties UnitTypeCasualties

	// killers provides the units of the unit type lost, by the name of the unit type which killed them.
	killers map[string]*DeadUnits
}

// newCasualties returns the casualties of a player, see Casualties.
func newCasualties(casualties combatHistory) Casualties {
	var result Casualties
	tallies := make(map[string]*unitTypeTally)
	for _, c := range casualties {
		dead := &result.Lost
		if c.isKill {
			dead = &result.Killed
		}
		dead.add(c.metal)
		if c.name == "" {
			continue
		}
		tally, ok := tallies[c.name]
		if !ok {
			tally = &unitTypeTally{
				casualties: UnitTypeCasualties{Name: c.name},
				killers:    make(map[string]*DeadUnits),
			}
			tallies[c.name] = tally
		}
		if c.isKill {
			tally.casualties.Killed.add(c.metal)
			continue
		}
		tally.casualties.Lost.add(c.metal)
		if c.killer != "" {
			killer, ok := tally.killers[c.killer]
			if !ok {
				killer = &DeadUnits{}
				tally.killers[c.killer] = killer
			}
			killer.add(c.metal)
		}
	}
	result.Killed = result.Killed.rounded()
	result.Lost = result.Lost.rounded()
	result.UnitTypes = make([]UnitTypeCasualties, 0, len(tallies))
	for _, tally := range tallies {
		unitType := tally.casualties
		unitType.KilledBy = topKillers(tally.killers)
		unitType.Killed = unitType.Killed.rounded()
		unitType.Lost = unitType.Lost.rounded()
		result.UnitTypes = append(result.UnitTypes, unitType)
	}
	slices.SortFunc(result.UnitTypes, func(a, b UnitTypeCasualties) int {
		return cmp.Or(
			cmp.Compare(b.Killed.Metal+b.Lost.Metal, a.Killed.Metal+a.Lost.Metal),
			cmp.Compare(b.Killed.Count+b.Lost.Count, a.Killed.Count+a.Lost.Count),
			cmp.Compare(a.Name, b.Name),
		)
	})
	return result
}

// topKillers returns the names of the killedByLength killers which killed the most, by the units they killed: the most
// metal first, then the most units, then by name.
func topKillers(killers map[string]*DeadUnits) []string {
	names := make([]string, 0, len(killers))
	for name := range killers {
		names = append(names, name)
	}
	slices.SortFunc(names, func(a, b string) int {
		return cmp.Or(
			cmp.Compare(killers[b].Metal, killers[a].Metal),
			cmp.Compare(killers[b].Count, killers[a].Count),
			cmp.Compare(a, b),
		)
	})
	return names[:min(len(names), killedByLength)]
}

// add adds a unit with metal.
func (d *DeadUnits) add(metal float64) {
	d.Count++
	d.Metal += metal
}

// rounded returns d with the metal rounded to whole metal.
func (d DeadUnits) rounded() DeadUnits {
	d.Metal = math.Round(d.Metal)
	return d
}
