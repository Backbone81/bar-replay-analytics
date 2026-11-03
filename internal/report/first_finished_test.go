package report_test

import (
	"cmp"
	"slices"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/backbone81/bar-replay-analytics-go/internal/events"
	"github.com/backbone81/bar-replay-analytics-go/internal/report"
)

var _ = Describe("FirstFinished", func() {
	It("lists the first unit of each unit def each player built over the game, in the order it was finished", func() {
		result, err := report.NewReport(newTestDemo(), newTestSetupScript(), newTestEventsFile())

		Expect(err).NotTo(HaveOccurred())
		Expect(result.Players[0].FirstFinished).To(Equal([]report.FirstFinishedEntry{
			{Time: report.Duration(20 * time.Second), Name: "Metal Extractor"},
			{Time: report.Duration(90 * time.Second), Name: "Bot Lab"},
			{Time: report.Duration(310 * time.Second), Name: "Construction Bot"},
		}))
		Expect(result.Players[1].FirstFinished).To(Equal([]report.FirstFinishedEntry{
			{Time: report.Duration(25 * time.Second), Name: "Metal Extractor"},
		}))
	})

	It("lists a unit def only once", func() {
		eventsFile := newTestEventsFile()
		addEvents(eventsFile,
			events.Event{
				Frame:       3600,
				UnitCreated: &events.UnitCreated{UnitID: 104, UnitDefID: 2, UnitTeam: 0, BuilderID: new(100)},
			},
			events.Event{
				Frame:        4200,
				UnitFinished: &events.UnitFinished{UnitID: 104, UnitDefID: 2, UnitTeam: 0},
			},
		)

		result, err := report.NewReport(newTestDemo(), newTestSetupScript(), eventsFile)

		Expect(err).NotTo(HaveOccurred())
		Expect(firstFinishedNames(result.Players[0].FirstFinished)).
			To(Equal([]string{"Metal Extractor", "Bot Lab", "Construction Bot"}))
	})

	It("lists unit defs of the same name each", func() {
		eventsFile := newTestEventsFile()
		addEvents(eventsFile,
			events.Event{
				Frame:       3600,
				UnitCreated: &events.UnitCreated{UnitID: 104, UnitDefID: 6, UnitTeam: 0, BuilderID: new(100)},
			},
			events.Event{
				Frame:        4200,
				UnitFinished: &events.UnitFinished{UnitID: 104, UnitDefID: 6, UnitTeam: 0},
			},
		)

		result, err := report.NewReport(newTestDemo(), newTestSetupScript(), eventsFile)

		Expect(err).NotTo(HaveOccurred())
		Expect(firstFinishedNames(result.Players[0].FirstFinished)).
			To(Equal([]string{"Metal Extractor", "Bot Lab", "Metal Extractor", "Construction Bot"}))
	})

	It("lists a unit def at the first unit built, not at an earlier resurrected one", func() {
		eventsFile := newTestEventsFile()
		addEvents(eventsFile,
			events.Event{
				Frame:       3600,
				UnitCreated: &events.UnitCreated{UnitID: 104, UnitDefID: 4, UnitTeam: 0, BuilderID: new(100)},
			},
			events.Event{
				Frame:        3600,
				UnitFinished: &events.UnitFinished{UnitID: 104, UnitDefID: 4, UnitTeam: 0},
			},
		)

		result, err := report.NewReport(newTestDemo(), newTestSetupScript(), eventsFile)

		Expect(err).NotTo(HaveOccurred())
		Expect(result.Players[0].FirstFinished[2]).To(Equal(report.FirstFinishedEntry{
			Time: report.Duration(310 * time.Second),
			Name: "Construction Bot",
		}))
	})

	It("leaves out units of Gaia", func() {
		eventsFile := newTestEventsFile()
		addEvents(eventsFile,
			events.Event{
				Frame:       600,
				UnitCreated: &events.UnitCreated{UnitID: 300, UnitDefID: 7, UnitTeam: 2, BuilderID: new(301)},
			},
			events.Event{
				Frame:        900,
				UnitFinished: &events.UnitFinished{UnitID: 300, UnitDefID: 7, UnitTeam: 2},
			},
		)

		result, err := report.NewReport(newTestDemo(), newTestSetupScript(), eventsFile)

		Expect(err).NotTo(HaveOccurred())
		Expect(firstFinishedNames(result.Players[0].FirstFinished)).NotTo(ContainElement("Legion Commander"))
		Expect(firstFinishedNames(result.Players[1].FirstFinished)).NotTo(ContainElement("Legion Commander"))
	})
})

// addEvents adds events to the events of eventsFile, keeping them in the order of their frames.
func addEvents(eventsFile *events.File, added ...events.Event) {
	eventsFile.Events = append(eventsFile.Events, added...)
	slices.SortStableFunc(eventsFile.Events, func(a, b events.Event) int {
		return cmp.Compare(a.Frame, b.Frame)
	})
}

// firstFinishedNames returns the names of first finished entries.
func firstFinishedNames(firstFinished []report.FirstFinishedEntry) []string {
	names := make([]string, len(firstFinished))
	for i, entry := range firstFinished {
		names[i] = entry.Name
	}
	return names
}
