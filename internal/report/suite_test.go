package report_test

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/backbone81/bar-replay-analytics-go/internal/demo"
	"github.com/backbone81/bar-replay-analytics-go/internal/tdf"
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

// testSetupScript is the setup script of the demo of newTestDemo: the user against BARb, as BAR's local lobby writes
// it. The allyteams, teams and AIs are numbered with gaps, which the engine closes: allyteam3 is allyteam 1, team2 is
// team 1.
const testSetupScript = `
	[game]
	{
		gametype=Beyond All Reason test-31756-0d4c3c3;
		mapname=Argent Strata 1.1;
		startpostype=2;
		[modoptions] { date_day=29; tweakdefs=; }
		[allyteam0] { startrectleft=0.12; startrecttop=0.72; startrectright=0.26; startrectbottom=0.885; }
		[allyteam3] { startrectleft=0.74; startrecttop=0.115; startrectright=0.88; startrectbottom=0.28; }
		[player0] { name=Player; team=0; }
		[team0] { allyteam=0; teamleader=0; side=Cortex; handicap=0; }
		[team2] { allyteam=3; teamleader=0; side=Random; handicap=0; }
		[ai1]
		{
			name=BARbarianAI(1);
			shortname=BARb;
			host=0;
			team=2;
			[options] { profile=Hard; disabledunits=; }
		}
	}
`

// newTestSetupScript returns testSetupScript parsed.
func newTestSetupScript() *tdf.Section {
	return parseScript(testSetupScript)
}

// parseScript returns the setup script text parsed.
func parseScript(text string) *tdf.Section {
	GinkgoHelper()
	script, err := tdf.Parse(text)
	Expect(err).NotTo(HaveOccurred())
	return script
}
