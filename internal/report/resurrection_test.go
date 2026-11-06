package report_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/backbone81/bar-replay-analytics-go/internal/events"
	"github.com/backbone81/bar-replay-analytics-go/internal/report"
)

var _ = Describe("Resurrection", func() {
	It("counts the metal of the units each player resurrected and their energy, over the game and per interval", func() {
		eventsFile := newTestEventsFile()
		construction := eventsFile.UnitDefs[4]
		construction.EnergyCost = 1700
		eventsFile.UnitDefs[4] = construction
		addEvents(eventsFile,
			// A Graverobber built by the bot lab, which takes longer than a frame.
			unitCreated(1000, 105, 10, 102),
			unitFinished(1400, 105, 10),
			// Resurrected at 1:00, the end of the second interval, and after the end of the game.
			unitCreated(1800, 106, 4, 105),
			unitFinished(1800, 106, 4),
			unitCreated(4600, 107, 2, 105),
			unitFinished(4600, 107, 2),
		)

		result, err := newIntervalReport(eventsFile)

		Expect(err).NotTo(HaveOccurred())
		Expect(result.Players[0].Resurrection).To(Equal(report.Resurrection{
			Total: report.Resurrected{MetalValue: 160, EnergySpent: 850},
			Intervals: []report.ResurrectionInterval{
				{Time: interval(0, 30)},
				{Time: interval(30, 60), Resurrected: report.Resurrected{MetalValue: 110, EnergySpent: 850}},
				{Time: interval(60, 90)},
				{Time: interval(90, 120)},
				{Time: interval(120, 150), Resurrected: report.Resurrected{MetalValue: 50}},
			},
		}))
		Expect(result.Players[1].Resurrection.Total).To(BeZero())
	})

	It("does not count a unit a resurrection bot built, nor one created finished by another builder", func() {
		eventsFile := newTestEventsFile()
		addEvents(eventsFile,
			unitCreated(1000, 105, 10, 102),
			unitFinished(1400, 105, 10),
			unitCreated(1800, 106, 2, 105),
			unitFinished(2100, 106, 2),
			unitCreated(2400, 107, 2, 100),
			unitFinished(2400, 107, 2),
		)

		result, err := newIntervalReport(eventsFile)

		Expect(err).NotTo(HaveOccurred())
		Expect(result.Players[0].Resurrection.Total).To(BeZero())
	})

	It("reports no intervals for a game of no length", func() {
		replay := newTestDemo()
		replay.FileHeader.GameTime = 0

		result, err := report.NewReport(replay, newTestSetupScript(), newTestEventsFile())

		Expect(err).NotTo(HaveOccurred())
		Expect(result.Players[0].Resurrection).To(Equal(report.Resurrection{
			Intervals: []report.ResurrectionInterval{},
		}))
	})
})

// unitCreated returns the event of the unit with unitID and unitDefID of player 0 started by builderID at frame.
func unitCreated(frame int, unitID int, unitDefID int, builderID int) events.Event {
	return events.Event{
		Frame:       frame,
		UnitCreated: &events.UnitCreated{UnitID: unitID, UnitDefID: unitDefID, UnitTeam: 0, BuilderID: new(builderID)},
	}
}

// unitFinished returns the event of the unit with unitID and unitDefID of player 0 finished at frame.
func unitFinished(frame int, unitID int, unitDefID int) events.Event {
	return events.Event{
		Frame:        frame,
		UnitFinished: &events.UnitFinished{UnitID: unitID, UnitDefID: unitDefID, UnitTeam: 0},
	}
}
