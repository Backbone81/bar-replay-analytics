package report_test

import (
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/backbone81/bar-replay-analytics-go/internal/demo"
	"github.com/backbone81/bar-replay-analytics-go/internal/events"
	"github.com/backbone81/bar-replay-analytics-go/internal/report"
)

var _ = Describe("LossLocations", func() {
	It("counts the units and buildings each player lost per cell, in one interval for the opening", func() {
		eventsFile := newTestEventsFile()
		eventsFile.UnitDefs[12] = events.UnitDef{
			Name:                "armllt",
			TranslatedHumanName: "Sentinel",
			MetalCost:           80,
			MaxWeaponRange:      430,
		}
		// Player 1's metal extractor dies in the opening at the top right, killed by the Cortex Commander. A
		// construction bot of player 0 dies there at 1:33, killed by a Sentinel, after the metal extractor of player 0
		// in the top left cell at 1:40.
		killedByTower := unitKilled(2800, 104, 4, 0, 1)
		killedByTower.UnitDestroyed.AttackerDefID = new(12)
		killedByTower.UnitDestroyed.X = 6500
		killedByTower.UnitDestroyed.Z = 1500
		killedMetalExtractor := unitKilled(1000, 201, 6, 1, 0)
		killedMetalExtractor.UnitDestroyed.X = 6600
		killedMetalExtractor.UnitDestroyed.Z = 1800
		addEvents(eventsFile,
			killedMetalExtractor,
			unitCreated(2750, 104, 4, 102),
			unitFinished(2760, 104, 4),
			killedByTower,
		)

		result, err := newOpeningReport(eventsFile)

		Expect(err).NotTo(HaveOccurred())
		Expect(result.Players[0].LossLocations).To(Equal([]report.LossLocationInterval{
			{Time: interval(0, 60), Cells: []report.CellLosses{}},
			{Time: interval(60, 90), Cells: []report.CellLosses{}},
			{
				Time: interval(90, 120),
				Cells: []report.CellLosses{
					{
						Cell:              "D1",
						Units:             report.DeadUnits{Count: 1, Metal: 110},
						KilledByBuildings: report.DeadUnits{Count: 1, Metal: 110},
					},
					{Cell: "A1", Buildings: report.DeadUnits{Count: 1, Metal: 50}},
				},
			},
			{Time: interval(120, 150), Cells: []report.CellLosses{}},
		}))
		Expect(result.Players[1].LossLocations[0]).To(Equal(report.LossLocationInterval{
			Time:  interval(0, 60),
			Cells: []report.CellLosses{{Cell: "D1", Buildings: report.DeadUnits{Count: 1, Metal: 50}}},
		}))
	})

	It("follows the intervals of the economy without opening", func() {
		result, err := newIntervalReport(newTestEventsFile())

		Expect(err).NotTo(HaveOccurred())
		Expect(result.Players[0].LossLocations).To(HaveLen(len(result.Players[0].Economy.Intervals)))
	})
})

// newOpeningReport returns the report of the events file in a game of 150 s, with an opening of 60 s in intervals of
// 30 s, and intervals of 30 s after it.
func newOpeningReport(eventsFile *events.File) (*report.Report, error) {
	replay := newTestDemo()
	replay.FileHeader.GameTime = 150
	replay.TeamStats = [][]demo.TeamStatistics{
		metalHistory(periodTimes(150)...),
		metalHistory(periodTimes(150)...),
	}
	return report.NewReport(
		replay,
		newTestSetupScript(),
		eventsFile,
		report.WithOpeningLength(60*time.Second),
		report.WithOpeningIntervalLength(30*time.Second),
		report.WithIntervalLength(30*time.Second),
	)
}
