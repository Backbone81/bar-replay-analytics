package report_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/backbone81/bar-replay-analytics/internal/report"
)

var _ = Describe("MapGrid", func() {
	It("divides the map into cells of at most the cell size, along each side apart", func() {
		eventsFile := newTestEventsFile()
		eventsFile.Game.MapSizeX = 10240
		eventsFile.Game.MapSizeZ = 5120
		eventsFile.Events[0].UnitCreated.Z = 4000
		eventsFile.Events[2].UnitCreated.X = 10000

		result, err := report.NewReport(newTestDemo(), newTestSetupScript(), eventsFile)

		Expect(err).NotTo(HaveOccurred())
		Expect(result.MapGrid).To(Equal(report.MapGrid{
			MapSize:  report.Size{X: 10240, Z: 5120},
			Columns:  5,
			Rows:     3,
			CellSize: report.Size{X: 2048, Z: 1706},
			StartPositions: []report.StartPosition{
				{PlayerID: 0, Cell: "A3", X: 1698, Z: 4000},
				{PlayerID: 1, Cell: "E2", X: 10000, Z: 1798},
			},
		}))
	})

	It("names columns past Z as spreadsheets do, and puts a position off the map into the nearest cell", func() {
		eventsFile := newTestEventsFile()
		eventsFile.Events[0].UnitCreated.X = 8191
		eventsFile.Events[2].UnitCreated.X = -5
		eventsFile.Events[2].UnitCreated.Z = 9000

		result, err := report.NewReport(
			newTestDemo(),
			newTestSetupScript(),
			eventsFile,
			report.WithMapGridCellSize(100),
		)

		Expect(err).NotTo(HaveOccurred())
		Expect(result.MapGrid.Columns).To(Equal(82))
		Expect(result.MapGrid.StartPositions[0].Cell).To(Equal("CD65"))
		Expect(result.MapGrid.StartPositions[1].Cell).To(Equal("A82"))
	})

	It("leaves out a player without start unit", func() {
		eventsFile := newTestEventsFile()
		// The Armada Commander of player 1 is built, so it is no start unit.
		eventsFile.Events[2].UnitCreated.BuilderID = new(100)

		result, err := report.NewReport(newTestDemo(), newTestSetupScript(), eventsFile)

		Expect(err).NotTo(HaveOccurred())
		Expect(result.MapGrid.StartPositions).To(Equal([]report.StartPosition{
			{PlayerID: 0, Cell: "A4", X: 1698, Z: 6452},
		}))
	})

	It("rejects an events file without map size", func() {
		eventsFile := newTestEventsFile()
		eventsFile.Game.MapSizeX = 0

		Expect(report.NewReport(newTestDemo(), newTestSetupScript(), eventsFile)).Error().To(HaveOccurred())
	})

	It("rejects a cell size which is not positive", func() {
		Expect(report.NewReport(
			newTestDemo(),
			newTestSetupScript(),
			newTestEventsFile(),
			report.WithMapGridCellSize(0),
		)).Error().To(HaveOccurred())
	})
})
