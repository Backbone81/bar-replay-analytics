package demo

import (
	"bytes"
	"compress/gzip"
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"unicode/utf8"
)

// Demo is a demo file's content in file order, without the demo stream. Fields are named after the members of
// CDemoReader (rts/System/LoadSave/DemoReader.h).
//
// The statistics fields are nil if the engine crashed while recording (DemoStreamSize 0).
type Demo struct {
	// FileHeader holds the format version, engine version, game duration and section sizes.
	FileHeader DemoFileHeader `json:"fileHeader"`

	// SetupScript is the start script: map, game, players, teams, allyteams and AIs.
	SetupScript string `json:"setupScript"`

	// WinningAllyTeams are the allyteams as the game's Lua passed them to Spring.GameOver, unchecked: one may appear
	// twice or be Gaia's, and the header has no allyteam count to check them against.
	WinningAllyTeams []uint8 `json:"winningAllyTeams"`

	// PlayerStats are indexed by player ID, spectators included.
	PlayerStats []PlayerStatistics `json:"playerStats"`

	// PlayerStatsByteSwapped is true if the demo held PlayerStats big-endian, as engines before
	// PlayerStatsByteOrderFixedVersion wrote them on some Linux hosts. PlayerStats holds the corrected values.
	PlayerStatsByteSwapped bool `json:"playerStatsByteSwapped"`

	// TeamStats are each team's history in frame order, indexed by team ID, Gaia excluded.
	TeamStats [][]TeamStatistics `json:"teamStats"`
}

// MarshalJSON writes WinningAllyTeams as an array of numbers. Go would write a []uint8 as a base64 string.
func (d Demo) MarshalJSON() ([]byte, error) {
	var winningAllyTeams []int
	if d.WinningAllyTeams != nil {
		winningAllyTeams = make([]int, len(d.WinningAllyTeams))
		for i, allyTeam := range d.WinningAllyTeams {
			winningAllyTeams[i] = int(allyTeam)
		}
	}
	return json.Marshal(struct {
		FileHeader             DemoFileHeader     `json:"fileHeader"`
		SetupScript            string             `json:"setupScript"`
		WinningAllyTeams       []int              `json:"winningAllyTeams"`
		PlayerStats            []PlayerStatistics `json:"playerStats"`
		PlayerStatsByteSwapped bool               `json:"playerStatsByteSwapped"`
		TeamStats              [][]TeamStatistics `json:"teamStats"`
	}{
		FileHeader:             d.FileHeader,
		SetupScript:            d.SetupScript,
		WinningAllyTeams:       winningAllyTeams,
		PlayerStats:            d.PlayerStats,
		PlayerStatsByteSwapped: d.PlayerStatsByteSwapped,
		TeamStats:              d.TeamStats,
	})
}

// Read decompresses, parses and validates a demo file. It holds the decompressed file in memory, at most
// MaxDecompressedSize bytes.
func Read(reader io.Reader) (*Demo, error) {
	gzipReader, err := gzip.NewReader(reader)
	if err != nil {
		return nil, fmt.Errorf("decompressing demo: %w", err)
	}
	defer gzipReader.Close() //nolint:errcheck

	return ReadUncompressed(gzipReader)
}

// ReadUncompressed is Read for an already decompressed demo file.
func ReadUncompressed(reader io.Reader) (*Demo, error) {
	// One byte over the limit tells a file of exactly MaxDecompressedSize bytes from a larger one.
	data, err := io.ReadAll(io.LimitReader(reader, MaxDecompressedSize+1))
	if err != nil {
		return nil, fmt.Errorf("reading demo: %w", err)
	}
	if int64(len(data)) > MaxDecompressedSize {
		return nil, fmt.Errorf("reading demo: larger than %d bytes", MaxDecompressedSize)
	}
	return parse(data)
}

// ReadFile reads the demo file at filePath with Read.
func ReadFile(filePath string) (*Demo, error) {
	file, err := os.Open(filePath) //nolint:gosec // Opening the file the caller names is the purpose of ReadFile.
	if err != nil {
		return nil, err
	}
	defer file.Close() //nolint:errcheck
	return Read(file)
}

// parse parses and validates a decompressed demo file.
//
//nolint:funlen // One section after the other, in file order, reads best in one function.
func parse(data []byte) (*Demo, error) {
	buffer := bytes.NewBuffer(data)

	fileHeader, err := ReadDemoFileHeader(data)
	if err != nil {
		return nil, err
	}
	_, err = nextSection(buffer, "DemoFileHeader", fileHeader.HeaderSize)
	if err != nil {
		return nil, err
	}

	setupScriptData, err := nextSection(buffer, "setup script", fileHeader.ScriptSize)
	if err != nil {
		return nil, err
	}
	setupScript, err := readSetupScript(setupScriptData)
	if err != nil {
		return nil, err
	}

	if fileHeader.DemoStreamSize == 0 {
		return &Demo{
			FileHeader:  fileHeader,
			SetupScript: setupScript,
		}, nil
	}
	_, err = nextSection(buffer, "demo stream", fileHeader.DemoStreamSize)
	if err != nil {
		return nil, err
	}

	winningAllyTeamsData, err := nextSection(buffer, "winning allyteams", fileHeader.WinningAllyTeamsSize)
	if err != nil {
		return nil, err
	}
	// Copied, so the Demo does not keep the whole decompressed file in memory.
	winningAllyTeams := make([]uint8, len(winningAllyTeamsData))
	copy(winningAllyTeams, winningAllyTeamsData)

	playerStatsData, err := nextSection(buffer, "player statistics", fileHeader.PlayerStatSize)
	if err != nil {
		return nil, err
	}
	playerStats, playerStatsByteSwapped, err := readPlayerStatistics(
		playerStatsData,
		int(fileHeader.NumPlayers),
		fileHeader.VersionStringAsString(),
	)
	if err != nil {
		return nil, err
	}

	teamStatsData, err := nextSection(buffer, "team statistics", fileHeader.TeamStatSize)
	if err != nil {
		return nil, err
	}
	teamStats, err := readTeamStatistics(teamStatsData, int(fileHeader.NumTeams))
	if err != nil {
		return nil, err
	}

	if buffer.Len() != 0 {
		return nil, fmt.Errorf("%d unexpected bytes after the team statistics", buffer.Len())
	}
	return &Demo{
		FileHeader:             fileHeader,
		SetupScript:            setupScript,
		WinningAllyTeams:       winningAllyTeams,
		PlayerStats:            playerStats,
		PlayerStatsByteSwapped: playerStatsByteSwapped,
		TeamStats:              teamStats,
	}, nil
}

// nextSection returns the next size bytes of buffer without copying. size must be >= 0 (Next panics otherwise), which
// the DemoFileHeader validation ensures.
func nextSection(buffer *bytes.Buffer, name string, size int32) ([]byte, error) {
	section := buffer.Next(int(size))
	if len(section) < int(size) {
		return nil, fmt.Errorf("%s: %d bytes, but only %d bytes left", name, size, len(section))
	}
	return section, nil
}

// readSetupScript rejects NUL bytes (the engine strips trailing ones before writing) and invalid UTF-8.
func readSetupScript(setupScriptData []byte) (string, error) {
	if i := bytes.IndexByte(setupScriptData, 0); i >= 0 {
		return "", fmt.Errorf("setup script: NUL byte at offset %d", i)
	}
	if !utf8.Valid(setupScriptData) {
		return "", errors.New("setup script: not valid UTF-8")
	}
	return string(setupScriptData), nil
}

// readPlayerStatistics reads numPlayers PlayerStatistics (the header validation made len(data) match) and reports
// whether they were big-endian.
//
// Engines before PlayerStatsByteOrderFixedVersion wrote them big-endian on some Linux hosts, such as the autohosts that
// record bar-db's replays: rts/System/Platform/byteorder.h compared __BYTE_ORDER with __BIG_ENDIAN without including
// <endian.h>, so both were 0 and PlayerStatistics::swab swapped the bytes (fixed in #3358). The rest of such a demo is
// little-endian. A swapped count is negative or implausibly large, so if little-endian fails validation for such an
// engine, all players are read again big-endian. If both fail, the error names both.
func readPlayerStatistics(
	playerStatsData []byte, numPlayers int, engineVersion string,
) ([]PlayerStatistics, bool, error) {
	stats, err := readAllPlayerStatistics(playerStatsData, numPlayers, ByteOrder)
	if err == nil {
		return stats, false, nil
	}
	if !versionBefore(engineVersion, PlayerStatsByteOrderFixedVersion) {
		return nil, false, err
	}
	swapped, swappedErr := readAllPlayerStatistics(playerStatsData, numPlayers, binary.BigEndian)
	if swappedErr != nil {
		return nil, false, fmt.Errorf(
			"%w; in big-endian, as engine %s may have written them: %w",
			err,
			engineVersion,
			swappedErr,
		)
	}
	return swapped, true, nil
}

// readAllPlayerStatistics reads numPlayers PlayerStatistics in byteOrder.
func readAllPlayerStatistics(
	playerStatsData []byte, numPlayers int, byteOrder binary.ByteOrder,
) ([]PlayerStatistics, error) {
	stats := make([]PlayerStatistics, numPlayers)
	for i := range stats {
		var err error
		stats[i], err = readPlayerStatisticsInOrder(playerStatsData[i*playerStatisticsSize:], byteOrder)
		if err != nil {
			return nil, fmt.Errorf("player %d: %w", i, err)
		}
	}
	return stats, nil
}

// readTeamStatistics reads the entry count of each team (uint32), then each team's history.
//
// This follows the writer, CDemoRecorder::WriteTeamStats. The engine's reader, CDemoReader::LoadStats, reads numTeams
// bytes of counts instead of numTeams uint32s, so it gets the team statistics wrong in every demo with a team.
func readTeamStatistics(teamStatsData []byte, numTeams int) ([][]TeamStatistics, error) {
	countsSize := numTeams * 4
	if len(teamStatsData) < countsSize {
		return nil, fmt.Errorf("team statistics: %d bytes, too small for %d team counts", len(teamStatsData), numTeams)
	}

	counts := make([]uint32, numTeams)
	if _, err := binary.Decode(teamStatsData, ByteOrder, counts); err != nil {
		return nil, fmt.Errorf("team statistics counts: %w", err)
	}

	var total int64
	for _, count := range counts {
		total += int64(count)
	}
	if want := int64(countsSize) + total*int64(teamStatisticsSize); want != int64(len(teamStatsData)) {
		return nil, fmt.Errorf("team statistics: teamStatSize is %d, but the team counts add up to %d bytes",
			len(teamStatsData), want)
	}

	stats := make([][]TeamStatistics, numTeams)
	offset := countsSize
	for team, count := range counts {
		history := make([]TeamStatistics, count)
		for i := range history {
			entry, err := ReadTeamStatistics(teamStatsData[offset:])
			if err != nil {
				return nil, fmt.Errorf("team %d entry %d: %w", team, i, err)
			}
			if i > 0 && entry.Frame < history[i-1].Frame {
				return nil, fmt.Errorf("team %d entry %d: frame %d is before the previous entry's frame %d", team, i,
					entry.Frame, history[i-1].Frame)
			}
			history[i] = entry
			offset += teamStatisticsSize
		}
		stats[team] = history
	}
	return stats, nil
}
