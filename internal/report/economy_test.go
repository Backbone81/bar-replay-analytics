package report_test

import (
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/backbone81/bar-replay-analytics/internal/demo"
	"github.com/backbone81/bar-replay-analytics/internal/events"
	"github.com/backbone81/bar-replay-analytics/internal/report"
)

var _ = Describe("Economy", func() {
	Context("Total and PerSecond", func() {
		It("takes the totals from the last entry and averages them over the game", func() {
			result, err := report.NewReport(newTestDemo(), newTestSetupScript(), newTestEventsFile())

			Expect(err).NotTo(HaveOccurred())
			economy := result.Players[0].Economy
			Expect(economy.Total).To(Equal(report.Resources{
				Metal:  report.Resource{Produced: 420, Used: 360},
				Energy: report.Resource{Produced: 2400, Wasted: 240},
			}))
			Expect(economy.PerSecond).To(Equal(report.Resources{
				Metal:  report.Resource{Produced: 7, Used: 6},
				Energy: report.Resource{Produced: 40, Wasted: 4},
			}))
		})

		It("includes what was shared with allies", func() {
			result, err := report.NewReport(newTestDemo(), newTestSetupScript(), newTestEventsFile())

			Expect(err).NotTo(HaveOccurred())
			economy := result.Players[1].Economy
			Expect(economy.Total.Metal.Sent).To(BeNumerically("==", 30))
			Expect(economy.Total.Energy.Received).To(BeNumerically("==", 90))
			Expect(economy.PerSecond.Metal.Sent).To(BeNumerically("~", 0.5))
			Expect(economy.PerSecond.Energy.Received).To(BeNumerically("~", 1.5))
		})

		It("leaves the amounts per second at 0 for a game of 0 s", func() {
			replay := newTestDemo()
			replay.FileHeader.GameTime = 0
			replay.TeamStats = [][]demo.TeamStatistics{
				{{Frame: 450, MetalProduced: 1}},
				{{Frame: 450, MetalProduced: 1}},
			}

			result, err := report.NewReport(replay, newTestSetupScript(), newTestEventsFile())

			Expect(err).NotTo(HaveOccurred())
			Expect(result.Players[0].Economy.Total.Metal.Produced).To(BeNumerically("==", 1))
			Expect(result.Players[0].Economy.PerSecond).To(BeZero())
			Expect(result.Players[0].Economy.Intervals).To(BeEmpty())
		})

		It("reports no economy for a player without statistics", func() {
			replay := newTestDemo()
			replay.TeamStats[1] = nil

			result, err := report.NewReport(replay, newTestSetupScript(), newTestEventsFile())

			Expect(err).NotTo(HaveOccurred())
			Expect(result.Players[0].Economy).NotTo(BeNil())
			Expect(result.Players[1].Economy).To(BeNil())
		})
	})

	Context("history", func() {
		It("takes the history of the events file if the replay holds only the totals", func() {
			replay := newTestDemo()
			replay.TeamStats = [][]demo.TeamStatistics{
				{{Frame: 1950, MetalProduced: 420}},
				{{Frame: 1950, MetalProduced: 500}},
			}
			eventsFile := newTestEventsFile()
			eventsFile.TeamStats = map[int][]events.TeamStats{
				0: {
					{Frame: 0},
					{Frame: 900, MetalProduced: 60, EnergyExcess: 30},
					{Frame: 1800, MetalProduced: 420, EnergyExcess: 240},
				},
			}

			result, err := report.NewReport(
				replay,
				newTestSetupScript(),
				eventsFile,
				report.WithOpeningLength(0),
				report.WithIntervalLength(30*time.Second),
			)

			Expect(err).NotTo(HaveOccurred())
			economy := result.Players[0].Economy
			Expect(economy.Total.Metal.Produced).To(BeNumerically("==", 420))
			Expect(economy.Intervals).To(Equal([]report.EconomyInterval{
				{
					Time: report.Interval{Start: 0, End: report.Duration(30 * time.Second)},
					PerSecond: report.Resources{
						Metal:  report.Resource{Produced: 2},
						Energy: report.Resource{Wasted: 1},
					},
					MetalExtractors: report.MetalExtractors{Tech1: 1},
					Builders:        report.BuilderClasses{Commander: report.BuilderUse{Alive: 1}},
				},
				{
					Time: report.Interval{Start: report.Duration(30 * time.Second), End: report.Duration(60 * time.Second)},
					PerSecond: report.Resources{
						Metal:  report.Resource{Produced: 12},
						Energy: report.Resource{Wasted: 7},
					},
					MetalExtractors: report.MetalExtractors{Tech1: 1},
					Builders:        report.BuilderClasses{Commander: report.BuilderUse{Alive: 1}},
				},
			}))
			Expect(result.Players[1].Economy.Total.Metal.Produced).To(BeNumerically("==", 500))
			Expect(result.Players[1].Economy.Intervals).To(BeEmpty())
		})

		It("takes the history of the events file if the replay has no statistics", func() {
			replay := newTestDemo()
			replay.TeamStats = nil
			eventsFile := newTestEventsFile()
			eventsFile.TeamStats = map[int][]events.TeamStats{
				1: {{Frame: 0}, {Frame: 1800, MetalProduced: 600}},
			}

			result, err := report.NewReport(replay, newTestSetupScript(), eventsFile)

			Expect(err).NotTo(HaveOccurred())
			Expect(result.Players[0].Economy).To(BeNil())
			Expect(result.Players[1].Economy.PerSecond.Metal.Produced).To(BeNumerically("==", 10))
		})

		It("keeps the history of the replay if the events file has no more entries", func() {
			eventsFile := newTestEventsFile()
			eventsFile.TeamStats = map[int][]events.TeamStats{
				0: {{Frame: 0}, {Frame: 900, MetalProduced: 1}, {Frame: 1800, MetalProduced: 2}},
			}

			result, err := report.NewReport(newTestDemo(), newTestSetupScript(), eventsFile)

			Expect(err).NotTo(HaveOccurred())
			Expect(result.Players[0].Economy.Total.Metal.Produced).To(BeNumerically("==", 420))
		})
	})

	Context("Intervals", func() {
		It("aggregates the opening in 1 minute intervals for 10 minutes and the rest in 5 minute ones by default", func() {
			replay := newTestDemo()
			replay.FileHeader.GameTime = 21 * 60
			replay.TeamStats[0] = metalHistory(periodTimes(21 * 60)...)

			result, err := report.NewReport(replay, newTestSetupScript(), newTestEventsFile())

			Expect(err).NotTo(HaveOccurred())
			Expect(intervalTimes(result.Players[0].Economy.Intervals)).To(Equal([]string{
				"0:00-1:00", "1:00-2:00", "2:00-3:00", "3:00-4:00", "4:00-5:00",
				"5:00-6:00", "6:00-7:00", "7:00-8:00", "8:00-9:00", "9:00-10:00",
				"10:00-15:00", "15:00-20:00", "20:00-21:00",
			}))
			Expect(result.Players[0].Economy.Intervals[10].PerSecond.Metal.Produced).To(BeNumerically("==", 1))
		})

		It("ends the last opening interval with the opening", func() {
			replay := newTestDemo()
			replay.FileHeader.GameTime = 120
			replay.TeamStats[0] = metalHistory(periodTimes(120)...)

			result, err := report.NewReport(replay, newTestSetupScript(), newTestEventsFile(),
				report.WithOpeningLength(time.Minute),
				report.WithOpeningIntervalLength(45*time.Second),
				report.WithIntervalLength(30*time.Second),
			)

			Expect(err).NotTo(HaveOccurred())
			Expect(intervalTimes(result.Players[0].Economy.Intervals)).To(Equal([]string{
				"0:00-0:45", "0:45-1:00", "1:00-1:30", "1:30-2:00",
			}))
		})

		It("puts the whole game into one interval if it is shorter than the interval length", func() {
			result, err := report.NewReport(
				newTestDemo(),
				newTestSetupScript(),
				newTestEventsFile(),
				report.WithOpeningLength(0),
			)

			Expect(err).NotTo(HaveOccurred())
			Expect(result.Players[0].Economy.Intervals).To(Equal([]report.EconomyInterval{{
				Time:            report.Interval{Start: 0, End: report.Duration(60 * time.Second)},
				PerSecond:       result.Players[0].Economy.PerSecond,
				MetalExtractors: report.MetalExtractors{Tech1: 1},
				Builders:        report.BuilderClasses{Commander: report.BuilderUse{Alive: 1}},
			}}))
		})

		It("averages each interval and ends the last one with the game", func() {
			replay := newTestDemo()
			replay.TeamStats[0] = []demo.TeamStatistics{
				{Frame: 0},
				{Frame: 1350, MetalProduced: 90, MetalUsed: 45, EnergyProduced: 900, EnergyExcess: 180},
				{Frame: 1800, MetalProduced: 420, MetalUsed: 360, EnergyProduced: 2400, EnergyExcess: 240},
			}

			result, err := report.NewReport(replay, newTestSetupScript(), newTestEventsFile(),
				report.WithOpeningLength(0),
				report.WithIntervalLength(45*time.Second),
			)

			Expect(err).NotTo(HaveOccurred())
			Expect(result.Players[0].Economy.Intervals).To(Equal([]report.EconomyInterval{
				{
					Time: report.Interval{Start: 0, End: report.Duration(45 * time.Second)},
					PerSecond: report.Resources{
						Metal:  report.Resource{Produced: 2, Used: 1},
						Energy: report.Resource{Produced: 20, Wasted: 4},
					},
					MetalExtractors: report.MetalExtractors{Tech1: 1},
					Builders:        report.BuilderClasses{Commander: report.BuilderUse{Alive: 1}},
				},
				{
					Time: report.Interval{Start: report.Duration(45 * time.Second), End: report.Duration(60 * time.Second)},
					PerSecond: report.Resources{
						Metal:  report.Resource{Produced: 22, Used: 21},
						Energy: report.Resource{Produced: 100, Wasted: 4},
					},
					MetalExtractors: report.MetalExtractors{Tech1: 1},
					Builders:        report.BuilderClasses{Commander: report.BuilderUse{Alive: 1}},
				},
			}))
		})

		It("combines statistics periods into intervals and rounds the interval length up to whole periods", func() {
			replay := newTestDemo()
			replay.FileHeader.GameTime = 60
			replay.TeamStats[0] = metalHistory(0, 15, 30, 45, 60)

			result, err := report.NewReport(
				replay,
				newTestSetupScript(),
				newTestEventsFile(),
				report.WithOpeningLength(0),
				report.WithIntervalLength(20*time.Second),
			)

			Expect(err).NotTo(HaveOccurred())
			Expect(intervalTimes(result.Players[0].Economy.Intervals)).To(Equal([]string{"0:00-0:30", "0:30-1:00"}))
			Expect(result.Players[0].Economy.Intervals[1].PerSecond.Metal.Produced).To(BeNumerically("==", 1))
		})

		It("ends the last interval with the game, however short it is", func() {
			replay := newTestDemo()
			replay.FileHeader.GameTime = 63
			replay.TeamStats[0] = metalHistory(0, 15, 30, 45, 60, 75)

			result, err := report.NewReport(
				replay,
				newTestSetupScript(),
				newTestEventsFile(),
				report.WithOpeningLength(0),
				report.WithIntervalLength(30*time.Second),
			)

			Expect(err).NotTo(HaveOccurred())
			Expect(intervalTimes(result.Players[0].Economy.Intervals)).To(Equal([]string{"0:00-0:30", "0:30-1:00", "1:00-1:03"}))
		})

		It("keeps the totals of a game which ended at the frame of the entry before", func() {
			replay := newTestDemo()
			replay.FileHeader.GameTime = 50

			result, err := report.NewReport(
				replay,
				newTestSetupScript(),
				newTestEventsFile(),
				report.WithOpeningLength(0),
				report.WithIntervalLength(50*time.Second),
			)

			Expect(err).NotTo(HaveOccurred())
			economy := result.Players[0].Economy
			Expect(economy.Total.Metal.Produced).To(BeNumerically("==", 420))
			Expect(economy.PerSecond.Metal.Produced).To(BeNumerically("~", 8.4))
			Expect(intervalTimes(economy.Intervals)).To(Equal([]string{"0:00-0:50"}))
			Expect(economy.Intervals[0].PerSecond.Metal.Produced).To(BeNumerically("~", 8.4))
		})

		It("takes the latest of the entries recorded at the same frame", func() {
			replay := newTestDemo()
			replay.TeamStats[0] = []demo.TeamStatistics{
				{Frame: 0},
				{Frame: 450, MetalProduced: 10},
				{Frame: 450, MetalProduced: 15},
				{Frame: 1950, MetalProduced: 60},
			}

			result, err := report.NewReport(
				replay,
				newTestSetupScript(),
				newTestEventsFile(),
				report.WithOpeningLength(0),
				report.WithIntervalLength(15*time.Second),
			)

			Expect(err).NotTo(HaveOccurred())
			intervals := result.Players[0].Economy.Intervals
			Expect(intervalTimes(intervals)).To(Equal([]string{"0:00-0:15", "0:15-0:30", "0:30-0:45", "0:45-1:00"}))
			Expect(intervals[0].PerSecond.Metal.Produced).To(BeNumerically("~", 1))
		})

		It("reports no intervals if the history holds only the totals", func() {
			replay := newTestDemo()
			replay.TeamStats[0] = []demo.TeamStatistics{{Frame: 1950, MetalProduced: 60}}

			result, err := report.NewReport(replay, newTestSetupScript(), newTestEventsFile())

			Expect(err).NotTo(HaveOccurred())
			Expect(result.Players[0].Economy.Total.Metal.Produced).To(BeNumerically("==", 60))
			Expect(result.Players[0].Economy.Intervals).To(BeEmpty())
		})

		It("does not change the statistics of the demo", func() {
			replay := newTestDemo()

			Expect(report.NewReport(replay, newTestSetupScript(), newTestEventsFile())).Error().NotTo(HaveOccurred())
			Expect(replay.TeamStats).To(Equal(newTestDemo().TeamStats))
		})
	})
})

// metalHistory returns a statistics history with entries at the given seconds and a metal income of 1 per second.
func metalHistory(times ...int32) []demo.TeamStatistics {
	history := make([]demo.TeamStatistics, len(times))
	for i, second := range times {
		history[i] = demo.TeamStatistics{Frame: second * report.GameSpeed, MetalProduced: float32(second)}
	}
	return history
}

// periodTimes returns the seconds of the statistics entries of a game of gameTime seconds: every 15 s from 0 to
// gameTime.
func periodTimes(gameTime int32) []int32 {
	var times []int32
	for second := int32(0); second <= gameTime; second += 15 {
		times = append(times, second)
	}
	return times
}

// intervalTimes returns the time of each interval as start-end in minutes:seconds.
func intervalTimes(intervals []report.EconomyInterval) []string {
	result := make([]string, len(intervals))
	for i, interval := range intervals {
		result[i] = interval.Time.String()
	}
	return result
}
