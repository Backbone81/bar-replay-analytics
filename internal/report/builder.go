package report

import (
	"fmt"
	"math"
	"slices"
	"strconv"

	"github.com/backbone81/bar-replay-analytics-go/internal/events"
)

// Builder is one commander or constructor of a player, while the player held it: from when it was finished or given to
// the player until it died, was given away or the game ended. Nano turrets and factories are no builders here: nano
// turrets often stand behind the front as repair stations, so their idle time is no mistake, and constructors are
// what starts new buildings.
//
// A constructor is a mobile builder which is no factory and cannot resurrect, so resurrection bots are left out. The
// events do not tell what a builder can build, so a minelayer or a spy, mobile builders which cannot build a metal
// extractor, count as constructors.
type Builder struct {
	// Name is the name the BAR UI shows, numbered per name in the order the player got them, like "Construction Bot 3".
	Name string `json:"name"`

	// Time is the span the player held this builder.
	Time Interval `json:"time"`

	// Idle is how long this builder was idle: from when its command queue ran empty (events.UnitIdle) until a command
	// left it non-empty (events.UnitCommand.CommandCount), so a state or stop given to an idle builder keeps it idle.
	// Walking to a build site or guarding counts as busy.
	Idle Duration `json:"idle"`

	// IdleFraction is Idle as a fraction of Time, from 0 to 1, rounded to two decimals: 0.25 means idle a quarter of
	// the time.
	IdleFraction float64 `json:"idleFraction"`

	// IdleStretches provides the spans this builder was idle for at least Config.IdleStretchLength, oldest first.
	IdleStretches []Interval `json:"idleStretches"`

	// Builds provides what this builder started building and was finished, counted per unit def, most first, of the
	// same count the one first finished first. Resurrected units are left out (see OpeningBuildOrder).
	Builds []BuildCount `json:"builds"`
}

// builderClass is the kind of a builder, by which BuilderClasses adds up their use. The report does not show it per
// builder: the name tells it, like "Advanced Construction Bot".
type builderClass string

const (
	// builderClassCommander is a commander.
	builderClassCommander builderClass = "commander"

	// builderClassTech1 is a constructor of tech level below 2, like the Construction Bot.
	builderClassTech1 builderClass = "tech1"

	// builderClassTech2 is a constructor of tech level 2 or more, like the Advanced Construction Bot.
	builderClassTech2 builderClass = "tech2"
)

// BuildCount is how many units of one unit def a builder built.
type BuildCount struct {
	// Name is the name the BAR UI shows, like "Wind Turbine".
	Name string `json:"name"`

	// Count is how many the builder built.
	Count int `json:"count"`
}

// BuilderClasses provides how the builders of a player were used in an interval, by class.
type BuilderClasses struct {
	// Commander provides the use of the commanders.
	Commander BuilderUse `json:"commander"`

	// Tech1 provides the use of the tech 1 constructors.
	Tech1 BuilderUse `json:"tech1"`

	// Tech2 provides the use of the tech 2 constructors.
	Tech2 BuilderUse `json:"tech2"`
}

// BuilderUse is how builders of one class were used in an interval, as times summed over these builders and divided by
// the interval's length, rounded to two decimals. Dividing by the length makes intervals of different lengths
// comparable, and Idle comparable with Alive: Idle / Alive is the fraction of the builders' time they were idle.
type BuilderUse struct {
	// Alive is the summed time the player held these builders in the interval, divided by its length: the average
	// number held.
	Alive float64 `json:"alive"`

	// Idle is the summed time these builders were idle (Builder.Idle) in the interval, divided by its length: 2.5 means
	// as much idle time as two and a half builders idle the whole interval, e.g. five idle half of it.
	Idle float64 `json:"idle"`
}

// builderLife is one builder while a player held it.
type builderLife struct {
	// unitID is the builder's unit ID.
	unitID int

	// unitDefID is the builder's unit def.
	unitDefID int

	// class is the kind of builder.
	class builderClass

	// span is the span the player held the builder.
	span frameSpan

	// idle provides the spans the builder was idle, oldest first.
	idle []frameSpan

	// builds provides how many units the builder built, by unit def.
	builds map[int]int

	// buildOrder provides the unit defs of builds in the order their first unit was finished.
	buildOrder []int
}

// builderActivity is what the builders of one player did.
type builderActivity struct {
	// lives provides the player's builders, in the order the player got them.
	lives []*builderLife
}

// builderTracker follows the builders of the players through the events.
type builderTracker struct {
	// unitDefs provides the unit defs of the events file.
	unitDefs map[int]events.UnitDef

	// activities provides the builder activity of each player, by player ID.
	activities []builderActivity

	// held provides the builders the players hold, by unit ID.
	held map[int]*builderLife

	// idleSince provides the frame each held builder is idle since, by unit ID. A busy builder has none.
	idleSince map[int]int

	// finished provides the finished units of all teams by unit ID: a builder of a team which is no player may be
	// given to one.
	finished map[int]bool
}

// newBuilderActivities prepares the builder activity of each of numPlayers players, by player ID, from the events:
// which builders each player held when, when they were idle, and what they built (see builtUnits). Builders held when
// the events end are held until then. Returns an error if the tech level of a builder is not a number.
func newBuilderActivities(eventsFile *events.File, numPlayers int) ([]builderActivity, error) {
	tracker := builderTracker{
		unitDefs:   eventsFile.UnitDefs,
		activities: make([]builderActivity, numPlayers),
		held:       make(map[int]*builderLife),
		idleSince:  make(map[int]int),
		finished:   make(map[int]bool),
	}
	for _, event := range eventsFile.Events {
		switch {
		case event.UnitCreated != nil:
			delete(tracker.finished, event.UnitCreated.UnitID)
		case event.UnitFinished != nil:
			finished := event.UnitFinished
			tracker.finished[finished.UnitID] = true
			if err := tracker.start(event.Frame, finished.UnitID, finished.UnitDefID, finished.UnitTeam); err != nil {
				return nil, err
			}
		case event.UnitDestroyed != nil:
			tracker.end(event.Frame, event.UnitDestroyed.UnitID)
			delete(tracker.finished, event.UnitDestroyed.UnitID)
		case event.UnitGiven != nil:
			given := event.UnitGiven
			tracker.end(event.Frame, given.UnitID)
			if !tracker.finished[given.UnitID] {
				continue
			}
			if err := tracker.start(event.Frame, given.UnitID, given.UnitDefID, given.NewTeam); err != nil {
				return nil, err
			}
		case event.UnitIdle != nil:
			tracker.setIdle(event.Frame, event.UnitIdle.UnitID)
		case event.UnitCommand != nil:
			if event.UnitCommand.CommandCount > 0 {
				tracker.setBusy(event.Frame, event.UnitCommand.UnitID)
			}
		}
	}
	for unitID := range tracker.held {
		tracker.end(eventsFile.End.Frame, unitID)
	}
	for unit := range builtUnits(eventsFile, numPlayers) {
		tracker.addBuild(unit)
	}
	return tracker.activities, nil
}

// start starts the life of a builder the player with playerID got at frame, if it is a builder and the player is one.
// Returns an error if the tech level of the builder is not a number.
func (t *builderTracker) start(frame int, unitID int, unitDefID int, playerID int) error {
	if playerID < 0 || playerID >= len(t.activities) {
		return nil
	}
	unitDef := t.unitDefs[unitDefID]
	class, ok, err := builderClassOf(unitDef)
	if err != nil {
		return fmt.Errorf("unit def %q: %w", unitDef.Name, err)
	}
	if !ok {
		return nil
	}
	life := &builderLife{
		unitID:    unitID,
		unitDefID: unitDefID,
		class:     class,
		span:      frameSpan{start: frame, end: frame},
		builds:    make(map[int]int),
	}
	t.activities[playerID].lives = append(t.activities[playerID].lives, life)
	t.held[unitID] = life
	return nil
}

// end ends the life of the builder with unitID at frame, if a player holds it.
func (t *builderTracker) end(frame int, unitID int) {
	life, ok := t.held[unitID]
	if !ok {
		return
	}
	t.setBusy(frame, unitID)
	life.span.end = frame
	delete(t.held, unitID)
}

// setIdle sets the builder with unitID idle from frame on, if a player holds it and it is busy.
func (t *builderTracker) setIdle(frame int, unitID int) {
	if _, ok := t.held[unitID]; !ok {
		return
	}
	if _, ok := t.idleSince[unitID]; !ok {
		t.idleSince[unitID] = frame
	}
}

// setBusy sets the builder with unitID busy from frame on, if a player holds it and it is idle.
func (t *builderTracker) setBusy(frame int, unitID int) {
	since, ok := t.idleSince[unitID]
	if !ok {
		return
	}
	delete(t.idleSince, unitID)
	if frame > since {
		life := t.held[unitID]
		life.idle = append(life.idle, frameSpan{start: since, end: frame})
	}
}

// addBuild counts unit for the builder which started it, if a player held that builder then. Unit IDs are reused once
// a unit died, so the builder is the life with its unit ID which spans the start.
func (t *builderTracker) addBuild(unit builtUnit) {
	for _, activity := range t.activities {
		for _, life := range activity.lives {
			if life.unitID != unit.builderID || unit.startFrame < life.span.start || unit.startFrame > life.span.end {
				continue
			}
			if life.builds[unit.unitDefID] == 0 {
				life.buildOrder = append(life.buildOrder, unit.unitDefID)
			}
			life.builds[unit.unitDefID]++
			return
		}
	}
}

// builders returns the player's builders, see Builder, with the idle stretches of at least
// Config.IdleStretchLength.
func (a builderActivity) builders(unitDefs map[int]events.UnitDef, config Config) []Builder {
	builders := make([]Builder, 0, len(a.lives))
	numbers := make(map[string]int)
	minStretch := int(frames(config.IdleStretchLength))
	for _, life := range a.lives {
		name := unitDefs[life.unitDefID].TranslatedHumanName
		numbers[name]++
		idle := 0
		stretches := []Interval{}
		for _, span := range life.idle {
			idle += span.end - span.start
			if span.end-span.start >= minStretch {
				stretches = append(stretches, span.interval())
			}
		}
		idleFraction := 0.0
		if length := life.span.end - life.span.start; length > 0 {
			idleFraction = roundTo2(float64(idle) / float64(length))
		}
		builders = append(builders, Builder{
			Name:          name + " " + strconv.Itoa(numbers[name]),
			Time:          life.span.interval(),
			Idle:          frameDuration(int32(idle)),
			IdleFraction:  idleFraction,
			IdleStretches: stretches,
			Builds:        life.buildCounts(unitDefs),
		})
	}
	return builders
}

// buildCounts returns what the builder built, see Builder.Builds.
func (l builderLife) buildCounts(unitDefs map[int]events.UnitDef) []BuildCount {
	counts := make([]BuildCount, 0, len(l.buildOrder))
	for _, unitDefID := range l.buildOrder {
		counts = append(counts, BuildCount{Name: unitDefs[unitDefID].TranslatedHumanName, Count: l.builds[unitDefID]})
	}
	slices.SortStableFunc(counts, func(a, b BuildCount) int { return b.Count - a.Count })
	return counts
}

// use returns how the player's builders were used in span, see BuilderClasses.
func (a builderActivity) use(span frameSpan) BuilderClasses {
	var use BuilderClasses
	if span.end <= span.start {
		return use
	}
	length := float64(span.end - span.start)
	for _, life := range a.lives {
		classUse := use.of(life.class)
		classUse.Alive += float64(life.span.overlap(span)) / length
		for _, idle := range life.idle {
			classUse.Idle += float64(idle.overlap(span)) / length
		}
	}
	for _, classUse := range []*BuilderUse{&use.Commander, &use.Tech1, &use.Tech2} {
		classUse.Alive = roundTo2(classUse.Alive)
		classUse.Idle = roundTo2(classUse.Idle)
	}
	return use
}

// of returns the use of the builders of class.
func (c *BuilderClasses) of(class builderClass) *BuilderUse {
	switch class {
	case builderClassCommander:
		return &c.Commander
	case builderClassTech1:
		return &c.Tech1
	case builderClassTech2:
		return &c.Tech2
	}
	panic(fmt.Sprintf("unknown builder class %q", class))
}

// builderClassOf returns the class of a unit def and true if it is a commander or constructor (see Builder), else
// false. Returns an error if the tech level of a constructor is not a number.
func builderClassOf(unitDef events.UnitDef) (builderClass, bool, error) {
	if !unitDef.IsBuilder || !unitDef.CanMove || unitDef.IsFactory || unitDef.CanResurrect {
		return "", false, nil
	}
	// BAR checks only whether a unit def has the param (e.g. luarules/gadgets/game_logger.lua).
	if _, ok := unitDef.CustomParams["iscommander"]; ok {
		return builderClassCommander, true, nil
	}
	isTech2, err := isTech2(unitDef)
	if err != nil {
		return "", false, err
	}
	if isTech2 {
		return builderClassTech2, true, nil
	}
	return builderClassTech1, true, nil
}

// roundTo2 returns x rounded to two decimals.
func roundTo2(x float64) float64 {
	return math.Round(x*100) / 100
}
