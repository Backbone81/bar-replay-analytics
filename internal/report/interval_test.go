package report_test

import (
	"encoding/json"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/backbone81/bar-replay-analytics/internal/report"
)

var _ = Describe("Interval", func() {
	Context("String", func() {
		DescribeTable("formats the interval as start-end in minutes:seconds",
			func(start time.Duration, end time.Duration, expected string) {
				interval := report.Interval{Start: report.Duration(start), End: report.Duration(end)}

				Expect(interval.String()).To(Equal(expected))
			},
			Entry("from the start of the game", time.Duration(0), 5*time.Minute, "0:00-5:00"),
			Entry("within the game", 15*time.Minute+23*time.Second, 16*time.Minute+20*time.Second, "15:23-16:20"),
			Entry("more than an hour, without hours", time.Hour, time.Hour+33*time.Second, "60:00-60:33"),
		)
	})

	Context("MarshalJSON", func() {
		It("writes the interval as a string", func() {
			interval := report.Interval{Start: report.Duration(time.Hour), End: report.Duration(time.Hour + 33*time.Second)}

			Expect(json.Marshal(interval)).To(MatchJSON(`"60:00-60:33"`))
		})

		It("writes the interval as a string inside a struct", func() {
			value := struct {
				Time report.Interval `json:"time"`
			}{
				Time: report.Interval{Start: 0, End: report.Duration(5 * time.Minute)},
			}

			Expect(json.Marshal(value)).To(MatchJSON(`{"time": "0:00-5:00"}`))
		})
	})
})
