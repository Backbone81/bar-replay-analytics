package report_test

import (
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/backbone81/bar-replay-analytics-go/internal/demo"
	"github.com/backbone81/bar-replay-analytics-go/internal/report"
)

var _ = Describe("Economy", func() {
	Context("Total and PerSecond", func() {
		It("takes the totals from the last entry and averages them over the game", func() {
			result, err := report.NewReport(newTestDemo())

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
			result, err := report.NewReport(newTestDemo())

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

			result, err := report.NewReport(replay)

			Expect(err).NotTo(HaveOccurred())
			Expect(result.Players[0].Economy.Total.Metal.Produced).To(BeNumerically("==", 1))
			Expect(result.Players[0].Economy.PerSecond).To(BeZero())
			Expect(result.Players[0].Economy.Intervals).To(BeEmpty())
		})

		It("reports no economy for a player without statistics", func() {
			replay := newTestDemo()
			replay.TeamStats[1] = nil

			result, err := report.NewReport(replay)

			Expect(err).NotTo(HaveOccurred())
			Expect(result.Players[0].Economy).NotTo(BeNil())
			Expect(result.Players[1].Economy).To(BeNil())
		})
	})

	Context("Intervals", func() {
		It("aggregates the opening in 1 minute intervals for 10 minutes and the rest in 5 minute ones by default", func() {
			replay := newTestDemo()
			replay.FileHeader.GameTime = 21 * 60
			replay.TeamStats[0] = metalHistory(periodTimes(21 * 60)...)

			result, err := report.NewReport(replay)

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

			result, err := report.NewReport(replay,
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
			result, err := report.NewReport(newTestDemo(), report.WithOpeningLength(0))

			Expect(err).NotTo(HaveOccurred())
			Expect(result.Players[0].Economy.Intervals).To(Equal([]report.EconomyInterval{{
				Time:      report.Interval{Start: 0, End: report.Duration(60 * time.Second)},
				PerSecond: result.Players[0].Economy.PerSecond,
			}}))
		})

		It("averages each interval and ends the last one with the game", func() {
			result, err := report.NewReport(newTestDemo(),
				report.WithOpeningLength(0),
				report.WithIntervalLength(50*time.Second),
			)

			Expect(err).NotTo(HaveOccurred())
			Expect(result.Players[0].Economy.Intervals).To(Equal([]report.EconomyInterval{
				{
					Time: report.Interval{Start: 0, End: report.Duration(50 * time.Second)},
					PerSecond: report.Resources{
						Metal:  report.Resource{Produced: 2, Used: 1},
						Energy: report.Resource{Produced: 20, Wasted: 4},
					},
				},
				{
					Time: report.Interval{Start: report.Duration(50 * time.Second), End: report.Duration(60 * time.Second)},
					PerSecond: report.Resources{
						Metal:  report.Resource{Produced: 32, Used: 31},
						Energy: report.Resource{Produced: 140, Wasted: 4},
					},
				},
			}))
		})

		It("combines statistics periods into intervals and rounds the interval length up to whole periods", func() {
			replay := newTestDemo()
			replay.FileHeader.GameTime = 60
			replay.TeamStats[0] = metalHistory(0, 15, 30, 45, 60)

			result, err := report.NewReport(replay, report.WithOpeningLength(0), report.WithIntervalLength(20*time.Second))

			Expect(err).NotTo(HaveOccurred())
			Expect(intervalTimes(result.Players[0].Economy.Intervals)).To(Equal([]string{"0:00-0:30", "0:30-1:00"}))
			Expect(result.Players[0].Economy.Intervals[1].PerSecond.Metal.Produced).To(BeNumerically("==", 1))
		})

		It("ends the last interval with the game, however short it is", func() {
			replay := newTestDemo()
			replay.FileHeader.GameTime = 63
			replay.TeamStats[0] = metalHistory(0, 15, 30, 45, 60, 75)

			result, err := report.NewReport(replay, report.WithOpeningLength(0), report.WithIntervalLength(30*time.Second))

			Expect(err).NotTo(HaveOccurred())
			Expect(intervalTimes(result.Players[0].Economy.Intervals)).To(Equal([]string{"0:00-0:30", "0:30-1:00", "1:00-1:03"}))
		})

		It("keeps the totals of a game which ended at the frame of the entry before", func() {
			replay := newTestDemo()
			replay.FileHeader.GameTime = 50

			result, err := report.NewReport(replay, report.WithOpeningLength(0), report.WithIntervalLength(50*time.Second))

			Expect(err).NotTo(HaveOccurred())
			economy := result.Players[0].Economy
			Expect(economy.Total.Metal.Produced).To(BeNumerically("==", 420))
			Expect(economy.PerSecond.Metal.Produced).To(BeNumerically("~", 8.4))
			Expect(intervalTimes(economy.Intervals)).To(Equal([]string{"0:00-0:50"}))
			Expect(economy.Intervals[0].PerSecond.Metal.Produced).To(BeNumerically("~", 8.4))
		})

		It("does not end the game before the entry before the totals", func() {
			replay := newTestDemo()
			replay.FileHeader.GameTime = 50
			replay.TeamStats[0] = []demo.TeamStatistics{
				{Frame: 0},
				{Frame: 1515, MetalProduced: 101},
				{Frame: 1965, MetalProduced: 202},
			}

			result, err := report.NewReport(replay, report.WithOpeningLength(0))

			Expect(err).NotTo(HaveOccurred())
			economy := result.Players[0].Economy
			Expect(economy.Intervals).To(HaveLen(1))
			Expect(economy.Intervals[0].Time.End).To(Equal(report.Duration(1515 * time.Second / report.GameSpeed)))
			Expect(economy.PerSecond.Metal.Produced).To(BeNumerically("~", 4))
		})

		It("skips entries of the same frame", func() {
			replay := newTestDemo()
			replay.TeamStats[0] = []demo.TeamStatistics{
				{Frame: 0},
				{Frame: 0, MetalProduced: 1},
				{Frame: 1500, MetalProduced: 31},
			}

			result, err := report.NewReport(replay, report.WithIntervalLength(time.Second))

			Expect(err).NotTo(HaveOccurred())
			Expect(intervalTimes(result.Players[0].Economy.Intervals)).To(Equal([]string{"0:00-0:50"}))
			Expect(result.Players[0].Economy.Intervals[0].PerSecond.Metal.Produced).To(BeNumerically("~", 0.62))
		})

		It("does not change the statistics of the demo", func() {
			replay := newTestDemo()

			Expect(report.NewReport(replay)).Error().NotTo(HaveOccurred())
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
