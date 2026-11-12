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
	// FormatVersion is the version of the file's format, the widget's FORMAT_VERSION (see widget.FormatVersion).
	FormatVersion int `json:"formatVersion"`

	// Events provides the events in frame order.
	Events []Event `json:"events"`

	// Game provides the game that was played.
	Game Game `json:"game"`

	// Sides provides the sides of the game.
	// Is empty in files of older widgets.
	Sides []Side `json:"sides"`

	// Teams provides every team but Gaia.
	Teams []Team `json:"teams"`

	// TeamStats provides every team's statistics history by teamID, an entry every 15 s.
	TeamStats map[int][]TeamStats `json:"teamStats"`

	// UnitDefs provides the unit defs the events and teams refer to, by unitDefID.
	UnitDefs map[int]UnitDef `json:"unitDefs"`

	// End provides when and why the widget stopped logging.
	// Is nil in an incomplete file, which Read rejects.
	End *End `json:"end"`
}

// Read parses an events file and validates it with File.Validate.
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

// ReadFile reads the events file at filePath with Read.
func ReadFile(filePath string) (*File, error) {
	file, err := os.Open(filePath) //nolint:gosec // Opening the file the caller names is the purpose of ReadFile.
	if err != nil {
		return nil, err
	}
	defer file.Close() //nolint:errcheck
	return Read(file)
}

// ReadFormatVersion parses only the format version and the end of an events file, which is enough to tell whether it is
// complete and current, at a fraction of the time and memory of Read. The returned File has every other field zero.
// Like Read, it returns an error if the file is not valid JSON or its End is missing or not EndReasonGameOver
// (File.ValidateEnd); it does not validate anything else.
func ReadFormatVersion(reader io.Reader) (*File, error) {
	data, err := io.ReadAll(reader)
	if err != nil {
		return nil, fmt.Errorf("reading events: %w", err)
	}

	// json.Unmarshal still scans the whole document, so a file cut off by a crash is rejected.
	var header struct {
		FormatVersion int  `json:"formatVersion"`
		End           *End `json:"end"`
	}
	if err := json.Unmarshal(data, &header); err != nil {
		return nil, fmt.Errorf("parsing events: %w", err)
	}
	file := File{
		FormatVersion: header.FormatVersion,
		End:           header.End,
	}
	if err := file.ValidateEnd(); err != nil {
		return nil, err
	}
	return &file, nil
}

// ReadFormatVersionFile reads the events file at filePath with ReadFormatVersion. Like ReadFile, it reads the file
// with os.ReadFile.
func ReadFormatVersionFile(filePath string) (*File, error) {
	file, err := os.Open(filePath) //nolint:gosec // Opening the file the caller names is the purpose of ReadFile.
	if err != nil {
		return nil, err
	}
	defer file.Close() //nolint:errcheck
	return ReadFormatVersion(file)
}

// ValidateEnd returns an error if the file is incomplete (it has no End) or the game did not end.
func (f *File) ValidateEnd() error {
	if f.End == nil {
		return errors.New("events: no end, the events file is incomplete")
	}
	if f.End.Reason != EndReasonGameOver {
		return fmt.Errorf("events: ended at frame %d with reason %q, not %q", f.End.Frame, f.End.Reason, EndReasonGameOver)
	}
	return nil
}

// Validate returns an error if the file is incomplete, the game did not end, the map has no size, an event is invalid,
// the frames go backwards, or an event or team refers to a unit def that is not in UnitDefs.
func (f *File) Validate() error {
	if err := f.ValidateEnd(); err != nil {
		return err
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
