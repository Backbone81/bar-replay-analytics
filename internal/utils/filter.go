package utils

import (
	"strings"
)

// Filter selects a replay which matches the filter. A replay matches if it matches every field that is not empty,
// and it matches a field if it matches one of the field's values. The zero Filter matches every replay.
type Filter struct {
	// MapNames match a replay whose map name contains one of them, ignoring case, so "argent strata" matches every
	// version of Argent Strata.
	MapNames []string

	// PlayerNames match a replay in which a player or an AI has one of these names, ignoring case. Spectators do not
	// count (see ReplayInfo.PlayerNames).
	PlayerNames []string
}

// Matches returns whether the replay replayInfo describes matches the filter.
func (f Filter) Matches(replayInfo *ReplayInfo) bool {
	return f.matchesMapName(replayInfo.MapName) && f.matchesPlayerNames(replayInfo.PlayerNames)
}

// matchesMapName returns whether mapName matches MapNames.
func (f Filter) matchesMapName(mapName string) bool {
	if len(f.MapNames) == 0 {
		return true
	}
	mapName = strings.ToLower(mapName)
	for _, wanted := range f.MapNames {
		if strings.Contains(mapName, strings.ToLower(wanted)) {
			return true
		}
	}
	return false
}

// matchesPlayerNames returns whether one of playerNames matches PlayerNames.
func (f Filter) matchesPlayerNames(playerNames []string) bool {
	if len(f.PlayerNames) == 0 {
		return true
	}
	for _, wanted := range f.PlayerNames {
		for _, playerName := range playerNames {
			if strings.EqualFold(playerName, wanted) {
				return true
			}
		}
	}
	return false
}
