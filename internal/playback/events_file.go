package playback

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"

	"github.com/backbone81/bar-replay-analytics-go/internal/events"
	"github.com/backbone81/bar-replay-analytics-go/internal/replayplayer/widget"
)

// EventsFileStatus is whether the events file of a replay's playback is current.
type EventsFileStatus struct {
	// Current is true if the events file is complete (it ends with the game over) and has at least the format version
	// of the embedded widget (widget.FormatVersion), so playing the replay back again would write the same events.
	Current bool

	// FormatVersion is the format version of the events file, 0 if it has none or there is no readable file.
	FormatVersion int

	// Reason describes the status, such as "current (format 1)" or "outdated (format 0, the widget writes 1)".
	Reason string
}

// CheckEventsFile returns whether the events file of the playback of the replay file replayFilePath
// (EventsFilePath) is current. A missing, unreadable or incomplete file is not. Returns an error only if the file
// exists but cannot be read.
func CheckEventsFile(replayFilePath string) (EventsFileStatus, error) {
	data, err := os.ReadFile(EventsFilePath(replayFilePath))
	if errors.Is(err, fs.ErrNotExist) {
		return EventsFileStatus{Reason: "no events file"}, nil
	}
	if err != nil {
		return EventsFileStatus{}, err
	}
	// Only the fields needed here; json.Unmarshal still checks that the whole file is valid JSON.
	var file struct {
		FormatVersion int         `json:"formatVersion"`
		End           *events.End `json:"end"`
	}
	if err := json.Unmarshal(data, &file); err != nil {
		//nolint:nilerr // An events file cut off by a crash is not current; playing the replay again replaces it.
		return EventsFileStatus{Reason: "events file unreadable: " + err.Error()}, nil
	}
	status := EventsFileStatus{FormatVersion: file.FormatVersion}
	switch {
	case file.End == nil || file.End.Reason != events.EndReasonGameOver:
		status.Reason = "events file incomplete"
	case file.FormatVersion < widget.FormatVersion:
		status.Reason = fmt.Sprintf("events file outdated (format %d, the widget writes %d)", file.FormatVersion,
			widget.FormatVersion)
	default:
		status.Current = true
		status.Reason = fmt.Sprintf("events file current (format %d)", file.FormatVersion)
	}
	return status, nil
}
