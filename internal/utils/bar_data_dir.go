package utils

import (
	"bufio"
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"strings"
)

// barDataDirMarker is the folder the BAR launcher creates in its write dir, the BAR data dir, on every start
// (spring-launcher src/spring_log.js).
const barDataDirMarker = "launcher-logs"

// barTitle is the BAR launcher's title (BYAR-Chobby dist_cfg/config.json), the name of its write dir on Linux.
const barTitle = "Beyond All Reason"

// BarDataDir returns the first of BarDataDirCandidates that is a BAR data dir (see IsBarDataDir).
func BarDataDir() (string, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}

	for _, candidate := range BarDataDirCandidates(runtime.GOOS, homeDir) {
		if IsBarDataDir(candidate) {
			return candidate, nil
		}
	}
	return "", errors.New("found no BAR data dir")
}

// IsBarDataDir reports whether dir holds barDataDirMarker.
func IsBarDataDir(dir string) bool {
	info, err := os.Stat(filepath.Join(dir, barDataDirMarker))
	return err == nil && info.IsDir()
}

// BarDataDirCandidates returns the BAR launcher's write dirs on the operating system goos (a runtime.GOOS value), in
// the order BarDataDir checks them (spring-launcher src/write_path.js, flathub info.beyondallreason.bar run.sh):
//
//   - windows: %LOCALAPPDATA%\Programs\Beyond-All-Reason\data, the installer's default folder
//   - linux: ~/.var/app/info.beyondallreason.bar/data (Flatpak), then for the AppImage <documents dir>/Beyond All
//     Reason (see documentsDir) and $XDG_STATE_HOME/Beyond All Reason ($XDG_STATE_HOME defaults to ~/.local/state)
//
// homeDir is the user's home folder. Other systems have none.
func BarDataDirCandidates(goos string, homeDir string) []string {
	switch goos {
	case "windows":
		localAppData := os.Getenv("LOCALAPPDATA")
		if localAppData == "" {
			return nil
		}
		return []string{filepath.Join(localAppData, "Programs", "Beyond-All-Reason", "data")}
	case "linux":
		stateHome := os.Getenv("XDG_STATE_HOME")
		if stateHome == "" {
			stateHome = filepath.Join(homeDir, ".local", "state")
		}
		return []string{
			filepath.Join(homeDir, ".var", "app", "info.beyondallreason.bar", "data"),
			filepath.Join(documentsDir(homeDir), barTitle),
			filepath.Join(stateHome, barTitle),
		}
	default:
		return nil
	}
}

// documentsDir returns the user's documents folder: XDG_DOCUMENTS_DIR from $XDG_CONFIG_HOME/user-dirs.dirs
// ($XDG_CONFIG_HOME defaults to ~/.config), or ~/Documents.
func documentsDir(homeDir string) string {
	configHome := os.Getenv("XDG_CONFIG_HOME")
	if configHome == "" {
		configHome = filepath.Join(homeDir, ".config")
	}

	//nolint:gosec
	file, err := os.Open(filepath.Join(configHome, "user-dirs.dirs"))
	if err != nil {
		return filepath.Join(homeDir, "Documents")
	}
	defer file.Close() //nolint:errcheck

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		value, found := strings.CutPrefix(strings.TrimSpace(scanner.Text()), "XDG_DOCUMENTS_DIR=")
		if !found {
			continue
		}
		value = strings.Trim(value, `"`)
		if rest, found := strings.CutPrefix(value, "$HOME"); found {
			return filepath.Join(homeDir, rest)
		}
		return value
	}
	return filepath.Join(homeDir, "Documents")
}
