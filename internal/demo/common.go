package demo

import "encoding/binary"

// ByteOrder is the byte order of all numbers in a demo file.
// See swab() in rts/System/LoadSave/demofile.h.
var ByteOrder binary.ByteOrder = binary.LittleEndian

// MaxPlayers is the maximum number of players.
// See MAX_PLAYERS in rts/Sim/Misc/GlobalConstants.h.
const MaxPlayers = 251

// MaxTeams is the maximum number of teams.
// See MAX_TEAMS in rts/Sim/Misc/GlobalConstants.h.
const MaxTeams = 255

// MaxSectionSize is the largest section of a demo file the Read functions accept, against gzip bombs and forged
// sizes. It is checked for each section before reading it, the skipped demo stream included.
var MaxSectionSize int64 = 1024 * 1024 * 1024

// DemofileMagic is the first 16 bytes of a demo file, terminating NUL included.
// See DEMOFILE_MAGIC in rts/System/LoadSave/demofile.h.
const DemofileMagic = "spring demofile\x00"

// DemofileVersion is the major version of the demo file format.
// See DEMOFILE_VERSION in rts/System/LoadSave/demofile.h.
const DemofileVersion = 5

// demoFileHeaderSize is sizeof(DemoFileHeader).
var demoFileHeaderSize = binary.Size(DemoFileHeader{})

// playerStatisticsSize is sizeof(PlayerStatistics).
var playerStatisticsSize = binary.Size(PlayerStatistics{})

// teamStatisticsSize is sizeof(TeamStatistics).
var teamStatisticsSize = binary.Size(TeamStatistics{})

// PlayerStatsByteOrderFixedVersion is the first engine release (2026.09.01, Recoil commit 8e77748d54) that writes the
// player statistics little-endian on every host, see Demo.PlayerStatsByteSwapped.
var PlayerStatsByteOrderFixedVersion = [3]int{2026, 9, 1}
