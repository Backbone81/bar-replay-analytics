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
				Metal:  report.Resource{Produced: 400, Used: 350},
				Energy: report.Resource{Produced: 2500, Wasted: 200},
			}))
			Expect(economy.PerSecond).To(Equal(report.Resources{
				Metal:  report.Resource{Produced: 4, Used: 3.5},
				Energy: report.Resource{Produced: 25, Wasted: 2},
			}))
		})

		It("includes what was shared with allies", func() {
			result, err := report.NewReport(newTestDemo())

			Expect(err).NotTo(HaveOccurred())
			economy := result.Players[1].Economy
			Expect(economy.Total.Metal.Sent).To(BeNumerically("==", 40))
			Expect(economy.Total.Energy.Received).To(BeNumerically("==", 90))
			Expect(economy.PerSecond.Metal.Sent).To(BeNumerically("~", 0.4))
			Expect(economy.PerSecond.Energy.Received).To(BeNumerically("~", 0.9))
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
		It("puts the whole game into one interval by default if it is shorter than the interval length", func() {
			result, err := report.NewReport(newTestDemo())

			Expect(err).NotTo(HaveOccurred())
			Expect(result.Players[0].Economy.Intervals).To(Equal([]report.EconomyInterval{{
				Time:      report.Interval{Start: 0, End: report.Duration(100 * time.Second)},
				PerSecond: result.Players[0].Economy.PerSecond,
			}}))
		})

		It("averages each interval and ends the last one with the game", func() {
			result, err := report.NewReport(newTestDemo(), report.WithIntervalLength(50*time.Second))

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
					Time: report.Interval{Start: report.Duration(50 * time.Second), End: report.Duration(100 * time.Second)},
					PerSecond: report.Resources{
						Metal:  report.Resource{Produced: 6, Used: 6},
						Energy: report.Resource{Produced: 30},
					},
				},
			}))
		})

		It("combines statistics periods into intervals and rounds the interval length up to whole periods", func() {
			replay := newTestDemo()
			replay.FileHeader.GameTime = 60
			replay.TeamStats[0] = metalHistory(0, 15, 30, 45, 60)

			result, err := report.NewReport(replay, report.WithIntervalLength(20*time.Second))

			Expect(err).NotTo(HaveOccurred())
			Expect(intervalTimes(result.Players[0].Economy.Intervals)).To(Equal([]string{"0:00-0:30", "0:30-1:00"}))
			Expect(result.Players[0].Economy.Intervals[1].PerSecond.Metal.Produced).To(BeNumerically("==", 1))
		})

		It("ends the last interval with the game, however short it is", func() {
			replay := newTestDemo()
			replay.FileHeader.GameTime = 63
			replay.TeamStats[0] = metalHistory(0, 15, 30, 45, 60, 75)

			result, err := report.NewReport(replay, report.WithIntervalLength(30*time.Second))

			Expect(err).NotTo(HaveOccurred())
			Expect(intervalTimes(result.Players[0].Economy.Intervals)).To(Equal([]string{"0:00-0:30", "0:30-1:00", "1:00-1:03"}))
		})

		It("drops totals which did not change since the entry before", func() {
			replay := newTestDemo()
			replay.TeamStats[0][2] = replay.TeamStats[0][1]
			replay.TeamStats[0][2].Frame = 4500

			result, err := report.NewReport(replay)

			Expect(err).NotTo(HaveOccurred())
			economy := result.Players[0].Economy
			Expect(intervalTimes(economy.Intervals)).To(Equal([]string{"0:00-0:50"}))
			Expect(economy.Total.Metal.Produced).To(BeNumerically("==", 100))
			Expect(economy.PerSecond.Metal.Produced).To(BeNumerically("==", 2))
		})

		It("keeps the totals of a game which ended at the frame of the entry before", func() {
			replay := newTestDemo()
			replay.FileHeader.GameTime = 50

			result, err := report.NewReport(replay)

			Expect(err).NotTo(HaveOccurred())
			economy := result.Players[0].Economy
			Expect(economy.Total.Metal.Produced).To(BeNumerically("==", 400))
			Expect(intervalTimes(economy.Intervals)).To(Equal([]string{"0:00-0:50"}))
			Expect(economy.Intervals[0].PerSecond.Metal.Produced).To(BeNumerically("==", 8))
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

		DescribeTable("rejects an interval length which is not positive",
			func(intervalLength time.Duration) {
				Expect(report.NewReport(newTestDemo(), report.WithIntervalLength(intervalLength))).Error().To(HaveOccurred())
			},
			Entry("zero", time.Duration(0)),
			Entry("negative", -time.Minute),
		)

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

// intervalTimes returns the time of each interval as start-end in minutes:seconds.
func intervalTimes(intervals []report.EconomyInterval) []string {
	result := make([]string, len(intervals))
	for i, interval := range intervals {
		result[i] = interval.Time.String()
	}
	return result
}
