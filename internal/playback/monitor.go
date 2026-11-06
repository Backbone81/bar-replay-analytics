package playback

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"time"
)

// monitor watches a running playback; Play calls check every monitorInterval.
type monitor struct {
	player     *Player
	replay     *Replay
	eventsFile string
	logFile    string

	// start is when the engine was started.
	start time.Time

	// lastChange is when the watched file last grew.
	lastChange time.Time

	// lastProgress is when the progress was last reported.
	lastProgress time.Time

	// watched and watchedSize are the watched file and its size when last looked at.
	watched     string
	watchedSize int64
}

// check reports the progress every Config.ProgressInterval and returns why the engine must be stopped, or "": the
// watched file (the events file, or the engine log until the widget creates it) has not grown for
// Config.StallTimeout, or the widget logged that it cannot write its events file. Without the widget the replay plays
// in real time, so that stops the engine at once.
func (m *monitor) check(now time.Time) string {
	config := &m.player.config
	eventsStarted := isFile(m.eventsFile)
	watched := m.logFile
	if eventsStarted {
		watched = m.eventsFile
	}
	var size int64
	if info, err := os.Stat(watched); err == nil {
		size = info.Size()
	}
	if watched != m.watched || size != m.watchedSize {
		m.lastChange, m.watched, m.watchedSize = now, watched, size
	}

	if !eventsStarted && m.logContains(analyticsFailed) {
		return "the analytics widget could not write its events file"
	}
	if now.Sub(m.lastChange) > config.StallTimeout {
		output := "engine output"
		if eventsStarted {
			output = "events"
		}
		return fmt.Sprintf("no %s for %v", output, config.StallTimeout)
	}

	if now.Sub(m.lastProgress) >= config.ProgressInterval {
		m.lastProgress = now
		progress := Progress{Replay: m.replay, Elapsed: now.Sub(m.start), Loading: !eventsStarted}
		if eventsStarted {
			progress.Frame = m.lastFrame()
		}
		m.player.reportProgress(progress)
	}
	return ""
}

// logContains reports whether the engine log contains text. It reads the whole log, so check calls it only while the
// engine loads.
func (m *monitor) logContains(text string) bool {
	data, err := os.ReadFile(m.logFile)
	return err == nil && bytes.Contains(data, []byte(text))
}

// lastFrame returns the frame of the last complete event of the events file, which holds one event per line, or 0 if
// there is none yet.
func (m *monitor) lastFrame() int {
	file, err := os.Open(m.eventsFile)
	if err != nil {
		return 0
	}
	defer file.Close() //nolint:errcheck
	info, err := file.Stat()
	if err != nil {
		return 0
	}
	offset := max(0, info.Size()-lastFrameWindow)
	data, err := io.ReadAll(io.NewSectionReader(file, offset, info.Size()-offset))
	if err != nil {
		return 0
	}
	return LastFrameIn(data)
}
