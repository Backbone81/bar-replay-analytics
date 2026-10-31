package demo

import "encoding/binary"

// ByteOrder is the byte order of all numbers in a demo file.
// See swab() in rts/System/LoadSave/demofile.h.
var ByteOrder binary.ByteOrder = binary.LittleEndian

// MaxPlayers is the maximum number of players in a demo file.
// See MAX_PLAYERS in rts/Sim/Misc/GlobalConstants.h.
const MaxPlayers = 251

// MaxTeams is the maximum number of teams in a demo file.
// See MAX_TEAMS in rts/Sim/Misc/GlobalConstants.h.
const MaxTeams = 255

// MaxDecompressedSize limits how many bytes Read and ReadUncompressed read, against gzip bombs.
var MaxDecompressedSize int64 = 1024 * 1024 * 1024

// DemofileMagic are the first 16 bytes of each demo file (including the terminating NUL).
// See DEMOFILE_MAGIC in rts/System/LoadSave/demofile.h.
const DemofileMagic = "spring demofile\x00"

// DemofileVersion is the major version of the demo file format.
// See DEMOFILE_VERSION in rts/System/LoadSave/demofile.h.
const DemofileVersion = 5

// demoFileHeaderSize is the size of DemoFileHeader.
var demoFileHeaderSize = binary.Size(DemoFileHeader{})

// playerStatisticsSize is sizeof(PlayerStatistics).
var playerStatisticsSize = binary.Size(PlayerStatistics{})

// teamStatisticsSize is the size of TeamStatistics.
var teamStatisticsSize = binary.Size(TeamStatistics{})

// PlayerStatsByteOrderFixedVersion is the first engine release which writes the player statistics in little-endian
// byte order on every host, see Demo.PlayerStatsByteSwapped. Recoil commit 8e77748d54 is in the releases 2026.09.01 and
// later.
var PlayerStatsByteOrderFixedVersion = [3]int{2026, 9, 1}
