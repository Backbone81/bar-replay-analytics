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

// ReadUntil tells the Read functions how far into a demo file to read. A demo file can only be read in order (it is
// gzip-compressed), so stopping early saves decompressing the rest. A partial read does not notice damage after the
// sections it reads.
type ReadUntil int

const (
	// ReadUntilEnd reads every section and rejects bytes after the team statistics.
	ReadUntilEnd ReadUntil = iota

	// ReadUntilHeader reads only the DemoFileHeader.
	ReadUntilHeader

	// ReadUntilSetupScript reads the DemoFileHeader and the setup script.
	ReadUntilSetupScript
)

// Read decompresses, parses and validates a demo file up to until. It streams the file: it holds one section at a time
// in memory, at most MaxSectionSize bytes, and skips the demo stream without keeping it.
func Read(reader io.Reader, until ReadUntil) (*Demo, error) {
	gzipReader, err := gzip.NewReader(reader)
	if err != nil {
		return nil, fmt.Errorf("decompressing demo: %w", err)
	}
	defer gzipReader.Close() //nolint:errcheck

	return ReadUncompressed(gzipReader, until)
}

// ReadFile reads the demo file at filePath with Read.
func ReadFile(filePath string, until ReadUntil) (*Demo, error) {
	file, err := os.Open(filePath) //nolint:gosec // Opening the file the caller names is the purpose of ReadFile.
	if err != nil {
		return nil, err
	}
	defer file.Close() //nolint:errcheck
	return Read(file, until)
}

// ReadUncompressed is Read for an already decompressed demo file.
// It reads, parses and validates a decompressed demo file from reader up to until. The fields of the sections it
// does not read are left at their zero values.
//
//nolint:funlen // One section after the other, in file order, reads best in one function.
func ReadUncompressed(reader io.Reader, until ReadUntil) (*Demo, error) {
	fileHeader, err := readFileHeader(reader)
	if err != nil {
		return nil, err
	}
	if until == ReadUntilHeader {
		return &Demo{
			FileHeader: fileHeader,
		}, nil
	}

	setupScriptData, err := readSection(reader, "setup script", int64(fileHeader.ScriptSize))
	if err != nil {
		return nil, err
	}
	setupScript, err := readSetupScript(setupScriptData)
	if err != nil {
		return nil, err
	}
	// A crashed demo's stream runs to the end of the file, so there is nothing after it to read.
	if until == ReadUntilSetupScript || fileHeader.DemoStreamSize == 0 {
		return &Demo{
			FileHeader:  fileHeader,
			SetupScript: setupScript,
		}, nil
	}

	if err := skipSection(reader, "demo stream", int64(fileHeader.DemoStreamSize)); err != nil {
		return nil, err
	}

	winningAllyTeams, err := readSection(reader, "winning allyteams", int64(fileHeader.WinningAllyTeamsSize))
	if err != nil {
		return nil, err
	}

	playerStatsData, err := readSection(reader, "player statistics", int64(fileHeader.PlayerStatSize))
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

	teamStatsData, err := readSection(reader, "team statistics", int64(fileHeader.TeamStatSize))
	if err != nil {
		return nil, err
	}
	teamStats, err := readTeamStatistics(teamStatsData, int(fileHeader.NumTeams))
	if err != nil {
		return nil, err
	}

	if err := expectEnd(reader); err != nil {
		return nil, err
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

// readFileHeader reads and validates the DemoFileHeader from reader and skips the fields a newer engine appended to
// it.
func readFileHeader(reader io.Reader) (DemoFileHeader, error) {
	data, err := readSection(reader, "DemoFileHeader", int64(demoFileHeaderSize))
	if err != nil {
		return DemoFileHeader{}, err
	}

	fileHeader, err := ReadDemoFileHeader(data)
	if err != nil {
		return DemoFileHeader{}, err
	}

	// The validation made HeaderSize >= demoFileHeaderSize.
	headerExtensionSize := int64(fileHeader.HeaderSize) - int64(demoFileHeaderSize)
	if err := skipSection(reader, "DemoFileHeader extension", headerExtensionSize); err != nil {
		return DemoFileHeader{}, err
	}
	return fileHeader, nil
}

// readSection reads the next section, of size bytes, from reader. size must be >= 0, which the DemoFileHeader
// validation ensures. A size above MaxSectionSize is rejected before anything is allocated, and the buffer grows only
// with the bytes actually read, so a forged size in a short file allocates no more than the file holds.
func readSection(reader io.Reader, name string, size int64) ([]byte, error) {
	if size > MaxSectionSize {
		return nil, fmt.Errorf("%s: %d bytes, more than the limit of %d bytes", name, size, MaxSectionSize)
	}

	data, err := io.ReadAll(io.LimitReader(reader, size))
	if err != nil {
		return nil, fmt.Errorf("%s: %w", name, err)
	}

	if int64(len(data)) < size {
		return nil, fmt.Errorf("%s: %d bytes, but only %d bytes left", name, size, len(data))
	}
	return data, nil
}

// skipSection skips the next section, of count bytes, from reader without keeping it. A count above MaxSectionSize is
// rejected.
func skipSection(reader io.Reader, name string, count int64) error {
	if count > MaxSectionSize {
		return fmt.Errorf("%s: %d bytes, more than the limit of %d bytes", name, count, MaxSectionSize)
	}

	skipped, err := io.CopyN(io.Discard, reader, count)
	if errors.Is(err, io.EOF) {
		return fmt.Errorf("%s: %d bytes, but only %d bytes left", name, count, skipped)
	}
	if err != nil {
		return fmt.Errorf("%s: %w", name, err)
	}
	return nil
}

// expectEnd rejects bytes after the team statistics. Reading up to the end also makes a gzip reader verify its
// checksum.
func expectEnd(reader io.Reader) error {
	var next [1]byte
	_, err := io.ReadFull(reader, next[:])
	if errors.Is(err, io.EOF) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("after the team statistics: %w", err)
	}
	return errors.New("unexpected bytes after the team statistics")
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
