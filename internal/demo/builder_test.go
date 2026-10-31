package demo_test

import (
	"bytes"
	"compress/gzip"
	"encoding/binary"

	. "github.com/onsi/gomega"

	"github.com/backbone81/bar-replay-analytics-go/internal/demo"
)

// testDemo holds the content of a demo file to encode for tests. Encode writes the header as it is, so a test can
// make the header disagree with the content after calling updateSizes.
type testDemo struct {
	header           demo.DemoFileHeader
	headerExtension  []byte
	setupScript      string
	demoStream       []byte
	winningAllyTeams []uint8
	playerStats      []demo.PlayerStatistics
	teamStats        [][]demo.TeamStatistics
	trailingData     []byte

	// playerStatsByteOrder is the byte order encode writes playerStats in, or nil for demo.ByteOrder. Engines before
	// demo.PlayerStatsByteOrderFixedVersion wrote them in big-endian on some Linux hosts.
	playerStatsByteOrder binary.ByteOrder
}

// newTestDemo returns a valid demo of a 1v1 with one player, two teams and two statistics entries per team.
func newTestDemo() *testDemo {
	testDemo := &testDemo{
		header: demo.DemoFileHeader{
			Version:            demo.DemofileVersion,
			GameID:             [16]byte{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16},
			UnixTime:           1792849701,
			GameTime:           1607,
			WallclockTime:      1759,
			PlayerStatElemSize: int32(binary.Size(demo.PlayerStatistics{})),
			TeamStatElemSize:   int32(binary.Size(demo.TeamStatistics{})),
			TeamStatPeriod:     15,
		},
		setupScript:      "[game]\n{\n\tmapname=Argent Strata 1.1;\n}\n",
		demoStream:       []byte{0, 0, 0, 0, 3, 0, 0, 0, 1, 2, 3},
		winningAllyTeams: []uint8{1},
		playerStats: []demo.PlayerStatistics{
			{
				TeamControllerStatistics: demo.TeamControllerStatistics{NumCommands: 364, UnitCommands: 864},
				MousePixels:              504549,
				MouseClicks:              652,
				KeyPresses:               110,
			},
		},
		teamStats: [][]demo.TeamStatistics{
			{
				{Frame: 0},
				{Frame: 450, MetalUsed: 120.5, MetalProduced: 130.25, UnitsProduced: 3, UnitsKilled: 1},
			},
			{
				{Frame: 0},
				{Frame: 450, EnergyUsed: 900, EnergyProduced: 1000, UnitsProduced: 2, UnitsDied: 1},
			},
		},
	}
	copy(testDemo.header.Magic[:], demo.DemofileMagic)
	copy(testDemo.header.VersionString[:], "2026.07.04")
	testDemo.updateSizes()
	return testDemo
}

// updateSizes sets the header's sizes and counts to match the content.
func (d *testDemo) updateSizes() {
	d.header.HeaderSize = int32(binary.Size(demo.DemoFileHeader{}) + len(d.headerExtension))
	d.header.ScriptSize = int32(len(d.setupScript))
	d.header.DemoStreamSize = int32(len(d.demoStream))
	d.header.WinningAllyTeamsSize = int32(len(d.winningAllyTeams))
	d.header.NumPlayers = int32(len(d.playerStats))
	d.header.PlayerStatSize = d.header.NumPlayers * d.header.PlayerStatElemSize
	d.header.NumTeams = int32(len(d.teamStats))
	d.header.TeamStatSize = int32(len(d.teamStats) * 4)
	for _, history := range d.teamStats {
		d.header.TeamStatSize += int32(len(history)) * d.header.TeamStatElemSize
	}
}

// encode returns the decompressed demo file.
func (d *testDemo) encode() ([]byte, error) {
	var buffer bytes.Buffer
	if err := binary.Write(&buffer, demo.ByteOrder, d.header); err != nil {
		return nil, err
	}
	buffer.Write(d.headerExtension)
	buffer.WriteString(d.setupScript)
	buffer.Write(d.demoStream)
	buffer.Write(d.winningAllyTeams)
	playerStatsByteOrder := d.playerStatsByteOrder
	if playerStatsByteOrder == nil {
		playerStatsByteOrder = demo.ByteOrder
	}
	if err := binary.Write(&buffer, playerStatsByteOrder, d.playerStats); err != nil {
		return nil, err
	}
	for _, history := range d.teamStats {
		if err := binary.Write(&buffer, demo.ByteOrder, uint32(len(history))); err != nil {
			return nil, err
		}
	}
	for _, history := range d.teamStats {
		if err := binary.Write(&buffer, demo.ByteOrder, history); err != nil {
			return nil, err
		}
	}
	buffer.Write(d.trailingData)
	return buffer.Bytes(), nil
}

// compressed returns the demo file as the engine writes it.
func (d *testDemo) compressed() ([]byte, error) {
	data, err := d.encode()
	if err != nil {
		return nil, err
	}
	return compress(data)
}

// read encodes the demo, compresses it and reads it back with demo.Read. It fails the spec if the demo cannot be
// encoded, so the error it returns is always one of demo.Read.
func (d *testDemo) read() (*demo.Demo, error) {
	compressed, err := d.compressed()
	Expect(err).NotTo(HaveOccurred())
	return demo.Read(bytes.NewReader(compressed))
}

// setEngineVersion sets the engine version in the header.
func (d *testDemo) setEngineVersion(version string) {
	d.header.VersionString = [256]byte{}
	copy(d.header.VersionString[:], version)
}

// compress gzips data.
func compress(data []byte) ([]byte, error) {
	var buffer bytes.Buffer
	gzipWriter := gzip.NewWriter(&buffer)
	if _, err := gzipWriter.Write(data); err != nil {
		return nil, err
	}
	if err := gzipWriter.Close(); err != nil {
		return nil, err
	}
	return buffer.Bytes(), nil
}

// encodeValue returns value as the engine writes it.
func encodeValue(value any) ([]byte, error) {
	var buffer bytes.Buffer
	if err := binary.Write(&buffer, demo.ByteOrder, value); err != nil {
		return nil, err
	}
	return buffer.Bytes(), nil
}
