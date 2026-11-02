package events_test

import (
	"path/filepath"
	"strings"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/backbone81/bar-replay-analytics-go/internal/events"
)

var _ = Describe("File", func() {
	Context("Read", func() {
		It("reads the events, the game, the teams, the unit defs and the end", func() {
			Expect(events.Read(strings.NewReader(document(gameOverEnd, unitFinished)))).To(Equal(&events.File{
				Events: []events.Event{
					{Frame: 30, UnitFinished: &events.UnitFinished{UnitID: 2858, UnitDefID: 101, UnitTeam: 0}},
				},
				Game:     events.Game{MapName: "Argent Strata 1.1", MaxUnits: 32000},
				Teams:    []events.Team{{TeamID: 0, StartUnit: new(101), Name: "Player"}},
				UnitDefs: map[int]events.UnitDef{101: {Name: "corcom"}},
				End:      &events.End{Frame: 1800, Reason: events.EndReasonGameOver},
			}))
		})

		It("reads a file without events", func() {
			file, err := events.Read(strings.NewReader(document(gameOverEnd)))

			Expect(err).NotTo(HaveOccurred())
			Expect(file.Events).To(BeEmpty())
		})

		It("rejects a file that is not valid JSON", func() {
			Expect(events.Read(strings.NewReader(`{"events":[`))).Error().To(HaveOccurred())
		})

		It("rejects a file with data after the document", func() {
			Expect(events.Read(strings.NewReader(document(gameOverEnd) + "{}"))).Error().To(HaveOccurred())
		})

		It("rejects a file without end", func() {
			Expect(events.Read(strings.NewReader(`{"events":[` + unitFinished + `]}`))).Error().To(HaveOccurred())
		})

		It("rejects a file whose game did not end", func() {
			stalledEnd := `{"frame":1800,"reason":"stalled"}`

			Expect(events.Read(strings.NewReader(document(stalledEnd, unitFinished)))).Error().To(HaveOccurred())
		})

		It("rejects an event with two types", func() {
			twoTypes := `{"frame":30,"unitFinished":{"unitDefID":101},"unitGiven":{"unitDefID":101}}`

			Expect(events.Read(strings.NewReader(document(gameOverEnd, twoTypes)))).Error().To(HaveOccurred())
		})

		It("rejects frames that go backwards", func() {
			earlier := `{"frame":29,"unitFinished":{"unitDefID":101}}`

			Expect(events.Read(strings.NewReader(document(gameOverEnd, unitFinished, earlier)))).Error().To(HaveOccurred())
		})

		It("rejects an end before the last event", func() {
			earlyEnd := `{"frame":29,"reason":"gameover"}`

			Expect(events.Read(strings.NewReader(document(earlyEnd, unitFinished)))).Error().To(HaveOccurred())
		})

		It("rejects an event that refers to a unit def not in the unit defs", func() {
			unknownUnitDef := `{"frame":30,"unitFinished":{"unitDefID":999}}`

			Expect(events.Read(strings.NewReader(document(gameOverEnd, unknownUnitDef)))).Error().To(HaveOccurred())
		})

		It("rejects a team whose start unit is not in the unit defs", func() {
			data := strings.Replace(document(gameOverEnd), `"startUnit":101`, `"startUnit":999`, 1)

			Expect(events.Read(strings.NewReader(data))).Error().To(HaveOccurred())
		})
	})

	Context("ReadFile", func() {
		It("reads every event type, skips unknown ones and ignores unknown fields", func() {
			Expect(events.ReadFile(filepath.Join("testdata", "events.json"))).To(Equal(testdataFile()))
		})

		It("returns an error for a missing file", func() {
			Expect(events.ReadFile(filepath.Join("testdata", "missing.json"))).Error().To(HaveOccurred())
		})
	})
})

// gameOverEnd is the end of a file whose game ended.
const gameOverEnd = `{"frame":1800,"reason":"gameover"}`

// unitFinished is an event as the widget writes it.
const unitFinished = `{"frame":30,"unitFinished":{"unitID":2858,"unitDefID":101,"unitTeam":0}}`

// document returns an events file with end and the events, as the widget writes it, with one team and one unit def.
func document(end string, events ...string) string {
	return "{\"events\":[\n" + strings.Join(events, ",\n") + "\n]," +
		`"game":{"mapName":"Argent Strata 1.1","maxUnits":32000},` +
		`"teams":[{"teamID":0,"startUnit":101,"name":"Player"}],` +
		`"unitDefs":{"101":{"name":"corcom"}},` +
		`"end":` + end + "}\n"
}

// testdataFile returns the content of testdata/events.json.
func testdataFile() *events.File {
	return &events.File{
		Events: []events.Event{
			{Frame: 0, UnitCreated: &events.UnitCreated{UnitID: 2858, UnitDefID: 101, UnitTeam: 0, X: 1698, Z: 6452}},
			{Frame: 0, UnitFinished: &events.UnitFinished{UnitID: 2858, UnitDefID: 101, UnitTeam: 0}},
			{Frame: 0, UnitIdle: &events.UnitIdle{UnitID: 2858, UnitDefID: 101, UnitTeam: 0}},
			{Frame: 15, UnitCommand: &events.UnitCommand{
				UnitID:    2858,
				UnitDefID: 101,
				UnitTeam:  0,
				CmdID:     -102,
				CmdParams: []float64{1800, 120.5, 6400, 0},
				CmdOpts:   32,
			}},
			{Frame: 30, UnitCreated: &events.UnitCreated{
				UnitID:    1604,
				UnitDefID: 102,
				UnitTeam:  0,
				X:         1800,
				Z:         6400,
				BuilderID: new(2858),
			}},
			{Frame: 900, UnitCmdDone: &events.UnitCmdDone{
				UnitID:    2858,
				UnitDefID: 101,
				UnitTeam:  0,
				CmdID:     -102,
				CmdParams: []float64{1800, 120.5, 6400, 0},
				CmdOpts:   32,
				CmdTag:    7,
			}},
			{Frame: 1200},
			{Frame: 1500, UnitGiven: &events.UnitGiven{UnitID: 1604, UnitDefID: 102, NewTeam: 1, OldTeam: 0}},
			{Frame: 1800, UnitDestroyed: &events.UnitDestroyed{
				UnitID:        2858,
				UnitDefID:     101,
				UnitTeam:      0,
				X:             1700,
				Z:             6450,
				BuildProgress: 1,
				AttackerID:    new(4711),
				AttackerDefID: new(101),
				AttackerTeam:  new(1),
				WeaponDefID:   17,
			}},
			{Frame: 1800, TeamDied: &events.TeamDied{TeamID: 0}},
			{Frame: 1800, GameOver: &events.GameOver{WinningAllyTeams: []int{1}}},
		},
		Game: events.Game{
			MapName:     "Argent Strata 1.1",
			MapSizeX:    8192,
			MapSizeZ:    8192,
			GameName:    "Beyond All Reason",
			GameVersion: "test-31720-0f97aa5",
			MaxUnits:    32000,
		},
		Sides: []events.Side{
			{SideName: "armada", CaseName: "Armada", StartUnit: "armcom"},
			{SideName: "cortex", CaseName: "Cortex", StartUnit: "corcom"},
			{SideName: "random", CaseName: "Random", StartUnit: "dummycom"},
			{SideName: "legion", CaseName: "Legion", StartUnit: "legcom"},
		},
		Teams: []events.Team{
			{
				TeamID:           0,
				Leader:           0,
				IsDead:           false,
				HasAI:            false,
				Side:             "cortex",
				AllyTeam:         0,
				IncomeMultiplier: 1,
				CustomTeamKeys:   map[string]string{},
				StartUnit:        new(101),
				Name:             "Player",
			},
			{
				TeamID:           1,
				Leader:           0,
				IsDead:           true,
				HasAI:            true,
				Side:             "random",
				AllyTeam:         1,
				IncomeMultiplier: 1.5,
				CustomTeamKeys:   map[string]string{"startpos": "1"},
				StartUnit:        new(101),
				Name:             "BARbarianAI(1) (UNKNOWN)",
			},
		},
		TeamStats: map[int][]events.TeamStats{
			0: {
				{},
				{
					Time:             60,
					Frame:            1800,
					MetalUsed:        1100.5,
					MetalProduced:    1200.25,
					MetalExcess:      1,
					MetalReceived:    2,
					MetalSent:        3,
					EnergyUsed:       9000,
					EnergyProduced:   9500,
					EnergyExcess:     4,
					EnergyReceived:   5,
					EnergySent:       6,
					DamageDealt:      700,
					DamageReceived:   800,
					UnitsProduced:    2,
					UnitsDied:        1,
					UnitsReceived:    7,
					UnitsSent:        1,
					UnitsCaptured:    8,
					UnitsOutCaptured: 9,
					UnitsKilled:      10,
				},
			},
		},
		UnitDefs: map[int]events.UnitDef{
			101: {
				Name:                "corcom",
				TranslatedHumanName: "Cortex Commander",
				MetalCost:           2700,
				EnergyCost:          26000,
				BuildTime:           75000,
				IsBuilder:           true,
				CanMove:             true,
				BuildSpeed:          300,
				BuildDistance:       145,
				EnergyMake:          30,
				MaxWeaponRange:      300,
				CustomParams:        map[string]string{"techlevel": "1", "unitgroup": "builder", "iscommander": "1"},
			},
			102: {
				Name:                "corlab",
				TranslatedHumanName: "Bot Lab",
				MetalCost:           600,
				EnergyCost:          1300,
				BuildTime:           6500,
				IsFactory:           true,
				IsBuilder:           true,
				CanMove:             true,
				BuildSpeed:          150,
				BuildDistance:       128,
				CustomParams:        map[string]string{},
			},
		},
		End: &events.End{Frame: 1800, Reason: events.EndReasonGameOver},
	}
}
