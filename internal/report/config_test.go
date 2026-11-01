package report_test

import (
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/backbone81/bar-replay-analytics-go/internal/report"
)

var _ = Describe("Config", func() {
	Context("Validate", func() {
		It("accepts the default configuration", func() {
			Expect(report.DefaultConfig.Validate()).To(Succeed())
		})

		DescribeTable("rejects invalid lengths",
			func(option report.Option) {
				config := report.DefaultConfig
				option(&config)

				Expect(config.Validate()).NotTo(Succeed())
			},
			Entry("zero interval length", report.WithIntervalLength(0)),
			Entry("negative interval length", report.WithIntervalLength(-time.Minute)),
			Entry("negative opening length", report.WithOpeningLength(-time.Minute)),
			Entry("zero opening interval length", report.WithOpeningIntervalLength(0)),
			Entry("negative opening interval length", report.WithOpeningIntervalLength(-time.Minute)),
		)

		It("accepts any opening interval length without an opening", func() {
			config := report.DefaultConfig
			config.OpeningLength = 0
			config.OpeningIntervalLength = 0

			Expect(config.Validate()).To(Succeed())
		})
	})
})
