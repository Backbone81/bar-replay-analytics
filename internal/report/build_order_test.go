package report_test

import (
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/backbone81/bar-replay-analytics-go/internal/events"
	"github.com/backbone81/bar-replay-analytics-go/internal/report"
)

var _ = Describe("OpeningBuildOrder", func() {
	It("lists what each player built in the opening, in the order it was finished", func() {
		result, err := report.NewReport(newTestDemo(), newTestSetupScript(), newTestEventsFile())

		Expect(err).NotTo(HaveOccurred())
		Expect(result.Players[0].OpeningBuildOrder).To(Equal([]report.BuildOrderEntry{
			{
				Time: report.Duration(20 * time.Second),
				Name: "Metal Extractor",
			},
			{
				Time: report.Duration(90 * time.Second),
				Name: "Bot Lab",
			},
		}))
		Expect(result.Players[1].OpeningBuildOrder).To(Equal([]report.BuildOrderEntry{
			{
				Time: report.Duration(25 * time.Second),
				Name: "Metal Extractor",
			},
		}))
	})

	It("leaves out what was finished after the opening, though started in it", func() {
		result, err := report.NewReport(
			newTestDemo(),
			newTestSetupScript(),
			newTestEventsFile(),
			report.WithOpeningLength(time.Minute),
		)

		Expect(err).NotTo(HaveOccurred())
		Expect(buildOrderNames(result.Players[0].OpeningBuildOrder)).To(Equal([]string{"Metal Extractor"}))
	})

	It("includes what was finished in the last frame of the opening", func() {
		eventsFile := newTestEventsFile()
		eventsFile.Events[13].Frame = 17999
		eventsFile.Events[14].Frame = 17999

		result, err := report.NewReport(newTestDemo(), newTestSetupScript(), eventsFile)

		Expect(err).NotTo(HaveOccurred())
		Expect(buildOrderNames(result.Players[0].OpeningBuildOrder)).
			To(Equal([]string{"Metal Extractor", "Bot Lab", "Construction Bot"}))
	})

	It("is empty without an opening", func() {
		result, err := report.NewReport(
			newTestDemo(),
			newTestSetupScript(),
			newTestEventsFile(),
			report.WithOpeningLength(0),
		)

		Expect(err).NotTo(HaveOccurred())
		Expect(result.Players[0].OpeningBuildOrder).NotTo(BeNil())
		Expect(result.Players[0].OpeningBuildOrder).To(BeEmpty())
	})

	It("leaves out units of Gaia", func() {
		eventsFile := newTestEventsFile()
		eventsFile.Events = append(eventsFile.Events[:6],
			events.Event{
				Frame: 600,
				UnitCreated: &events.UnitCreated{
					UnitID:    300,
					UnitDefID: 2,
					UnitTeam:  2,
					BuilderID: new(301),
				},
			},
			events.Event{
				Frame: 600,
				UnitFinished: &events.UnitFinished{
					UnitID:    300,
					UnitDefID: 2,
					UnitTeam:  2,
				},
			},
		)

		result, err := report.NewReport(newTestDemo(), newTestSetupScript(), eventsFile)

		Expect(err).NotTo(HaveOccurred())
		Expect(result.Players[0].OpeningBuildOrder).To(BeEmpty())
		Expect(result.Players[1].OpeningBuildOrder).To(BeEmpty())
	})

	Context("events file", func() {
		It("rejects an invalid events file", func() {
			eventsFile := newTestEventsFile()
			eventsFile.End = nil

			Expect(report.NewReport(newTestDemo(), newTestSetupScript(), eventsFile)).Error().To(HaveOccurred())
		})

		DescribeTable("rejects an events file of another game",
			func(change func(eventsFile *events.File)) {
				eventsFile := newTestEventsFile()
				change(eventsFile)

				Expect(report.NewReport(newTestDemo(), newTestSetupScript(), eventsFile)).Error().To(HaveOccurred())
			},
			Entry("on another map", changeMapName),
			Entry("of another game version", changeGameVersion),
			Entry("with another number of teams", removeLastTeam),
		)
	})
})

// buildOrderNames returns the names of a build order.
func buildOrderNames(buildOrder []report.BuildOrderEntry) []string {
	names := make([]string, len(buildOrder))
	for i, entry := range buildOrder {
		names[i] = entry.Name
	}
	return names
}

// changeMapName sets another map in the events file.
func changeMapName(eventsFile *events.File) {
	eventsFile.Game.MapName = "Onyx Cauldron 2.2.2"
}

// changeGameVersion sets another game version in the events file.
func changeGameVersion(eventsFile *events.File) {
	eventsFile.Game.GameVersion = "test-31720-0f97aa5"
}

// removeLastTeam removes the last team of the events file.
func removeLastTeam(eventsFile *events.File) {
	eventsFile.Teams = eventsFile.Teams[:len(eventsFile.Teams)-1]
}
