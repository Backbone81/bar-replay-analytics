package player

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/backbone81/bar-replay-analytics/internal/replay/player/widget"
)

// ErrAnalyticsWidgetNotActive is the cause with which a Watchdog stops a playback in which the analytics widget did
// not log widget.ActiveLogLine in time: it is not loaded or removed itself, so the replay plays in real time and the
// engine never quits, or the engine hangs before the widget is loaded.
var ErrAnalyticsWidgetNotActive = errors.New("the analytics widget did not become active")

// ErrAnalyticsWidgetHeartbeatMissing is the cause with which a Watchdog stops a playback in which the analytics widget
// stopped logging its heartbeat (widget.HeartbeatLogLine): the widget was removed (after a Lua error, the widget
// handler removes it), so the engine does not quit at the end, or the engine hangs.
var ErrAnalyticsWidgetHeartbeatMissing = errors.New("the analytics widget's heartbeat is missing")

// maxUnfinishedLineLength is how much of a line without its end yet the Watchdog keeps. A longer one is cut at the
// front: it is no line of the widget, whose log lines are short.
const maxUnfinishedLineLength = 64 * 1024

// Watchdog stops a playback in which the analytics widget gives no sign of life. It is an io.Writer for the engine's
// output that looks for the widget's log lines (widget.ActiveLogLine and widget.HeartbeatLogLine), and Watch cancels
// the playback when the first does not come within StartTimeout or the next not within HeartbeatTimeout. The widget
// itself quits when the replay stalls or plays in real time; the Watchdog covers what the widget cannot see.
//
// Write must not be called concurrently (exec.Cmd calls it from one goroutine when Stdout and Stderr are the same
// writer). Write and Watch may run concurrently.
type Watchdog struct {
	// startTimeout is how long Watch waits for the first sign of life.
	startTimeout time.Duration

	// heartbeatTimeout is how long Watch waits for each further sign of life.
	heartbeatTimeout time.Duration

	// signsOfLife receives a value for the widget's log lines; Write drops one while the last is not taken yet.
	signsOfLife chan struct{}

	// buffer is the output Write has not analyzed yet. Between calls, it holds what came after the last line end: the
	// start of a line whose end Write has not received yet.
	buffer []byte
}

// NewWatchdog returns a Watchdog that waits startTimeout for the analytics widget's first sign of life and
// heartbeatTimeout for each further one.
func NewWatchdog(startTimeout time.Duration, heartbeatTimeout time.Duration) *Watchdog {
	return &Watchdog{
		startTimeout:     startTimeout,
		heartbeatTimeout: heartbeatTimeout,
		signsOfLife:      make(chan struct{}, 1),
	}
}

// Write looks for the analytics widget's log lines in the engine output data. It never fails.
func (w *Watchdog) Write(data []byte) (int, error) {
	w.buffer = append(w.buffer, data...)

	remaining := w.buffer
	for index := bytes.IndexByte(remaining, '\n'); index >= 0; index = bytes.IndexByte(remaining, '\n') {
		w.checkLine(remaining[:index])
		remaining = remaining[index+1:]
	}
	if len(remaining) > maxUnfinishedLineLength {
		remaining = remaining[len(remaining)-maxUnfinishedLineLength:]
	}

	// Moved to the buffer's start, so it does not grow with the lines analyzed.
	w.buffer = append(w.buffer[:0], remaining...)
	return len(data), nil
}

// checkLine signals a sign of life if line is one of the analytics widget's log lines.
func (w *Watchdog) checkLine(line []byte) {
	if !bytes.Contains(line, []byte(widget.ActiveLogLine)) && !bytes.Contains(line, []byte(widget.HeartbeatLogLine)) {
		return
	}
	select {
	case w.signsOfLife <- struct{}{}:
	default:
	}
}

// Watch waits for the analytics widget's signs of life until ctx is done. If one does not come in time, it calls cancel
// with ErrAnalyticsWidgetNotActive or ErrAnalyticsWidgetHeartbeatMissing (wrapped) and returns.
func (w *Watchdog) Watch(ctx context.Context, cancel context.CancelCauseFunc) {
	timer := time.NewTimer(w.startTimeout)
	defer timer.Stop()

	cause := fmt.Errorf("%w within %s", ErrAnalyticsWidgetNotActive, w.startTimeout)
	for {
		select {
		case <-ctx.Done():
			return
		case <-w.signsOfLife:
			timer.Reset(w.heartbeatTimeout)
			cause = fmt.Errorf("%w for %s", ErrAnalyticsWidgetHeartbeatMissing, w.heartbeatTimeout)
		case <-timer.C:
			cancel(cause)
			return
		}
	}
}
