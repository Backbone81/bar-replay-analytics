package demo_test

import (
	"encoding/binary"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/backbone81/bar-replay-analytics-go/internal/demo"
)

var _ = Describe("PlayerStatistics", func() {
	It("has the size of the engine's PlayerStatistics", func() {
		Expect(binary.Size(demo.PlayerStatistics{})).To(Equal(20))
	})

	Context("ReadPlayerStatistics", func() {
		It("reads valid statistics", func() {
			stats := newTestDemo().playerStats[0]

			data, err := encodeValue(stats)
			Expect(err).NotTo(HaveOccurred())

			result, err := demo.ReadPlayerStatistics(data)

			Expect(err).NotTo(HaveOccurred())
			Expect(result).To(Equal(stats))
		})

		It("rejects data shorter than the statistics", func() {
			_, err := demo.ReadPlayerStatistics(make([]byte, 19))

			Expect(err).To(MatchError(ContainSubstring("reading PlayerStatistics")))
		})

		DescribeTable("rejects a negative field",
			func(modify func(stats *demo.PlayerStatistics), expectedError string) {
				stats := newTestDemo().playerStats[0]
				modify(&stats)

				data, err := encodeValue(stats)
				Expect(err).NotTo(HaveOccurred())

				_, err = demo.ReadPlayerStatistics(data)

				Expect(err).To(MatchError(ContainSubstring("invalid PlayerStatistics: " + expectedError)))
			},
			Entry("numCommands", func(stats *demo.PlayerStatistics) { stats.NumCommands = -1 }, "numCommands"),
			Entry("unitCommands", func(stats *demo.PlayerStatistics) { stats.UnitCommands = -1 }, "unitCommands"),
			Entry("mousePixels", func(stats *demo.PlayerStatistics) { stats.MousePixels = -1 }, "mousePixels"),
			Entry("mouseClicks", func(stats *demo.PlayerStatistics) { stats.MouseClicks = -1 }, "mouseClicks"),
			Entry("keyPresses", func(stats *demo.PlayerStatistics) { stats.KeyPresses = -1 }, "keyPresses"),
		)
	})
})
