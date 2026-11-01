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
		It("reads the events and the end", func() {
			Expect(events.Read(strings.NewReader(document(gameOverEnd, unitFinished)))).To(Equal(&events.File{
				Events: []events.Event{
					{Frame: 30, UnitFinished: &events.UnitFinished{UnitID: 2858, UnitDef: "corcom", TeamID: 0}},
				},
				End: &events.End{Frame: 1800, Reason: events.EndReasonGameOver},
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
			twoTypes := `{"frame":30,"unitFinished":{"unitID":1},"unitGiven":{"unitID":1}}`

			Expect(events.Read(strings.NewReader(document(gameOverEnd, twoTypes)))).Error().To(HaveOccurred())
		})

		It("rejects frames that go backwards", func() {
			earlier := `{"frame":29,"unitFinished":{"unitID":1}}`

			Expect(events.Read(strings.NewReader(document(gameOverEnd, unitFinished, earlier)))).Error().To(HaveOccurred())
		})

		It("rejects an end before the last event", func() {
			earlyEnd := `{"frame":29,"reason":"gameover"}`

			Expect(events.Read(strings.NewReader(document(earlyEnd, unitFinished)))).Error().To(HaveOccurred())
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
const unitFinished = `{"frame":30,"unitFinished":{"unitID":2858,"unitDef":"corcom","teamID":0}}`

// document returns an events file with end and the events, as the widget writes it.
func document(end string, events ...string) string {
	return "{\"events\":[\n" + strings.Join(events, ",\n") + "\n],\"end\":" + end + "}\n"
}

// testdataFile returns the content of testdata/events.json.
func testdataFile() *events.File {
	return &events.File{
		Events: []events.Event{
			{Frame: 0, Init: &events.Init{
				Map:      "Argent Strata 1.1",
				MapSizeX: 8192,
				MapSizeZ: 8192,
				Game:     "Beyond All Reason test-31756-0d4c3c3",
				Teams: []events.Team{
					{TeamID: 0, AllyTeamID: 0, IsAI: false, Side: "cortex", Name: "Player"},
					{TeamID: 1, AllyTeamID: 1, IsAI: true, Side: "random", Name: "BARbarianAI(1) (UNKNOWN)"},
				},
			}},
			{Frame: 0, UnitDef: &events.UnitDef{
				Name:           "corcom",
				HumanName:      "Cortex Commander",
				MetalCost:      2700,
				EnergyCost:     26000,
				BuildTime:      75000,
				TechLevel:      1,
				UnitGroup:      "builder",
				IsBuilder:      true,
				IsCommander:    true,
				IsMobile:       true,
				BuildDistance:  145,
				EnergyMake:     30,
				MaxWeaponRange: 300,
			}},
			{Frame: 0, UnitCreated: &events.UnitCreated{
				UnitID:   2858,
				UnitDef:  "corcom",
				TeamID:   0,
				Position: &events.Position{1698, 6452},
			}},
			{Frame: 0, UnitFinished: &events.UnitFinished{UnitID: 2858, UnitDef: "corcom", TeamID: 0}},
			{Frame: 0, BuilderState: &events.BuilderState{UnitID: 2858, State: events.BuilderStateIdle}},
			{Frame: 0, Resources: &events.Resources{
				TeamID:           0,
				Metal:            events.Resource{Current: 1000, Storage: 1000},
				Energy:           events.Resource{Current: 1000, Storage: 1000},
				UnitCount:        1,
				BuilderCount:     1,
				IdleBuilderCount: 1,
			}},
			{Frame: 30, UnitCreated: &events.UnitCreated{
				UnitID:     1604,
				UnitDef:    "corlab",
				TeamID:     0,
				Position:   &events.Position{1800, 6400},
				BuilderID:  new(2858),
				BuilderDef: "corcom",
			}},
			{Frame: 45, BuilderState: &events.BuilderState{
				UnitID:   2858,
				State:    events.BuilderStateWorking,
				Command:  new(-151),
				TargetID: new(1604),
			}},
			{Frame: 900, ArmyPositions: &events.ArmyPositions{
				TeamID: 0,
				Units:  []events.ArmyUnit{{UnitDef: "corak", Position: events.Position{2000, 6000}}},
			}},
			{Frame: 1200},
			{Frame: 1500, UnitGiven: &events.UnitGiven{UnitID: 1604, UnitDef: "corlab", TeamID: 1, OldTeamID: 0}},
			{Frame: 1800, UnitDestroyed: &events.UnitDestroyed{
				UnitID:         2858,
				UnitDef:        "corcom",
				TeamID:         0,
				Position:       &events.Position{1700, 6450},
				BuildProgress:  1,
				AttackerDef:    "corak",
				AttackerTeamID: new(1),
				Weapon:         "corak_gun",
			}},
			{Frame: 1800, GameOver: &events.GameOver{WinningAllyTeamIDs: []int{1}}},
		},
		End: &events.End{Frame: 1800, Reason: events.EndReasonGameOver},
	}
}
