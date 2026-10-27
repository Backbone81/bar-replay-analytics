package demo

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"math"
	"unicode"
	"unicode/utf8"
)

// DemoFileHeader is the header at the start of every demo file. It holds the sizes of all following sections.
//
// See rts/System/LoadSave/demofile.h.
type DemoFileHeader struct {
	// Magic is DemofileMagic.
	Magic [16]byte

	// Version is DemofileVersion, the major version of the file format.
	Version int32

	// HeaderSize is the size of the DemoFileHeader, the minor version of the file format.
	HeaderSize int32

	// VersionString is the engine version, NUL-terminated, e.g. "2026.07.04".
	VersionString [256]byte

	// GameID is the unique game identifier. Identical for each player of the game.
	GameID [16]byte

	// UnixTime is the Unix time when the game was started.
	UnixTime uint64

	// ScriptSize is the size of the start script.
	ScriptSize int32

	// DemoStreamSize is the size of the demo stream; 0 if the engine crashed while recording.
	DemoStreamSize int32

	// GameTime is the total number of seconds game time.
	GameTime int32

	// WallclockTime is the total number of seconds wallclock time.
	WallclockTime int32

	// NumPlayers is the number of players for which stats are saved, spectators included.
	NumPlayers int32

	// PlayerStatSize is the size of the entire player statistics chunk.
	PlayerStatSize int32

	// PlayerStatElemSize is sizeof(PlayerStatistics).
	PlayerStatElemSize int32

	// NumTeams is the number of teams for which stats are saved.
	NumTeams int32

	// TeamStatSize is the size of the entire team statistics chunk.
	TeamStatSize int32

	// TeamStatElemSize is sizeof(TeamStatistics).
	TeamStatElemSize int32

	// TeamStatPeriod is the interval (in seconds) between team stats.
	TeamStatPeriod int32

	// WinningAllyTeamsSize is the size of the list of winning allyteams.
	WinningAllyTeamsSize int32
}

// ReadDemoFileHeader reads and validates a DemoFileHeader from the start of data.
func ReadDemoFileHeader(data []byte) (DemoFileHeader, error) {
	var header DemoFileHeader
	if _, err := binary.Decode(data, ByteOrder, &header); err != nil {
		return DemoFileHeader{}, fmt.Errorf("reading DemoFileHeader: %w", err)
	}
	if err := header.validate(); err != nil {
		return DemoFileHeader{}, fmt.Errorf("invalid DemoFileHeader: %w", err)
	}
	return header, nil
}

// VersionStringAsString returns VersionString up to its terminating NUL.
func (h *DemoFileHeader) VersionStringAsString() string {
	return string(h.VersionString[:bytes.IndexByte(h.VersionString[:], 0)])
}

// validate checks every field on its own. Whether the sizes fit the file is checked while reading the sections.
func (h *DemoFileHeader) validate() error {
	if string(h.Magic[:]) != DemofileMagic {
		return fmt.Errorf("magic: got %q, want %q", h.Magic[:], DemofileMagic)
	}

	if h.Version != DemofileVersion {
		return fmt.Errorf("version: got %d, want %d", h.Version, DemofileVersion)
	}

	// A newer engine may append fields to the header (headerSize is the minor version). parse skips them.
	if int(h.HeaderSize) < demoFileHeaderSize {
		return fmt.Errorf("headerSize: got %d, want >= %d", h.HeaderSize, demoFileHeaderSize)
	}

	if err := validateVersionString(h.VersionString[:]); err != nil {
		return fmt.Errorf("versionString: %w", err)
	}

	// gameID: any 16 bytes are valid.

	if h.UnixTime > math.MaxInt64 {
		return fmt.Errorf("unixTime: %d does not fit a signed 64-bit Unix time", h.UnixTime)
	}

	if err := h.validateGame(); err != nil {
		return err
	}

	return h.validateStatistics()
}

// validateGame checks the fields about the start script, the demo stream and the game's duration.
func (h *DemoFileHeader) validateGame() error {
	if h.ScriptSize <= 0 {
		return fmt.Errorf("scriptSize: got %d, want > 0", h.ScriptSize)
	}

	if h.DemoStreamSize < 0 {
		return fmt.Errorf("demoStreamSize: got %d, want >= 0", h.DemoStreamSize)
	}

	if h.GameTime < 0 {
		return fmt.Errorf("gameTime: got %d, want >= 0", h.GameTime)
	}

	if h.WallclockTime < 0 {
		return fmt.Errorf("wallclockTime: got %d, want >= 0", h.WallclockTime)
	}
	return nil
}

// validateStatistics checks the fields about the winning allyteams and the player and team statistics.
func (h *DemoFileHeader) validateStatistics() error {
	if h.NumPlayers < 0 || h.NumPlayers > MaxPlayers {
		return fmt.Errorf("numPlayers: got %d, want 0 to %d", h.NumPlayers, MaxPlayers)
	}

	if int(h.PlayerStatElemSize) != playerStatisticsSize {
		return fmt.Errorf("playerStatElemSize: got %d, want %d", h.PlayerStatElemSize, playerStatisticsSize)
	}

	if h.PlayerStatSize != h.NumPlayers*h.PlayerStatElemSize {
		return fmt.Errorf("playerStatSize: got %d, want numPlayers %d * playerStatElemSize %d",
			h.PlayerStatSize, h.NumPlayers, h.PlayerStatElemSize)
	}

	if h.NumTeams < 0 || h.NumTeams > MaxTeams {
		return fmt.Errorf("numTeams: got %d, want 0 to %d", h.NumTeams, MaxTeams)
	}

	if h.TeamStatSize < 0 {
		return fmt.Errorf("teamStatSize: got %d, want >= 0", h.TeamStatSize)
	}

	if int(h.TeamStatElemSize) != teamStatisticsSize {
		return fmt.Errorf("teamStatElemSize: got %d, want %d", h.TeamStatElemSize, teamStatisticsSize)
	}

	if h.TeamStatPeriod != StatsPeriod {
		return fmt.Errorf("teamStatPeriod: got %d, want %d", h.TeamStatPeriod, StatsPeriod)
	}

	if h.WinningAllyTeamsSize < 0 || h.WinningAllyTeamsSize > MaxTeams {
		return fmt.Errorf("winningAllyTeamsSize: got %d, want 0 to %d", h.WinningAllyTeamsSize, MaxTeams)
	}
	return nil
}

// validateVersionString checks for a non-empty, NUL-terminated string of printable characters.
func validateVersionString(versionString []byte) error {
	end := bytes.IndexByte(versionString, 0)
	if end < 0 {
		return errors.New("not NUL-terminated")
	}
	if end == 0 {
		return errors.New("empty")
	}

	version := versionString[:end]
	if !utf8.Valid(version) {
		return errors.New("not valid UTF-8")
	}

	for _, r := range string(version) {
		if !unicode.IsPrint(r) {
			return fmt.Errorf("contains non-printable character %q", r)
		}
	}
	return nil
}
