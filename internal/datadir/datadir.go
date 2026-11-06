// Package datadir finds the BAR data dir: the folder of a BAR install that holds demos/, engine/, maps/ and the game
// packages, where the BAR launcher downloads to and BAR saves its replays.
package datadir

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
)

// markers are the folders of which a well-known location must hold at least one to count as a BAR data dir: the
// launcher downloads the engine first, BAR saves its replays in demos/.
var markers = []string{"engine", "demos"}

// Candidates returns the well-known locations of the BAR data dir on the operating system goos (as runtime.GOOS
// names it), in the order Find checks them. homeDir is the user's home folder; a location whose environment variable
// is unset is left out. The locations are where the BAR installers put it:
//
//   - windows: %LOCALAPPDATA%\Programs\Beyond-All-Reason\data
//   - linux: the Flatpak's ~/.var/app/info.beyondallreason.bar/data, and the AppImage's
//     $XDG_STATE_HOME/Beyond All Reason ($XDG_STATE_HOME defaults to ~/.local/state)
//
// BAR has no installer for other systems, so they have none.
func Candidates(goos string, homeDir string) []string {
	var candidates []string
	switch goos {
	case "windows":
		if localAppData := os.Getenv("LOCALAPPDATA"); localAppData != "" {
			candidates = append(candidates, filepath.Join(localAppData, "Programs", "Beyond-All-Reason", "data"))
		}
	case "linux":
		if homeDir != "" {
			candidates = append(candidates, filepath.Join(homeDir, ".var", "app", "info.beyondallreason.bar", "data"))
		}
		stateHome := os.Getenv("XDG_STATE_HOME")
		if stateHome == "" && homeDir != "" {
			stateHome = filepath.Join(homeDir, ".local", "state")
		}
		if stateHome != "" {
			candidates = append(candidates, filepath.Join(stateHome, "Beyond All Reason"))
		}
	}
	return candidates
}

// IsDataDir reports whether dir is a folder that holds one of the folders a BAR data dir has (see markers).
func IsDataDir(dir string) bool {
	for _, marker := range markers {
		if info, err := os.Stat(filepath.Join(dir, marker)); err == nil && info.IsDir() {
			return true
		}
	}
	return false
}

// FindIn returns the first of candidates that is a BAR data dir (see IsDataDir). A user with several installs passes
// the one to use with --bar-data-dir. It returns an error if none is.
func FindIn(candidates []string) (string, error) {
	for _, candidate := range candidates {
		if IsDataDir(candidate) {
			return candidate, nil
		}
	}
	if len(candidates) == 0 {
		return "", errors.New("no BAR data dir is known on this system; pass it with --bar-data-dir")
	}
	return "", fmt.Errorf("found no BAR data dir in %s; pass it with --bar-data-dir", strings.Join(candidates, ", "))
}

// Find returns the BAR data dir of this system: the first of Candidates that is one (see FindIn).
func Find() (string, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		homeDir = ""
	}
	return FindIn(Candidates(runtime.GOOS, homeDir))
}
