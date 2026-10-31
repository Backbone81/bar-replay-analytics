package report_test

import (
	"encoding/json"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/backbone81/bar-replay-analytics-go/internal/report"
)

var _ = Describe("Duration", func() {
	Context("String", func() {
		DescribeTable("formats the duration as minutes:seconds",
			func(duration time.Duration, expected string) {
				Expect(report.Duration(duration).String()).To(Equal(expected))
			},
			Entry("zero", time.Duration(0), "0:00"),
			Entry("seconds only", 5*time.Second, "0:05"),
			Entry("minutes and seconds", 14*time.Minute+7*time.Second, "14:07"),
			Entry("more than an hour, without hours", time.Hour+33*time.Second, "60:33"),
			Entry("rounded down below half a second", 59*time.Second+499*time.Millisecond, "0:59"),
			Entry("rounded up from half a second", 59*time.Second+500*time.Millisecond, "1:00"),
			Entry("negative", -(time.Minute+5*time.Second), "-1:05"),
		)
	})

	Context("MarshalJSON", func() {
		It("writes the duration as a string", func() {
			Expect(json.Marshal(report.Duration(time.Hour + 33*time.Second))).To(MatchJSON(`"60:33"`))
		})

		It("writes the duration as a string inside a struct", func() {
			value := struct {
				Start report.Duration `json:"start"`
			}{
				Start: report.Duration(90 * time.Second),
			}

			Expect(json.Marshal(value)).To(MatchJSON(`{"start": "1:30"}`))
		})
	})
})
