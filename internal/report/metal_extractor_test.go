package report_test

import (
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/backbone81/bar-replay-analytics-go/internal/demo"
	"github.com/backbone81/bar-replay-analytics-go/internal/events"
	"github.com/backbone81/bar-replay-analytics-go/internal/report"
)

var _ = Describe("MetalExtractors", func() {
	It("counts the metal extractors each player held at the end of each interval", func() {
		result, err := newMetalExtractorReport(newTestEventsFile())

		Expect(err).NotTo(HaveOccurred())
		// The metal extractor of player 0 is finished at 0:20 and dies at 1:40, that of player 1 is finished at 0:25.
		Expect(metalExtractors(result.Players[0])).To(Equal([]report.MetalExtractors{
			{Tech1: 1}, {Tech1: 1}, {Tech1: 1}, {}, {},
		}))
		Expect(metalExtractors(result.Players[1])).To(Equal([]report.MetalExtractors{
			{Tech1: 1}, {Tech1: 1}, {Tech1: 1}, {Tech1: 1}, {Tech1: 1},
		}))
	})

	It("counts tech 2 metal extractors apart, also one replacing a tech 1 one", func() {
		eventsFile := newTestEventsFile()
		addEvents(eventsFile,
			events.Event{
				Frame:       2400,
				UnitCreated: &events.UnitCreated{UnitID: 104, UnitDefID: 8, UnitTeam: 0, BuilderID: new(100)},
			},
			events.Event{
				Frame:        3000,
				UnitFinished: &events.UnitFinished{UnitID: 104, UnitDefID: 8, UnitTeam: 0},
			},
		)

		result, err := newMetalExtractorReport(eventsFile)

		Expect(err).NotTo(HaveOccurred())
		Expect(metalExtractors(result.Players[0])).To(Equal([]report.MetalExtractors{
			{Tech1: 1}, {Tech1: 1}, {Tech1: 1}, {Tech2: 1}, {Tech2: 1},
		}))
	})

	It("counts a metal extractor for the player it was given to", func() {
		eventsFile := newTestEventsFile()
		addEvents(eventsFile, events.Event{
			Frame:     1500,
			UnitGiven: &events.UnitGiven{UnitID: 201, UnitDefID: 6, NewTeam: 0, OldTeam: 1},
		})

		result, err := newMetalExtractorReport(eventsFile)

		Expect(err).NotTo(HaveOccurred())
		Expect(metalExtractors(result.Players[0])[1]).To(Equal(report.MetalExtractors{Tech1: 2}))
		Expect(metalExtractors(result.Players[1])[1]).To(Equal(report.MetalExtractors{}))
	})

	It("stops counting a metal extractor given to a team which is no player", func() {
		eventsFile := newTestEventsFile()
		addEvents(eventsFile, events.Event{
			Frame:     1500,
			UnitGiven: &events.UnitGiven{UnitID: 201, UnitDefID: 6, NewTeam: 2, OldTeam: 1},
		})

		result, err := newMetalExtractorReport(eventsFile)

		Expect(err).NotTo(HaveOccurred())
		Expect(metalExtractors(result.Players[1])[1]).To(Equal(report.MetalExtractors{}))
	})

	It("does not count a metal extractor which died in the frame an interval ends at", func() {
		eventsFile := newTestEventsFile()
		eventsFile.Events[10].Frame = 1800
		addEvents(eventsFile)

		result, err := newMetalExtractorReport(eventsFile)

		Expect(err).NotTo(HaveOccurred())
		Expect(metalExtractors(result.Players[0])[:2]).To(Equal([]report.MetalExtractors{{Tech1: 1}, {}}))
	})

	It("does not count an unfinished metal extractor", func() {
		eventsFile := newTestEventsFile()
		addEvents(eventsFile, events.Event{
			Frame:       2400,
			UnitCreated: &events.UnitCreated{UnitID: 104, UnitDefID: 8, UnitTeam: 0, BuilderID: new(100)},
		})

		result, err := newMetalExtractorReport(eventsFile)

		Expect(err).NotTo(HaveOccurred())
		Expect(metalExtractors(result.Players[0])[3]).To(Equal(report.MetalExtractors{}))
	})

	It("counts a metal extractor without tech level as tech 1", func() {
		eventsFile := newTestEventsFile()
		eventsFile.UnitDefs[2] = events.UnitDef{Name: "cormex", ExtractsMetal: 0.001}

		result, err := newMetalExtractorReport(eventsFile)

		Expect(err).NotTo(HaveOccurred())
		Expect(metalExtractors(result.Players[0])[0]).To(Equal(report.MetalExtractors{Tech1: 1}))
	})

	It("rejects a metal extractor whose tech level is not a number", func() {
		eventsFile := newTestEventsFile()
		eventsFile.UnitDefs[2] = events.UnitDef{
			Name:          "cormex",
			ExtractsMetal: 0.001,
			CustomParams:  map[string]string{"techlevel": "one"},
		}

		Expect(newMetalExtractorReport(eventsFile)).Error().To(HaveOccurred())
	})
})

// newMetalExtractorReport returns the report of a game of 150 s with eventsFile, with intervals of 30 s and no
// opening: intervals end at 0:30, 1:00, 1:30, 2:00 and 2:30.
func newMetalExtractorReport(eventsFile *events.File) (*report.Report, error) {
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
		report.WithOpeningLength(0),
		report.WithIntervalLength(30*time.Second),
	)
}

// metalExtractors returns the metal extractors a player held at the end of each interval.
func metalExtractors(player report.Player) []report.MetalExtractors {
	result := make([]report.MetalExtractors, len(player.Economy.Intervals))
	for i, interval := range player.Economy.Intervals {
		result[i] = interval.MetalExtractors
	}
	return result
}
