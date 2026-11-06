package events

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
)

// File is the content of an events file.
type File struct {
	// Events provides the events in the order the widget wrote them, which is the order of their frames.
	Events []Event `json:"events"`

	// Game provides the game that was played.
	Game Game `json:"game"`

	// Sides provides the sides of the game.
	// Is empty in a file of a widget from before it wrote them.
	Sides []Side `json:"sides"`

	// Teams provides every team but Gaia.
	Teams []Team `json:"teams"`

	// TeamStats provides every team's statistics history by teamID, an entry every 15 s.
	TeamStats map[int][]TeamStats `json:"teamStats"`

	// UnitDefs provides the unit defs the events and teams refer to, by unitDefID.
	UnitDefs map[int]UnitDef `json:"unitDefs"`

	// End provides when and why the widget stopped logging.
	// Is nil if the file is incomplete, but Read rejects such a file.
	End *End `json:"end"`
}

// Read reads an events file from reader, parses and validates it.
// Returns an error if the file is not valid JSON or validation fails.
func Read(reader io.Reader) (*File, error) {
	data, err := io.ReadAll(reader)
	if err != nil {
		return nil, fmt.Errorf("reading events: %w", err)
	}
	var file File
	if err := json.Unmarshal(data, &file); err != nil {
		return nil, fmt.Errorf("parsing events: %w", err)
	}
	if err := file.Validate(); err != nil {
		return nil, err
	}
	return &file, nil
}

// ReadFile opens the events file named by filePath and reads it with Read.
// Returns an error if the file cannot be opened or validation fails.
func ReadFile(filePath string) (*File, error) {
	file, err := os.Open(filePath) //nolint:gosec // Opening the file the caller names is the purpose of ReadFile.
	if err != nil {
		return nil, err
	}
	defer file.Close() //nolint:errcheck
	return Read(file)
}

// Validate returns an error if the file is incomplete, the game did not end, the map has no size, an event is invalid,
// the frames go backwards, or an event or team refers to a unit def that is not in UnitDefs.
func (f *File) Validate() error {
	if f.End == nil {
		return errors.New("events: no end, the events file is incomplete")
	}
	if f.End.Reason != EndReasonGameOver {
		return fmt.Errorf("events: ended at frame %d with reason %q, not %q", f.End.Frame, f.End.Reason, EndReasonGameOver)
	}
	if f.Game.MapSizeX <= 0 || f.Game.MapSizeZ <= 0 {
		return fmt.Errorf("events: map size %d x %d, want both > 0", f.Game.MapSizeX, f.Game.MapSizeZ)
	}
	lastFrame := 0
	for i, event := range f.Events {
		if err := event.Validate(); err != nil {
			return fmt.Errorf("events: event %d: %w", i, err)
		}
		if event.Frame < lastFrame {
			return fmt.Errorf("events: event %d: frame %d is before frame %d", i, event.Frame, lastFrame)
		}
		lastFrame = event.Frame
		for _, unitDefID := range event.UnitDefIDs() {
			if _, ok := f.UnitDefs[unitDefID]; !ok {
				return fmt.Errorf("events: event %d: unit def %d is not in unitDefs", i, unitDefID)
			}
		}
	}
	if f.End.Frame < lastFrame {
		return fmt.Errorf("events: end frame %d is before frame %d", f.End.Frame, lastFrame)
	}
	for _, team := range f.Teams {
		if team.StartUnit == nil {
			continue
		}
		if _, ok := f.UnitDefs[*team.StartUnit]; !ok {
			return fmt.Errorf("events: team %d: start unit def %d is not in unitDefs", team.TeamID, *team.StartUnit)
		}
	}
	return nil
}
