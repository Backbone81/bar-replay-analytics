package report_test

import (
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/backbone81/bar-replay-analytics-go/internal/events"
	"github.com/backbone81/bar-replay-analytics-go/internal/report"
)

var _ = Describe("Combat", func() {
	It("counts the units each player killed and lost, with their metal, over the game and per interval", func() {
		eventsFile := newTestEventsFile()
		addEvents(eventsFile,
			unitKilled(1000, 201, 6, 1, 0),
			events.Event{
				Frame:       1500,
				UnitCreated: &events.UnitCreated{UnitID: 104, UnitDefID: 4, UnitTeam: 0, BuilderID: new(102)},
			},
			events.Event{
				Frame: 1800,
				UnitDestroyed: &events.UnitDestroyed{
					UnitID:        104,
					UnitDefID:     4,
					UnitTeam:      0,
					BuildProgress: 0.5,
					AttackerID:    new(200),
					AttackerDefID: new(5),
					AttackerTeam:  new(1),
					WeaponDefID:   1,
				},
			},
		)

		result, err := newIntervalReport(eventsFile)

		Expect(err).NotTo(HaveOccurred())
		// The metal extractor of player 0 dies at 1:40 without attacker, the half-built construction bot at 1:00, at
		// the end of the second interval.
		Expect(result.Players[0].Combat).To(Equal(report.Combat{
			Total: report.Casualties{
				Killed: report.DeadUnits{Count: 1, Metal: 50},
				Lost:   report.DeadUnits{Count: 2, Metal: 105},
			},
			Intervals: []report.CombatInterval{
				{Time: interval(0, 30)},
				{
					Time: interval(30, 60),
					Casualties: report.Casualties{
						Killed: report.DeadUnits{Count: 1, Metal: 50},
						Lost:   report.DeadUnits{Count: 1, Metal: 55},
					},
				},
				{Time: interval(60, 90)},
				{Time: interval(90, 120), Casualties: report.Casualties{Lost: report.DeadUnits{Count: 1, Metal: 50}}},
				{Time: interval(120, 150)},
			},
		}))
		Expect(result.Players[1].Combat.Total).To(Equal(report.Casualties{
			Killed: report.DeadUnits{Count: 1, Metal: 55},
			Lost:   report.DeadUnits{Count: 1, Metal: 50},
		}))
	})

	It("counts a unit killed by its own player as lost, not as killed", func() {
		eventsFile := newTestEventsFile()
		eventsFile.Events[10] = unitKilled(3000, 101, 2, 0, 0)

		result, err := newIntervalReport(eventsFile)

		Expect(err).NotTo(HaveOccurred())
		Expect(result.Players[0].Combat.Total).To(Equal(report.Casualties{Lost: report.DeadUnits{Count: 1, Metal: 50}}))
	})

	It("counts a self-destructed unit as lost", func() {
		eventsFile := newTestEventsFile()
		eventsFile.Events[10].UnitDestroyed.WeaponDefID = -10

		result, err := newIntervalReport(eventsFile)

		Expect(err).NotTo(HaveOccurred())
		Expect(result.Players[0].Combat.Total).To(Equal(report.Casualties{Lost: report.DeadUnits{Count: 1, Metal: 50}}))
	})

	It("does not count a unit its own player reclaimed", func() {
		eventsFile := newTestEventsFile()
		eventsFile.Events[10] = unitKilled(3000, 101, 2, 0, 0)
		eventsFile.Events[10].UnitDestroyed.WeaponDefID = events.EnvDamageReclaimed

		result, err := newIntervalReport(eventsFile)

		Expect(err).NotTo(HaveOccurred())
		Expect(result.Players[0].Combat.Total).To(BeZero())
	})

	It("counts a unit an enemy reclaimed as lost and as killed by the enemy", func() {
		eventsFile := newTestEventsFile()
		eventsFile.Events[10] = unitKilled(3000, 101, 2, 0, 1)
		eventsFile.Events[10].UnitDestroyed.WeaponDefID = events.EnvDamageReclaimed

		result, err := newIntervalReport(eventsFile)

		Expect(err).NotTo(HaveOccurred())
		Expect(result.Players[0].Combat.Total).To(Equal(report.Casualties{Lost: report.DeadUnits{Count: 1, Metal: 50}}))
		Expect(result.Players[1].Combat.Total).To(Equal(report.Casualties{Killed: report.DeadUnits{Count: 1, Metal: 50}}))
	})

	It("does not count a metal extractor replaced by an upgrade", func() {
		eventsFile := newTestEventsFile()
		eventsFile.Events[10].UnitDestroyed.WeaponDefID = events.EnvDamageKilledByLua

		result, err := newIntervalReport(eventsFile)

		Expect(err).NotTo(HaveOccurred())
		Expect(result.Players[0].Combat.Total).To(BeZero())
	})

	It("counts neither as lost nor as killed the units removed from a dead player", func() {
		eventsFile := newTestEventsFile()
		eventsFile.Events[10] = unitKilled(3000, 101, 2, 0, 1)
		addEvents(eventsFile, events.Event{Frame: 2900, TeamDied: &events.TeamDied{TeamID: 0}})

		result, err := newIntervalReport(eventsFile)

		Expect(err).NotTo(HaveOccurred())
		Expect(result.Players[0].Combat.Total).To(BeZero())
		Expect(result.Players[1].Combat.Total).To(BeZero())
	})

	It("counts a death after the last interval in the last interval", func() {
		eventsFile := newTestEventsFile()
		eventsFile.Events[10].Frame = 4600
		addEvents(eventsFile)

		result, err := newIntervalReport(eventsFile)

		Expect(err).NotTo(HaveOccurred())
		intervals := result.Players[0].Combat.Intervals
		Expect(intervals[len(intervals)-1].Lost).To(Equal(report.DeadUnits{Count: 1, Metal: 50}))
	})

	It("reports the intervals for a player without economy", func() {
		replay := newTestDemo()
		replay.TeamStats = nil

		result, err := report.NewReport(replay, newTestSetupScript(), newTestEventsFile())

		Expect(err).NotTo(HaveOccurred())
		Expect(result.Players[0].Economy).To(BeNil())
		lost := report.Casualties{Lost: report.DeadUnits{Count: 1, Metal: 50}}
		Expect(result.Players[0].Combat).To(Equal(report.Combat{
			Total:     lost,
			Intervals: []report.CombatInterval{{Time: interval(0, 60), Casualties: lost}},
		}))
	})

	It("reports no intervals for a game of no length", func() {
		replay := newTestDemo()
		replay.FileHeader.GameTime = 0

		result, err := report.NewReport(replay, newTestSetupScript(), newTestEventsFile())

		Expect(err).NotTo(HaveOccurred())
		Expect(result.Players[0].Combat.Total.Lost.Count).To(Equal(1))
		Expect(result.Players[0].Combat.Intervals).To(BeEmpty())
	})
})

// unitKilled returns the event of the finished unit with unitID and unitDefID of the player unitTeam dying at frame,
// killed by a weapon of the commander of the player attackerTeam (unit ID 100 for player 0, 200 for player 1).
func unitKilled(frame int, unitID int, unitDefID int, unitTeam int, attackerTeam int) events.Event {
	attackerIDs := []int{100, 200}
	attackerDefIDs := []int{1, 5}
	return events.Event{
		Frame: frame,
		UnitDestroyed: &events.UnitDestroyed{
			UnitID:        unitID,
			UnitDefID:     unitDefID,
			UnitTeam:      unitTeam,
			BuildProgress: 1,
			AttackerID:    new(attackerIDs[attackerTeam]),
			AttackerDefID: new(attackerDefIDs[attackerTeam]),
			AttackerTeam:  new(attackerTeam),
			WeaponDefID:   1,
		},
	}
}

// interval returns the interval from start to end seconds of game time.
func interval(start int, end int) report.Interval {
	return report.Interval{
		Start: report.Duration(time.Duration(start) * time.Second),
		End:   report.Duration(time.Duration(end) * time.Second),
	}
}
