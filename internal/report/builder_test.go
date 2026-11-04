package report_test

import (
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/backbone81/bar-replay-analytics-go/internal/events"
	"github.com/backbone81/bar-replay-analytics-go/internal/report"
)

var _ = Describe("Builders", func() {
	It("lists each commander and constructor with the span it was held, its idle time and its builds", func() {
		eventsFile := newTestEventsFile()
		addEvents(eventsFile, commanderIdleEvent(2700, 100), commanderCommandEvent(6300, -2, 1))

		result, err := report.NewReport(newTestDemo(), newTestSetupScript(), eventsFile)

		Expect(err).NotTo(HaveOccurred())
		Expect(result.Players[0].Builders[0]).To(Equal(report.Builder{
			Name:         "Cortex Commander 1",
			Time:         report.Interval{Start: 0, End: report.Duration(310 * time.Second)},
			Idle:         report.Duration(120 * time.Second),
			IdleFraction: 0.39,
			IdleStretches: []report.Interval{
				{Start: report.Duration(90 * time.Second), End: report.Duration(210 * time.Second)},
			},
			Builds: []report.BuildCount{
				{Name: "Metal Extractor", Count: 1},
				{Name: "Bot Lab", Count: 1},
			},
		}))
		Expect(builderNames(result.Players[0].Builders)).
			To(Equal([]string{"Cortex Commander 1", "Cortex Commander 2", "Construction Bot 1"}))
		Expect(builderNames(result.Players[1].Builders)).To(Equal([]string{"Armada Commander 1"}))
	})

	It("counts short idle time but lists only stretches of at least the idle stretch length", func() {
		eventsFile := newTestEventsFile()
		addEvents(eventsFile, commanderIdleEvent(2700, 100), commanderCommandEvent(3000, -2, 1))

		result, err := report.NewReport(newTestDemo(), newTestSetupScript(), eventsFile)
		Expect(err).NotTo(HaveOccurred())
		shortResult, err := report.NewReport(newTestDemo(), newTestSetupScript(), eventsFile,
			report.WithIdleStretchLength(5*time.Second),
		)
		Expect(err).NotTo(HaveOccurred())

		Expect(result.Players[0].Builders[0].Idle).To(Equal(report.Duration(10 * time.Second)))
		Expect(result.Players[0].Builders[0].IdleStretches).To(BeEmpty())
		Expect(shortResult.Players[0].Builders[0].IdleStretches).To(Equal([]report.Interval{
			{Start: report.Duration(90 * time.Second), End: report.Duration(100 * time.Second)},
		}))
	})

	It("keeps a builder idle through commands which leave its queue empty", func() {
		eventsFile := newTestEventsFile()
		addEvents(eventsFile,
			commanderIdleEvent(2700, 100),
			commanderCommandEvent(3000, 45, 0),
			commanderCommandEvent(6300, 10, 1),
		)

		result, err := report.NewReport(newTestDemo(), newTestSetupScript(), eventsFile)

		Expect(err).NotTo(HaveOccurred())
		Expect(result.Players[0].Builders[0].Idle).To(Equal(report.Duration(120 * time.Second)))
	})

	It("ends the idle time and the span of a builder when it dies", func() {
		eventsFile := newTestEventsFile()
		addEvents(eventsFile,
			commanderIdleEvent(3300, 101),
			events.Event{
				Frame:         6000,
				UnitDestroyed: &events.UnitDestroyed{UnitID: 101, UnitDefID: 1, UnitTeam: 0, BuildProgress: 1},
			},
		)

		result, err := report.NewReport(newTestDemo(), newTestSetupScript(), eventsFile)

		Expect(err).NotTo(HaveOccurred())
		builder := result.Players[0].Builders[1]
		Expect(builder.Time).To(Equal(report.Interval{
			Start: report.Duration(110 * time.Second),
			End:   report.Duration(200 * time.Second),
		}))
		Expect(builder.Idle).To(Equal(report.Duration(90 * time.Second)))
		Expect(builder.IdleFraction).To(BeNumerically("==", 1))
	})

	It("moves a given builder to the player it was given to", func() {
		eventsFile := newTestEventsFile()
		addEvents(eventsFile, events.Event{
			Frame:     4500,
			UnitGiven: &events.UnitGiven{UnitID: 100, UnitDefID: 1, NewTeam: 1, OldTeam: 0},
		})

		result, err := report.NewReport(newTestDemo(), newTestSetupScript(), eventsFile)

		Expect(err).NotTo(HaveOccurred())
		Expect(result.Players[0].Builders[0].Time).To(Equal(report.Interval{
			Start: 0,
			End:   report.Duration(150 * time.Second),
		}))
		Expect(builderNames(result.Players[1].Builders)).To(Equal([]string{"Armada Commander 1", "Cortex Commander 1"}))
		Expect(result.Players[1].Builders[1].Time).To(Equal(report.Interval{
			Start: report.Duration(150 * time.Second),
			End:   report.Duration(310 * time.Second),
		}))
	})

	It("classes constructors by tech level and leaves out factories and resurrection bots", func() {
		eventsFile := newTestEventsFile()
		addEvents(eventsFile,
			events.Event{
				Frame:       3600,
				UnitCreated: &events.UnitCreated{UnitID: 104, UnitDefID: 9, UnitTeam: 0, BuilderID: new(102)},
			},
			events.Event{
				Frame:       3600,
				UnitCreated: &events.UnitCreated{UnitID: 105, UnitDefID: 10, UnitTeam: 0, BuilderID: new(102)},
			},
			events.Event{Frame: 4200, UnitFinished: &events.UnitFinished{UnitID: 104, UnitDefID: 9, UnitTeam: 0}},
			events.Event{Frame: 4200, UnitFinished: &events.UnitFinished{UnitID: 105, UnitDefID: 10, UnitTeam: 0}},
			events.Event{
				Frame:       3600,
				UnitCreated: &events.UnitCreated{UnitID: 106, UnitDefID: 4, UnitTeam: 0, BuilderID: new(102)},
			},
			events.Event{Frame: 4200, UnitFinished: &events.UnitFinished{UnitID: 106, UnitDefID: 4, UnitTeam: 0}},
		)

		result, err := newIntervalReport(eventsFile)

		Expect(err).NotTo(HaveOccurred())
		Expect(builderNames(result.Players[0].Builders)).To(Equal([]string{
			"Cortex Commander 1",
			"Cortex Commander 2",
			"Advanced Construction Bot 1",
			"Construction Bot 1",
			"Construction Bot 2",
		}))
		// The constructors are held from 2:20, a third of the last interval, the Graverobber is no constructor.
		lastInterval := result.Players[0].Economy.Intervals[4]
		Expect(lastInterval.Builders.Tech1).To(Equal(report.BuilderUse{Alive: 0.33}))
		Expect(lastInterval.Builders.Tech2).To(Equal(report.BuilderUse{Alive: 0.33}))
	})

	It("counts the builds of a unit def together, most first", func() {
		eventsFile := newTestEventsFile()
		addEvents(eventsFile,
			events.Event{
				Frame:       3600,
				UnitCreated: &events.UnitCreated{UnitID: 104, UnitDefID: 2, UnitTeam: 0, BuilderID: new(100)},
			},
			events.Event{Frame: 4200, UnitFinished: &events.UnitFinished{UnitID: 104, UnitDefID: 2, UnitTeam: 0}},
		)

		result, err := report.NewReport(newTestDemo(), newTestSetupScript(), eventsFile)

		Expect(err).NotTo(HaveOccurred())
		Expect(result.Players[0].Builders[0].Builds).To(Equal([]report.BuildCount{
			{Name: "Metal Extractor", Count: 2},
			{Name: "Bot Lab", Count: 1},
		}))
	})

	It("averages how many builders were alive and idle in each interval", func() {
		eventsFile := newTestEventsFile()
		addEvents(eventsFile, commanderIdleEvent(2700, 100), commanderCommandEvent(4500, -2, 1))

		result, err := newIntervalReport(eventsFile)

		Expect(err).NotTo(HaveOccurred())
		// Commander 1 is idle from 1:30 to 2:30, commander 2 is held from 1:50.
		Expect(commanderUse(result.Players[0])).To(Equal([]report.BuilderUse{
			{Alive: 1}, {Alive: 1}, {Alive: 1}, {Alive: 1.33, Idle: 1}, {Alive: 2, Idle: 1},
		}))
	})

	It("rejects a builder whose tech level is not a number", func() {
		eventsFile := newTestEventsFile()
		eventsFile.UnitDefs[4] = events.UnitDef{
			Name:         "corck",
			IsBuilder:    true,
			CanMove:      true,
			CustomParams: map[string]string{"techlevel": "one"},
		}

		Expect(report.NewReport(newTestDemo(), newTestSetupScript(), eventsFile)).Error().To(HaveOccurred())
	})
})

// commanderIdleEvent returns the event of the Cortex commander of player 0 with unitID running out of orders at frame.
func commanderIdleEvent(frame int, unitID int) events.Event {
	return events.Event{
		Frame:    frame,
		UnitIdle: &events.UnitIdle{UnitID: unitID, UnitDefID: 1, UnitTeam: 0},
	}
}

// commanderCommandEvent returns the event of the start commander of player 0 being given the command cmdID at frame,
// with commandCount commands in its queue after it.
func commanderCommandEvent(frame int, cmdID int, commandCount int) events.Event {
	return events.Event{
		Frame: frame,
		UnitCommand: &events.UnitCommand{
			UnitID:       100,
			UnitDefID:    1,
			UnitTeam:     0,
			CmdID:        cmdID,
			CommandCount: commandCount,
		},
	}
}

// builderNames returns the names of builders.
func builderNames(builders []report.Builder) []string {
	names := make([]string, len(builders))
	for i, builder := range builders {
		names[i] = builder.Name
	}
	return names
}

// commanderUse returns the use of a player's commanders in each interval.
func commanderUse(player report.Player) []report.BuilderUse {
	result := make([]report.BuilderUse, len(player.Economy.Intervals))
	for i, interval := range player.Economy.Intervals {
		result[i] = interval.Builders.Commander
	}
	return result
}
