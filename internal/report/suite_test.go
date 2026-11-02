package report_test

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/backbone81/bar-replay-analytics-go/internal/demo"
	"github.com/backbone81/bar-replay-analytics-go/internal/events"
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

// newTestEventsFile returns an events file of the game of newTestDemo, in which player 0 starts with a Cortex commander
// and player 1, Random in the setup script, with an Armada one. Each builds a metal extractor, player 0 also a bot lab,
// and after the opening build order of 5 minutes (frame 9000) a construction bot. Player 0 has enough resources for its
// metal extractor and its construction bot (build rate 1) and gets help with its bot lab (2), player 1 lacks resources
// for its metal extractor (0.83). The ID of the metal extractor of player 0 is reused after it died, for a commander
// given by a gadget.
func newTestEventsFile() *events.File {
	return &events.File{
		Events: []events.Event{
			{Frame: 0, UnitCreated: &events.UnitCreated{UnitID: 100, UnitDefID: 1, UnitTeam: 0}},
			{Frame: 0, UnitFinished: &events.UnitFinished{UnitID: 100, UnitDefID: 1, UnitTeam: 0}},
			{Frame: 0, UnitCreated: &events.UnitCreated{UnitID: 200, UnitDefID: 5, UnitTeam: 1}},
			{Frame: 0, UnitFinished: &events.UnitFinished{UnitID: 200, UnitDefID: 5, UnitTeam: 1}},
			{Frame: 30, UnitCreated: &events.UnitCreated{UnitID: 101, UnitDefID: 2, UnitTeam: 0, BuilderID: new(100)}},
			{Frame: 60, UnitCreated: &events.UnitCreated{UnitID: 201, UnitDefID: 6, UnitTeam: 1, BuilderID: new(200)}},
			{Frame: 600, UnitFinished: &events.UnitFinished{UnitID: 101, UnitDefID: 2, UnitTeam: 0}},
			{Frame: 750, UnitFinished: &events.UnitFinished{UnitID: 201, UnitDefID: 6, UnitTeam: 1}},
			{Frame: 900, UnitCreated: &events.UnitCreated{UnitID: 102, UnitDefID: 3, UnitTeam: 0, BuilderID: new(100)}},
			{Frame: 2700, UnitFinished: &events.UnitFinished{UnitID: 102, UnitDefID: 3, UnitTeam: 0}},
			{Frame: 3000, UnitDestroyed: &events.UnitDestroyed{UnitID: 101, UnitDefID: 2, UnitTeam: 0, BuildProgress: 1}},
			{Frame: 3300, UnitCreated: &events.UnitCreated{UnitID: 101, UnitDefID: 1, UnitTeam: 0}},
			{Frame: 3300, UnitFinished: &events.UnitFinished{UnitID: 101, UnitDefID: 1, UnitTeam: 0}},
			{Frame: 8400, UnitCreated: &events.UnitCreated{UnitID: 103, UnitDefID: 4, UnitTeam: 0, BuilderID: new(102)}},
			{Frame: 9300, UnitFinished: &events.UnitFinished{UnitID: 103, UnitDefID: 4, UnitTeam: 0}},
		},
		Game: events.Game{
			MapName:     "Argent Strata 1.1",
			GameName:    "Beyond All Reason",
			GameVersion: "test-31756-0d4c3c3",
		},
		Sides: []events.Side{
			{SideName: "armada", CaseName: "Armada", StartUnit: "armcom"},
			{SideName: "cortex", CaseName: "Cortex", StartUnit: "corcom"},
			{SideName: "random", CaseName: "Random", StartUnit: "dummycom"},
			{SideName: "legion", CaseName: "Legion", StartUnit: "legcom"},
		},
		Teams: []events.Team{
			{TeamID: 0, AllyTeam: 0, StartUnit: new(1), Name: "Player"},
			{TeamID: 1, AllyTeam: 1, HasAI: true, StartUnit: new(5), Name: "BARbarianAI(1) (UNKNOWN)"},
		},
		UnitDefs: map[int]events.UnitDef{
			1: {Name: "corcom", TranslatedHumanName: "Cortex Commander", BuildSpeed: 300},
			2: {Name: "cormex", TranslatedHumanName: "Metal Extractor", BuildTime: 5700},
			3: {Name: "corlab", TranslatedHumanName: "Bot Lab", BuildTime: 36000, BuildSpeed: 150},
			4: {Name: "corck", TranslatedHumanName: "Construction Bot", BuildTime: 4500},
			5: {Name: "armcom", TranslatedHumanName: "Armada Commander", BuildSpeed: 300},
			6: {Name: "armmex", TranslatedHumanName: "Metal Extractor", BuildTime: 5700},
			7: {Name: "legcom", TranslatedHumanName: "Legion Commander"},
		},
		End: &events.End{Frame: 9300, Reason: events.EndReasonGameOver},
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
