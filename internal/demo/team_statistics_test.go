package demo_test

import (
	"encoding/binary"
	"math"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/backbone81/bar-replay-analytics-go/internal/demo"
)

var _ = Describe("TeamStatistics", func() {
	It("has the size of the engine's TeamStatistics", func() {
		Expect(binary.Size(demo.TeamStatistics{})).To(Equal(80))
	})

	Context("ReadTeamStatistics", func() {
		It("reads valid statistics", func() {
			stats := demo.TeamStatistics{
				Frame:            48600,
				MetalUsed:        48014.668,
				EnergyUsed:       944569.3,
				MetalProduced:    47485.953,
				EnergyProduced:   952862.06,
				MetalExcess:      95.41394,
				EnergyExcess:     8811.604,
				MetalReceived:    1,
				EnergyReceived:   2,
				MetalSent:        3,
				EnergySent:       4,
				DamageDealt:      178004.86,
				DamageReceived:   254112.19,
				UnitsProduced:    462,
				UnitsDied:        462,
				UnitsReceived:    5,
				UnitsSent:        6,
				UnitsCaptured:    7,
				UnitsOutCaptured: 8,
				UnitsKilled:      143,
			}

			data, err := encodeValue(stats)
			Expect(err).NotTo(HaveOccurred())

			result, err := demo.ReadTeamStatistics(data)

			Expect(err).NotTo(HaveOccurred())
			Expect(result).To(Equal(stats))
		})

		It("accepts negative resource and damage values", func() {
			stats := demo.TeamStatistics{MetalUsed: -1, DamageDealt: -1}

			data, err := encodeValue(stats)
			Expect(err).NotTo(HaveOccurred())

			Expect(demo.ReadTeamStatistics(data)).Error().NotTo(HaveOccurred())
		})

		It("rejects data shorter than the statistics", func() {
			Expect(demo.ReadTeamStatistics(make([]byte, 79))).Error().To(MatchError(ContainSubstring("reading TeamStatistics")))
		})

		DescribeTable("rejects an invalid field",
			func(modify func(stats *demo.TeamStatistics), expectedError string) {
				var stats demo.TeamStatistics
				modify(&stats)

				data, err := encodeValue(stats)
				Expect(err).NotTo(HaveOccurred())

				Expect(demo.ReadTeamStatistics(data)).Error().To(
					MatchError(ContainSubstring("invalid TeamStatistics: " + expectedError)))
			},
			Entry("negative frame", func(stats *demo.TeamStatistics) { stats.Frame = -1 }, "frame"),
			Entry("NaN metalUsed",
				func(stats *demo.TeamStatistics) { stats.MetalUsed = float32(math.NaN()) }, "metalUsed"),
			Entry("infinite energyUsed",
				func(stats *demo.TeamStatistics) { stats.EnergyUsed = float32(math.Inf(1)) }, "energyUsed"),
			Entry("NaN metalProduced",
				func(stats *demo.TeamStatistics) { stats.MetalProduced = float32(math.NaN()) }, "metalProduced"),
			Entry("NaN energyProduced",
				func(stats *demo.TeamStatistics) { stats.EnergyProduced = float32(math.NaN()) }, "energyProduced"),
			Entry("NaN metalExcess",
				func(stats *demo.TeamStatistics) { stats.MetalExcess = float32(math.NaN()) }, "metalExcess"),
			Entry("NaN energyExcess",
				func(stats *demo.TeamStatistics) { stats.EnergyExcess = float32(math.NaN()) }, "energyExcess"),
			Entry("NaN metalReceived",
				func(stats *demo.TeamStatistics) { stats.MetalReceived = float32(math.NaN()) }, "metalReceived"),
			Entry("NaN energyReceived",
				func(stats *demo.TeamStatistics) { stats.EnergyReceived = float32(math.NaN()) }, "energyReceived"),
			Entry("NaN metalSent",
				func(stats *demo.TeamStatistics) { stats.MetalSent = float32(math.NaN()) }, "metalSent"),
			Entry("NaN energySent",
				func(stats *demo.TeamStatistics) { stats.EnergySent = float32(math.NaN()) }, "energySent"),
			Entry("NaN damageDealt",
				func(stats *demo.TeamStatistics) { stats.DamageDealt = float32(math.NaN()) }, "damageDealt"),
			Entry("negative infinite damageReceived",
				func(stats *demo.TeamStatistics) { stats.DamageReceived = float32(math.Inf(-1)) }, "damageReceived"),
			Entry("negative unitsProduced",
				func(stats *demo.TeamStatistics) { stats.UnitsProduced = -1 }, "unitsProduced"),
			Entry("negative unitsDied", func(stats *demo.TeamStatistics) { stats.UnitsDied = -1 }, "unitsDied"),
			Entry("negative unitsReceived",
				func(stats *demo.TeamStatistics) { stats.UnitsReceived = -1 }, "unitsReceived"),
			Entry("negative unitsSent", func(stats *demo.TeamStatistics) { stats.UnitsSent = -1 }, "unitsSent"),
			Entry("negative unitsCaptured",
				func(stats *demo.TeamStatistics) { stats.UnitsCaptured = -1 }, "unitsCaptured"),
			Entry("negative unitsOutCaptured",
				func(stats *demo.TeamStatistics) { stats.UnitsOutCaptured = -1 }, "unitsOutCaptured"),
			Entry("negative unitsKilled",
				func(stats *demo.TeamStatistics) { stats.UnitsKilled = -1 }, "unitsKilled"),
		)
	})
})
