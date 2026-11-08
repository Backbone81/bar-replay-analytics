package cmd

// ReplayPaths exposes replayPaths to the tests.
var ReplayPaths = replayPaths

// SetBarDataDir sets the --bar-data-dir of the commands for a test and returns a function that restores it.
func SetBarDataDir(dir string) func() {
	previous := barDataDir
	barDataDir = dir
	return func() { barDataDir = previous }
}
