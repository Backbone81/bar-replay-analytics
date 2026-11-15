package report

import (
	"fmt"
	"sort"
	"strconv"

	"github.com/backbone81/bar-replay-analytics/internal/events"
)

// MetalExtractors provides the number of metal extractors a player held, by tech level, so that expanding and
// upgrading show apart: a tech 2 metal extractor replaces a tech 1 one and extracts four times as much.
type MetalExtractors struct {
	// Tech1 is the number of tech 1 metal extractors: those with a tech level below 2, like Legion's Overcharged Metal
	// Extractor (tech level 1, twice the extraction of a Metal Extractor).
	Tech1 int `json:"tech1"`

	// Tech2 is the number of metal extractors of tech level 2 or more, like the Advanced Metal Extractor.
	Tech2 int `json:"tech2"`
}

// metalExtractorChange is the metal extractors a player held from frame on.
type metalExtractorChange struct {
	// frame is the frame from which on the player held them.
	frame int

	// held is the metal extractors the player held.
	held MetalExtractors
}

// metalExtractorHistory is how the metal extractors a player held changed over the game, oldest first.
type metalExtractorHistory []metalExtractorChange

// heldMetalExtractor is a finished metal extractor and who holds it.
type heldMetalExtractor struct {
	// playerID is the ID of the player holding it, or of another team, such as Gaia.
	playerID int

	// isTech2 is true for a metal extractor of tech level 2 or more.
	isTech2 bool
}

// newMetalExtractorHistories prepares the metal extractor history of each of numPlayers players, by player ID: the
// finished metal extractors (events.UnitDef.ExtractsMetal) a player held, from when they were finished or given to it
// until they died or were given away. Returns an error if the tech level of a metal extractor is not a number.
func newMetalExtractorHistories(eventsFile *events.File, numPlayers int) ([]metalExtractorHistory, error) {
	histories := make([]metalExtractorHistory, numPlayers)
	// The finished metal extractors by unit ID, of all teams: one of a team which is no player may be given to one.
	extractors := make(map[int]heldMetalExtractor)
	for _, event := range eventsFile.Events {
		switch {
		case event.UnitFinished != nil:
			finished := event.UnitFinished
			unitDef := eventsFile.UnitDefs[finished.UnitDefID]
			if unitDef.ExtractsMetal <= 0 {
				continue
			}
			isTech2, err := isTech2(unitDef)
			if err != nil {
				return nil, fmt.Errorf("unit def %q: %w", unitDef.Name, err)
			}
			extractor := heldMetalExtractor{playerID: finished.UnitTeam, isTech2: isTech2}
			extractors[finished.UnitID] = extractor
			extractor.addTo(histories, event.Frame, 1)
		case event.UnitDestroyed != nil:
			if extractor, ok := extractors[event.UnitDestroyed.UnitID]; ok {
				delete(extractors, event.UnitDestroyed.UnitID)
				extractor.addTo(histories, event.Frame, -1)
			}
		case event.UnitGiven != nil:
			if extractor, ok := extractors[event.UnitGiven.UnitID]; ok {
				extractor.addTo(histories, event.Frame, -1)
				extractor.playerID = event.UnitGiven.NewTeam
				extractors[event.UnitGiven.UnitID] = extractor
				extractor.addTo(histories, event.Frame, 1)
			}
		}
	}
	return histories, nil
}

// addTo adds delta to the metal extractors of e's tech level which its holder holds from frame on, in histories by
// player ID, unless the holder is no player.
func (e heldMetalExtractor) addTo(histories []metalExtractorHistory, frame int, delta int) {
	if e.playerID < 0 || e.playerID >= len(histories) {
		return
	}
	histories[e.playerID] = histories[e.playerID].add(frame, e.isTech2, delta)
}

// isTech2 returns true if unitDef has a tech level (customParams "techlevel") of 2 or more. BAR sets tech level 1 for
// unit defs which have none (gamedata/alldefs_post.lua), and so does isTech2. Returns an error if the tech level is not
// a number.
func isTech2(unitDef events.UnitDef) (bool, error) {
	techLevel, ok := unitDef.CustomParams["techlevel"]
	if !ok {
		return false, nil
	}
	level, err := strconv.ParseFloat(techLevel, 64)
	if err != nil {
		return false, fmt.Errorf("tech level: %w", err)
	}
	return level >= 2, nil
}

// add returns h with delta added to the metal extractors of a tech level held from frame on: tech 2 if isTech2, else
// tech 1. frame must not be before the frame of the last change. Several changes in one frame make one change.
func (h metalExtractorHistory) add(frame int, isTech2 bool, delta int) metalExtractorHistory {
	var held MetalExtractors
	if len(h) > 0 {
		held = h[len(h)-1].held
	}
	if isTech2 {
		held.Tech2 += delta
	} else {
		held.Tech1 += delta
	}
	if len(h) > 0 && h[len(h)-1].frame == frame {
		h[len(h)-1].held = held
		return h
	}
	return append(h, metalExtractorChange{frame: frame, held: held})
}

// at returns the metal extractors held at the end of frame: after the changes in frame.
func (h metalExtractorHistory) at(frame int) MetalExtractors {
	i := sort.Search(len(h), func(i int) bool { return h[i].frame > frame })
	if i == 0 {
		return MetalExtractors{}
	}
	return h[i-1].held
}
