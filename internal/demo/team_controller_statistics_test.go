package demo_test

import (
	"encoding/binary"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/backbone81/bar-replay-analytics-go/internal/demo"
)

var _ = Describe("TeamControllerStatistics", func() {
	It("has the size of the engine's TeamControllerStatistics", func() {
		Expect(binary.Size(demo.TeamControllerStatistics{})).To(Equal(8))
	})

	Context("ReadTeamControllerStatistics", func() {
		It("reads valid statistics", func() {
			stats := demo.TeamControllerStatistics{NumCommands: 364, UnitCommands: 864}

			data, err := encodeValue(stats)
			Expect(err).NotTo(HaveOccurred())

			result, err := demo.ReadTeamControllerStatistics(data)

			Expect(err).NotTo(HaveOccurred())
			Expect(result).To(Equal(stats))
		})

		It("rejects data shorter than the statistics", func() {
			Expect(demo.ReadTeamControllerStatistics(make([]byte, 7))).Error().To(
				MatchError(ContainSubstring("reading TeamControllerStatistics")))
		})

		DescribeTable("rejects a negative field",
			func(stats demo.TeamControllerStatistics, expectedError string) {
				data, err := encodeValue(stats)
				Expect(err).NotTo(HaveOccurred())

				Expect(demo.ReadTeamControllerStatistics(data)).Error().To(
					MatchError(ContainSubstring("invalid TeamControllerStatistics: " + expectedError)))
			},
			Entry("numCommands", demo.TeamControllerStatistics{NumCommands: -1}, "numCommands"),
			Entry("unitCommands", demo.TeamControllerStatistics{UnitCommands: -1}, "unitCommands"),
		)
	})
})
