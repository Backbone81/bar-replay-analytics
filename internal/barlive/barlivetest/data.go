package barlivetest

import (
	"bytes"
	"compress/gzip"
	"encoding/binary"
	"encoding/hex"
	"fmt"

	"github.com/backbone81/bar-replay-analytics-go/internal/barlive"
	"github.com/backbone81/bar-replay-analytics-go/internal/demo"
)

// DemoFile returns a gzip-compressed demo file of the game gameID (hex, as barlive.Replay.ID) with a valid header
// followed by filler, which is all barlive.Client.DownloadDemoIntoDir and demo.ReadDemoFileHeaderFromFilePath read.
func DemoFile(gameID string) ([]byte, error) {
	header := demo.DemoFileHeader{
		Version:            demo.DemofileVersion,
		HeaderSize:         int32(binary.Size(demo.DemoFileHeader{})), //nolint:gosec // This does not overflow
		ScriptSize:         1,
		PlayerStatElemSize: int32(binary.Size(demo.PlayerStatistics{})), //nolint:gosec // This does not overflow
		TeamStatElemSize:   int32(binary.Size(demo.TeamStatistics{})),   //nolint:gosec // This does not overflow
		TeamStatPeriod:     15,                                          //nolint:mnd // the engine's period
	}
	copy(header.Magic[:], demo.DemofileMagic)
	copy(header.VersionString[:], "2026.07.04")
	id, err := hex.DecodeString(gameID)
	if err != nil {
		return nil, fmt.Errorf("game ID %q: %w", gameID, err)
	}
	if len(id) != len(header.GameID) {
		return nil, fmt.Errorf("game ID %q: got %d bytes, want %d", gameID, len(id), len(header.GameID))
	}
	copy(header.GameID[:], id)

	var buffer bytes.Buffer
	gzipWriter := gzip.NewWriter(&buffer)
	if err := binary.Write(gzipWriter, demo.ByteOrder, header); err != nil {
		return nil, err
	}
	if _, err := gzipWriter.Write([]byte("rest of the demo")); err != nil {
		return nil, err
	}
	if err := gzipWriter.Close(); err != nil {
		return nil, err
	}
	return buffer.Bytes(), nil
}

// ReplaySummary returns a replay with ID id as a bar-db search returns it, whose allyteams have the given numbers of
// players and no AIs.
func ReplaySummary(id string, allyTeamSizes ...int) barlive.ReplaySummary {
	replay := barlive.ReplaySummary{ID: id}
	for _, size := range allyTeamSizes {
		allyTeam := barlive.AllyTeam{Players: []barlive.Player{}, AIs: []barlive.AI{}}
		for i := range size {
			allyTeam.Players = append(allyTeam.Players, barlive.Player{Name: fmt.Sprintf("Player%d", i+1)})
		}
		replay.AllyTeams = append(replay.AllyTeams, allyTeam)
	}
	return replay
}
