package report_test

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/backbone81/bar-replay-analytics-go/internal/demo"
)

func TestReport(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Report Suite")
}

// newTestDemo returns a demo of a 1v1 of 60 s which allyteam 1 won. Each team has statistics at frames 0 and 1500,
// and its totals labeled, as the engine does, with frame 1950 of the next period, after the end of the game at frame
// 1800.
func newTestDemo() *demo.Demo {
	return &demo.Demo{
		FileHeader: demo.DemoFileHeader{
			Version:        demo.DemofileVersion,
			GameID:         [16]byte{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16},
			VersionString:  versionString("2026.07.04"),
			UnixTime:       1792849701,
			DemoStreamSize: 11,
			GameTime:       60,
			WallclockTime:  66,
			NumTeams:       2,
			TeamStatPeriod: 15,
		},
		WinningAllyTeams: []uint8{1},
		TeamStats: [][]demo.TeamStatistics{
			{
				{Frame: 0},
				{Frame: 1500, MetalProduced: 100, MetalUsed: 50, EnergyProduced: 1000, EnergyExcess: 200},
				{Frame: 1950, MetalProduced: 420, MetalUsed: 360, EnergyProduced: 2400, EnergyExcess: 240},
			},
			{
				{Frame: 0},
				{Frame: 1500, MetalProduced: 200, EnergyProduced: 2000, MetalSent: 10, EnergyReceived: 30},
				{Frame: 1950, MetalProduced: 500, EnergyProduced: 5000, MetalSent: 30, EnergyReceived: 90},
			},
		},
	}
}

// versionString returns version as the NUL-padded VersionString of a DemoFileHeader.
func versionString(version string) [256]byte {
	var result [256]byte
	copy(result[:], version)
	return result
}
