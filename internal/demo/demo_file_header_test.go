package demo_test

import (
	"encoding/binary"
	"math"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/backbone81/bar-replay-analytics-go/internal/demo"
)

var _ = Describe("DemoFileHeader", func() {
	It("has the size of the engine's DemoFileHeader", func() {
		Expect(binary.Size(demo.DemoFileHeader{})).To(Equal(352))
	})

	Context("ReadDemoFileHeader", func() {
		It("reads a valid header", func() {
			header := newTestDemo().header

			data, err := encodeValue(header)
			Expect(err).NotTo(HaveOccurred())

			result, err := demo.ReadDemoFileHeader(data)

			Expect(err).NotTo(HaveOccurred())
			Expect(result).To(Equal(header))
		})

		It("accepts a header larger than its own size", func() {
			header := newTestDemo().header
			header.HeaderSize += 4

			data, err := encodeValue(header)
			Expect(err).NotTo(HaveOccurred())

			_, err = demo.ReadDemoFileHeader(data)

			Expect(err).NotTo(HaveOccurred())
		})

		It("rejects data shorter than the header", func() {
			data, err := encodeValue(newTestDemo().header)
			Expect(err).NotTo(HaveOccurred())

			_, err = demo.ReadDemoFileHeader(data[:len(data)-1])

			Expect(err).To(MatchError(ContainSubstring("reading DemoFileHeader")))
		})

		DescribeTable("rejects an invalid field",
			func(modify func(header *demo.DemoFileHeader), expectedError string) {
				header := newTestDemo().header
				modify(&header)

				data, err := encodeValue(header)
				Expect(err).NotTo(HaveOccurred())

				_, err = demo.ReadDemoFileHeader(data)

				Expect(err).To(MatchError(ContainSubstring("invalid DemoFileHeader: " + expectedError)))
			},
			Entry("magic",
				func(header *demo.DemoFileHeader) { header.Magic[0] = 'x' },
				"magic"),
			Entry("magic without NUL",
				func(header *demo.DemoFileHeader) { header.Magic[15] = '!' },
				"magic"),
			Entry("version",
				func(header *demo.DemoFileHeader) { header.Version = 4 },
				"version: got 4, want 5"),
			Entry("headerSize smaller than the header",
				func(header *demo.DemoFileHeader) { header.HeaderSize = 348 },
				"headerSize: got 348, want >= 352"),
			Entry("negative headerSize",
				func(header *demo.DemoFileHeader) { header.HeaderSize = -1 },
				"headerSize: got -1, want >= 352"),
			Entry("versionString without NUL",
				func(header *demo.DemoFileHeader) {
					for i := range header.VersionString {
						header.VersionString[i] = 'a'
					}
				},
				"versionString: not NUL-terminated"),
			Entry("empty versionString",
				func(header *demo.DemoFileHeader) { header.VersionString = [256]byte{} },
				"versionString: empty"),
			Entry("versionString that is not UTF-8",
				func(header *demo.DemoFileHeader) { header.VersionString[0] = 0xff },
				"versionString: not valid UTF-8"),
			Entry("versionString with a control character",
				func(header *demo.DemoFileHeader) { header.VersionString[0] = '\n' },
				"versionString: contains non-printable character"),
			Entry("unixTime beyond int64",
				func(header *demo.DemoFileHeader) { header.UnixTime = math.MaxInt64 + 1 },
				"unixTime"),
			Entry("scriptSize 0",
				func(header *demo.DemoFileHeader) { header.ScriptSize = 0 },
				"scriptSize: got 0, want > 0"),
			Entry("negative demoStreamSize",
				func(header *demo.DemoFileHeader) { header.DemoStreamSize = -1 },
				"demoStreamSize: got -1, want >= 0"),
			Entry("negative gameTime",
				func(header *demo.DemoFileHeader) { header.GameTime = -1 },
				"gameTime: got -1, want >= 0"),
			Entry("negative wallclockTime",
				func(header *demo.DemoFileHeader) { header.WallclockTime = -1 },
				"wallclockTime: got -1, want >= 0"),
			Entry("negative numPlayers",
				func(header *demo.DemoFileHeader) { header.NumPlayers = -1 },
				"numPlayers: got -1"),
			Entry("numPlayers above MaxPlayers",
				func(header *demo.DemoFileHeader) { header.NumPlayers = demo.MaxPlayers + 1 },
				"numPlayers: got 252"),
			Entry("playerStatElemSize",
				func(header *demo.DemoFileHeader) { header.PlayerStatElemSize = 24 },
				"playerStatElemSize: got 24, want 20"),
			Entry("playerStatSize not matching numPlayers",
				func(header *demo.DemoFileHeader) { header.PlayerStatSize = 40 },
				"playerStatSize: got 40"),
			Entry("negative numTeams",
				func(header *demo.DemoFileHeader) { header.NumTeams = -1 },
				"numTeams: got -1"),
			Entry("numTeams above MaxTeams",
				func(header *demo.DemoFileHeader) { header.NumTeams = demo.MaxTeams + 1 },
				"numTeams: got 256"),
			Entry("negative teamStatSize",
				func(header *demo.DemoFileHeader) { header.TeamStatSize = -1 },
				"teamStatSize: got -1, want >= 0"),
			Entry("teamStatElemSize",
				func(header *demo.DemoFileHeader) { header.TeamStatElemSize = 84 },
				"teamStatElemSize: got 84, want 80"),
			Entry("teamStatPeriod",
				func(header *demo.DemoFileHeader) { header.TeamStatPeriod = 16 },
				"teamStatPeriod: got 16, want 15"),
			Entry("negative winningAllyTeamsSize",
				func(header *demo.DemoFileHeader) { header.WinningAllyTeamsSize = -1 },
				"winningAllyTeamsSize: got -1"),
			Entry("winningAllyTeamsSize above MaxTeams",
				func(header *demo.DemoFileHeader) { header.WinningAllyTeamsSize = demo.MaxTeams + 1 },
				"winningAllyTeamsSize: got 256"),
		)
	})

	Context("VersionStringAsString", func() {
		It("returns the version string without the NUL padding", func() {
			header := newTestDemo().header

			Expect(header.VersionStringAsString()).To(Equal("2026.07.04"))
		})
	})
})
