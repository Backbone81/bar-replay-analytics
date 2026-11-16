package widget

// ActiveLogLine is the line the analytics widget logs (Spring.Echo, so it is in the engine's output) once it has
// started logging in a playback: it is loaded, in spring-headless, and its events file is open.
const ActiveLogLine = "Replay Analytics: active"

// HeartbeatLogLine starts the line the analytics widget logs once per minute of wall time while the playback runs
// (HEARTBEAT_SECONDS), followed by the frame.
const HeartbeatLogLine = "Replay Analytics: alive"
