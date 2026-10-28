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

// Demo is a demo file's content in the order it appears in the file. The demo stream (the recorded network messages)
// is skipped.
//
// The fields are named after the members of CDemoReader in the Recoil engine source: rts/System/LoadSave/DemoReader.h.
type Demo struct {
	// FileHeader provides metadata about this file.
	FileHeader DemoFileHeader `json:"fileHeader"`

	// SetupScript provides the setup script: map, game, players, teams, allyteams and AIs.
	SetupScript string `json:"setupScript"`

	// WinningAllyTeams provides the allyteams which have won, as the game's Lua passed them to Spring.GameOver: an
	// allyteam may be listed twice, and Gaia's allyteam is accepted as well. The header holds no allyteam count to
	// check them against.
	// Is nil if the engine crashed while recording the demo.
	WinningAllyTeams []uint8 `json:"winningAllyTeams"`

	// PlayerStats provides statistics per player.
	// Is nil if the engine crashed while recording the demo.
	PlayerStats []PlayerStatistics `json:"playerStats"`

	// TeamStats provides statistics per team.
	// Is nil if the engine crashed while recording the demo.
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
		FileHeader       DemoFileHeader     `json:"fileHeader"`
		SetupScript      string             `json:"setupScript"`
		WinningAllyTeams []int              `json:"winningAllyTeams"`
		PlayerStats      []PlayerStatistics `json:"playerStats"`
		TeamStats        [][]TeamStatistics `json:"teamStats"`
	}{
		FileHeader:       d.FileHeader,
		SetupScript:      d.SetupScript,
		WinningAllyTeams: winningAllyTeams,
		PlayerStats:      d.PlayerStats,
		TeamStats:        d.TeamStats,
	})
}

// Read decompresses a demo file from reader into memory, parses and validates it.
// Returns an error if validation fails.
func Read(reader io.Reader) (*Demo, error) {
	gzipReader, err := gzip.NewReader(reader)
	if err != nil {
		return nil, fmt.Errorf("decompressing demo: %w", err)
	}
	defer gzipReader.Close() //nolint:errcheck

	return ReadUncompressed(gzipReader)
}

// ReadUncompressed reads an already decompressed demo file from reader into memory, parses and validates it.
// Returns an error if validation fails.
func ReadUncompressed(reader io.Reader) (*Demo, error) {
	// We read one byte more than the max, so we can detect if there would have been more than the max number of bytes.
	data, err := io.ReadAll(io.LimitReader(reader, MaxDecompressedSize+1))
	if err != nil {
		return nil, fmt.Errorf("reading demo: %w", err)
	}
	if int64(len(data)) > MaxDecompressedSize {
		return nil, fmt.Errorf("reading demo: larger than %d bytes", MaxDecompressedSize)
	}
	return parse(data)
}

// ReadFile opens the demo file named by filePath and reads it with Read.
// Returns an error if the file cannot be opened or validation fails.
func ReadFile(filePath string) (*Demo, error) {
	file, err := os.Open(filePath) //nolint:gosec // Opening the file the caller names is the purpose of ReadFile.
	if err != nil {
		return nil, err
	}
	defer file.Close() //nolint:errcheck
	return Read(file)
}

// parse parses a decompressed demo file.
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
	playerStats, err := readPlayerStatistics(playerStatsData, int(fileHeader.NumPlayers))
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
		FileHeader:       fileHeader,
		SetupScript:      setupScript,
		WinningAllyTeams: winningAllyTeams,
		PlayerStats:      playerStats,
		TeamStats:        teamStats,
	}, nil
}

// nextSection returns the next size bytes of buffer, without copying. Section sizes come from the DemoFileHeader,
// whose validation ensures size >= 0: Next panics on a negative size.
func nextSection(buffer *bytes.Buffer, name string, size int32) ([]byte, error) {
	section := buffer.Next(int(size))
	if len(section) < int(size) {
		return nil, fmt.Errorf("%s: %d bytes, but only %d bytes left", name, size, len(section))
	}
	return section, nil
}

// readSetupScript validates the setup script. The engine strips trailing NULs before writing it.
func readSetupScript(setupScriptData []byte) (string, error) {
	if i := bytes.IndexByte(setupScriptData, 0); i >= 0 {
		return "", fmt.Errorf("setup script: NUL byte at offset %d", i)
	}
	if !utf8.Valid(setupScriptData) {
		return "", errors.New("setup script: not valid UTF-8")
	}
	return string(setupScriptData), nil
}

// readPlayerStatistics reads numPlayers PlayerStatistics. The header validation made len(data) match numPlayers.
func readPlayerStatistics(playerStatsData []byte, numPlayers int) ([]PlayerStatistics, error) {
	stats := make([]PlayerStatistics, numPlayers)
	for i := range stats {
		var err error
		if stats[i], err = ReadPlayerStatistics(playerStatsData[i*playerStatisticsSize:]); err != nil {
			return nil, fmt.Errorf("player %d: %w", i, err)
		}
	}
	return stats, nil
}

// readTeamStatistics reads the number of TeamStatistics per team (one uint32 each), then each team's history.
//
// This follows the writer, CDemoRecorder::WriteTeamStats. The engine's reader, CDemoReader::LoadStats, reads only
// numTeams bytes of counts instead of numTeams uint32s, so its team statistics are wrong in every demo with a team.
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
