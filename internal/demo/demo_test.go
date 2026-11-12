package demo_test

import (
	"bytes"
	"compress/gzip"
	"encoding/binary"
	"encoding/json"
	"io/fs"
	"math"
	"os"
	"path/filepath"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/backbone81/bar-replay-analytics-go/internal/demo"
)

var _ = Describe("Demo", func() {
	// A low limit keeps the specs about it small. The test demos are far below it.
	var maxDecompressedSizeBackup int64

	BeforeEach(func() {
		maxDecompressedSizeBackup = demo.MaxDecompressedSize
		demo.MaxDecompressedSize = 4096
	})

	AfterEach(func() {
		demo.MaxDecompressedSize = maxDecompressedSizeBackup
	})

	Context("Read", func() {
		It("reads every section of a demo", func() {
			testDemo := newTestDemo()

			compressed, err := testDemo.compressed()
			Expect(err).NotTo(HaveOccurred())

			result, err := demo.Read(bytes.NewReader(compressed))

			Expect(err).NotTo(HaveOccurred())
			Expect(result.FileHeader).To(Equal(testDemo.header))
			Expect(result.SetupScript).To(Equal(testDemo.setupScript))
			Expect(result.WinningAllyTeams).To(Equal(testDemo.winningAllyTeams))
			Expect(result.PlayerStats).To(Equal(testDemo.playerStats))
			Expect(result.TeamStats).To(Equal(testDemo.teamStats))
		})

		It("reads a demo without players, teams and winners", func() {
			testDemo := newTestDemo()
			testDemo.winningAllyTeams = nil
			testDemo.playerStats = nil
			testDemo.teamStats = nil
			testDemo.updateSizes()

			compressed, err := testDemo.compressed()
			Expect(err).NotTo(HaveOccurred())

			result, err := demo.Read(bytes.NewReader(compressed))

			Expect(err).NotTo(HaveOccurred())
			Expect(result.WinningAllyTeams).To(BeEmpty())
			Expect(result.PlayerStats).To(BeEmpty())
			Expect(result.TeamStats).To(BeEmpty())
		})

		It("reads winning allyteams listed twice or beyond numTeams", func() {
			testDemo := newTestDemo()
			testDemo.winningAllyTeams = []uint8{2, 2}
			testDemo.updateSizes()

			compressed, err := testDemo.compressed()
			Expect(err).NotTo(HaveOccurred())

			result, err := demo.Read(bytes.NewReader(compressed))

			Expect(err).NotTo(HaveOccurred())
			Expect(result.WinningAllyTeams).To(Equal([]uint8{2, 2}))
		})

		It("reads a team without statistics entries", func() {
			testDemo := newTestDemo()
			testDemo.teamStats[1] = nil
			testDemo.updateSizes()

			compressed, err := testDemo.compressed()
			Expect(err).NotTo(HaveOccurred())

			result, err := demo.Read(bytes.NewReader(compressed))

			Expect(err).NotTo(HaveOccurred())
			Expect(result.TeamStats).To(HaveLen(2))
			Expect(result.TeamStats[0]).To(Equal(testDemo.teamStats[0]))
			Expect(result.TeamStats[1]).To(BeEmpty())
		})

		It("skips header fields appended by a newer engine", func() {
			testDemo := newTestDemo()
			testDemo.headerExtension = []byte{1, 2, 3, 4}
			testDemo.updateSizes()

			compressed, err := testDemo.compressed()
			Expect(err).NotTo(HaveOccurred())

			result, err := demo.Read(bytes.NewReader(compressed))

			Expect(err).NotTo(HaveOccurred())
			Expect(result.FileHeader.HeaderSize).To(BeEquivalentTo(binary.Size(demo.DemoFileHeader{}) + 4))
			Expect(result.SetupScript).To(Equal(testDemo.setupScript))
			Expect(result.TeamStats).To(Equal(testDemo.teamStats))
		})

		It("reads only header and setup script if the engine crashed while recording", func() {
			testDemo := newTestDemo()
			testDemo.winningAllyTeams = nil
			testDemo.playerStats = nil
			testDemo.teamStats = nil
			testDemo.updateSizes()
			testDemo.header.DemoStreamSize = 0

			compressed, err := testDemo.compressed()
			Expect(err).NotTo(HaveOccurred())

			result, err := demo.Read(bytes.NewReader(compressed))

			Expect(err).NotTo(HaveOccurred())
			Expect(result.FileHeader).To(Equal(testDemo.header))
			Expect(result.SetupScript).To(Equal(testDemo.setupScript))
			Expect(result.WinningAllyTeams).To(BeNil())
			Expect(result.PlayerStats).To(BeNil())
			Expect(result.TeamStats).To(BeNil())
		})

		It("rejects data that is not gzip-compressed", func() {
			data, err := newTestDemo().encode()
			Expect(err).NotTo(HaveOccurred())

			Expect(demo.Read(bytes.NewReader(data))).Error().To(MatchError(ContainSubstring("decompressing demo")))
		})

		It("rejects a gzip stream with a broken checksum", func() {
			compressed, err := newTestDemo().compressed()
			Expect(err).NotTo(HaveOccurred())
			compressed[len(compressed)-8] ^= 0xff // The CRC-32 is in the gzip trailer's first 4 of 8 bytes.

			Expect(demo.Read(bytes.NewReader(compressed))).Error().To(MatchError(gzip.ErrChecksum))
		})

		It("rejects a demo larger than MaxDecompressedSize", func() {
			compressed, err := compress(make([]byte, demo.MaxDecompressedSize+1))
			Expect(err).NotTo(HaveOccurred())

			Expect(demo.Read(bytes.NewReader(compressed))).Error().To(MatchError("reading demo: larger than 4096 bytes"))
		})

		DescribeTable("rejects an invalid demo",
			func(modify func(testDemo *testDemo), expectedError string) {
				testDemo := newTestDemo()
				modify(testDemo)

				compressed, err := testDemo.compressed()
				Expect(err).NotTo(HaveOccurred())

				Expect(demo.Read(bytes.NewReader(compressed))).Error().To(MatchError(ContainSubstring(expectedError)))
			},
			Entry("with an invalid header",
				func(testDemo *testDemo) { testDemo.header.Version = 4 },
				"invalid DemoFileHeader: version"),
			Entry("with a header size beyond the end of the file",
				func(testDemo *testDemo) { testDemo.header.HeaderSize = math.MaxInt32 },
				"DemoFileHeader: 2147483647 bytes"),
			Entry("with a setup script beyond the end of the file",
				func(testDemo *testDemo) { testDemo.header.ScriptSize = math.MaxInt32 },
				"setup script: 2147483647 bytes"),
			Entry("with a NUL byte in the setup script",
				func(testDemo *testDemo) {
					testDemo.setupScript = "[game]\x00{}"
					testDemo.updateSizes()
				},
				"setup script: NUL byte at offset 6"),
			Entry("with a setup script that is not UTF-8",
				func(testDemo *testDemo) {
					testDemo.setupScript = "[game]\xff{}"
					testDemo.updateSizes()
				},
				"setup script: not valid UTF-8"),
			Entry("with a demo stream beyond the end of the file",
				func(testDemo *testDemo) { testDemo.header.DemoStreamSize = math.MaxInt32 },
				"demo stream: 2147483647 bytes"),
			Entry("with winning allyteams beyond the end of the file",
				func(testDemo *testDemo) {
					testDemo.playerStats = nil
					testDemo.teamStats = nil
					testDemo.updateSizes()
					testDemo.header.WinningAllyTeamsSize = 2
				},
				"winning allyteams: 2 bytes, but only 1 bytes left"),
			Entry("with player statistics beyond the end of the file",
				func(testDemo *testDemo) {
					testDemo.teamStats = nil
					testDemo.updateSizes()
					testDemo.winningAllyTeams = nil
					testDemo.header.WinningAllyTeamsSize = 0
					testDemo.header.NumPlayers = 2
					testDemo.header.PlayerStatSize = 2 * testDemo.header.PlayerStatElemSize
				},
				"player statistics: 40 bytes, but only 20 bytes left"),
			Entry("with invalid player statistics",
				func(testDemo *testDemo) { testDemo.playerStats[0].KeyPresses = -1 },
				"player 0: invalid PlayerStatistics: keyPresses"),
			Entry("with team statistics beyond the end of the file",
				func(testDemo *testDemo) { testDemo.header.TeamStatSize++ },
				"team statistics: 329 bytes, but only 328 bytes left"),
			Entry("with team statistics too small for the team counts",
				func(testDemo *testDemo) { testDemo.header.TeamStatSize = 4 },
				"team statistics: 4 bytes, too small for 2 team counts"),
			Entry("with team counts that do not add up to the team statistics size",
				func(testDemo *testDemo) {
					testDemo.teamStats[1] = testDemo.teamStats[1][:1]
					testDemo.header.TeamStatSize -= testDemo.header.TeamStatElemSize - 1
					testDemo.trailingData = make([]byte, testDemo.header.TeamStatElemSize-1)
				},
				"teamStatSize is 249, but the team counts add up to 248 bytes"),
			Entry("with invalid team statistics",
				func(testDemo *testDemo) { testDemo.teamStats[1][1].DamageDealt = float32(math.NaN()) },
				"team 1 entry 1: invalid TeamStatistics: damageDealt"),
			Entry("with a team statistics entry before the previous one",
				func(testDemo *testDemo) { testDemo.teamStats[0][0].Frame = 900 },
				"team 0 entry 1: frame 450 is before the previous entry's frame 900"),
			Entry("with data after the team statistics",
				func(testDemo *testDemo) { testDemo.trailingData = []byte{0} },
				"1 unexpected bytes after the team statistics"),
		)
	})

	Context("byte-swapped player statistics", func() {
		It("reads little-endian player statistics as they are", func() {
			result, err := newTestDemo().read()

			Expect(err).NotTo(HaveOccurred())
			Expect(result.PlayerStatsByteSwapped).To(BeFalse())
		})

		It("reads big-endian player statistics of an engine before the fix", func() {
			testDemo := newTestDemo()
			testDemo.playerStatsByteOrder = binary.BigEndian

			result, err := testDemo.read()

			Expect(err).NotTo(HaveOccurred())
			Expect(result.PlayerStats).To(Equal(testDemo.playerStats))
			Expect(result.PlayerStatsByteSwapped).To(BeTrue())
			Expect(result.TeamStats).To(Equal(testDemo.teamStats))
		})

		It("reads the statistics of all players in one byte order", func() {
			testDemo := newTestDemo()
			// A spectator's statistics are all 0 in either byte order.
			testDemo.playerStats = append(testDemo.playerStats, demo.PlayerStatistics{})
			testDemo.updateSizes()
			testDemo.playerStatsByteOrder = binary.BigEndian

			result, err := testDemo.read()

			Expect(err).NotTo(HaveOccurred())
			Expect(result.PlayerStats).To(Equal(testDemo.playerStats))
			Expect(result.PlayerStatsByteSwapped).To(BeTrue())
		})

		It("writes whether it swapped the bytes to JSON", func() {
			testDemo := newTestDemo()
			testDemo.playerStatsByteOrder = binary.BigEndian
			result, err := testDemo.read()
			Expect(err).NotTo(HaveOccurred())

			Expect(json.Marshal(result)).To(ContainSubstring(`"playerStatsByteSwapped":true`))
		})

		DescribeTable("reads big-endian player statistics only of an engine before the fix",
			func(version string, beforeFix bool) {
				testDemo := newTestDemo()
				testDemo.setEngineVersion(version)
				testDemo.playerStatsByteOrder = binary.BigEndian

				result, err := testDemo.read()

				if beforeFix {
					Expect(err).NotTo(HaveOccurred())
					Expect(result.PlayerStats).To(Equal(testDemo.playerStats))
					Expect(result.PlayerStatsByteSwapped).To(BeTrue())
				} else {
					Expect(err).To(MatchError(ContainSubstring("player 0: invalid PlayerStatistics")))
					Expect(err).NotTo(MatchError(ContainSubstring("big-endian")))
				}
			},
			Entry("an earlier release", "2026.07.04", true),
			Entry("an earlier development build", "2026.07.04-12-gabcdef0", true),
			Entry("an earlier month", "2026.08.99", true),
			Entry("an earlier year", "2025.12.31", true),
			Entry("a Spring version", "105.1.1-2590-gb9462a0 BAR105", true),
			Entry("a version with two numbers, the third counting as 0", "2026.9", true),
			Entry("the release with the fix", "2026.09.01", false),
			Entry("a development build of it", "2026.09.01-3-g1234567", false),
			Entry("a later release", "2026.09.02", false),
			Entry("a later year", "2027.01.01", false),
			Entry("a version which is not a number", "dev", false),
			Entry("a version with an empty number", "2026..01", false),
		)

		It("names both byte orders if both fail", func() {
			testDemo := newTestDemo()
			testDemo.playerStats[0].KeyPresses = -1

			_, err := testDemo.read()

			Expect(err).To(MatchError(MatchRegexp(
				`player 0: invalid PlayerStatistics: keyPresses.*; in big-endian, as engine 2026\.07\.04 may have ` +
					`written them: player 0: invalid PlayerStatistics`)))
		})
	})

	Context("ReadUncompressed", func() {
		It("reads every section of a decompressed demo", func() {
			testDemo := newTestDemo()

			data, err := testDemo.encode()
			Expect(err).NotTo(HaveOccurred())

			result, err := demo.ReadUncompressed(bytes.NewReader(data))

			Expect(err).NotTo(HaveOccurred())
			Expect(result.FileHeader).To(Equal(testDemo.header))
			Expect(result.SetupScript).To(Equal(testDemo.setupScript))
			Expect(result.WinningAllyTeams).To(Equal(testDemo.winningAllyTeams))
			Expect(result.PlayerStats).To(Equal(testDemo.playerStats))
			Expect(result.TeamStats).To(Equal(testDemo.teamStats))
		})

		It("rejects a gzip-compressed demo", func() {
			compressed, err := newTestDemo().compressed()
			Expect(err).NotTo(HaveOccurred())

			Expect(demo.ReadUncompressed(bytes.NewReader(compressed))).Error().To(HaveOccurred())
		})

		It("rejects a demo larger than MaxDecompressedSize", func() {
			Expect(demo.ReadUncompressed(bytes.NewReader(make([]byte, demo.MaxDecompressedSize+1)))).Error().To(
				MatchError("reading demo: larger than 4096 bytes"))
		})
	})

	Context("ReadFile", func() {
		It("reads a demo file", func() {
			testDemo := newTestDemo()
			compressed, err := testDemo.compressed()
			Expect(err).NotTo(HaveOccurred())
			filePath := filepath.Join(GinkgoT().TempDir(), "test.sdfz")
			Expect(os.WriteFile(filePath, compressed, 0o600)).To(Succeed())

			result, err := demo.ReadFile(filePath)

			Expect(err).NotTo(HaveOccurred())
			Expect(result.FileHeader).To(Equal(testDemo.header))
			Expect(result.TeamStats).To(Equal(testDemo.teamStats))
		})

		It("returns the error of a file that cannot be opened", func() {
			Expect(demo.ReadFile(filepath.Join(GinkgoT().TempDir(), "missing.sdfz"))).Error().To(MatchError(fs.ErrNotExist))
		})

		It("returns the error of an invalid demo file", func() {
			filePath := filepath.Join(GinkgoT().TempDir(), "invalid.sdfz")
			Expect(os.WriteFile(filePath, []byte("not a demo"), 0o600)).To(Succeed())

			Expect(demo.ReadFile(filePath)).Error().To(MatchError(ContainSubstring("decompressing demo")))
		})
	})

	Context("ReadFileHeader", func() {
		It("reads the header of a demo file", func() {
			testDemo := newTestDemo()
			compressed, err := testDemo.compressed()
			Expect(err).NotTo(HaveOccurred())

			header, err := demo.ReadDemoFileHeaderFromFilePath(writeTempFile(compressed))

			Expect(err).NotTo(HaveOccurred())
			Expect(header).To(Equal(testDemo.header))
			Expect(header.GameIDAsString()).To(Equal("0102030405060708090a0b0c0d0e0f10"))
		})

		It("reads the header of a demo which is cut off after it", func() {
			data, err := newTestDemo().encode()
			Expect(err).NotTo(HaveOccurred())
			compressed, err := compress(data[:binary.Size(demo.DemoFileHeader{})])
			Expect(err).NotTo(HaveOccurred())

			Expect(demo.ReadDemoFileHeaderFromFilePath(writeTempFile(compressed))).Error().NotTo(HaveOccurred())
		})

		It("rejects a demo shorter than the header", func() {
			data, err := newTestDemo().encode()
			Expect(err).NotTo(HaveOccurred())
			compressed, err := compress(data[:binary.Size(demo.DemoFileHeader{})-1])
			Expect(err).NotTo(HaveOccurred())

			Expect(demo.ReadDemoFileHeaderFromFilePath(writeTempFile(compressed))).Error().To(
				MatchError(ContainSubstring("reading DemoFileHeader")))
		})

		It("rejects an invalid header", func() {
			testDemo := newTestDemo()
			testDemo.header.Version = 4
			compressed, err := testDemo.compressed()
			Expect(err).NotTo(HaveOccurred())

			Expect(demo.ReadDemoFileHeaderFromFilePath(writeTempFile(compressed))).Error().To(
				MatchError(ContainSubstring("invalid DemoFileHeader: version")))
		})

		It("rejects data that is not gzip-compressed", func() {
			Expect(demo.ReadDemoFileHeaderFromFilePath(writeTempFile([]byte("not a demo")))).Error().To(
				MatchError(ContainSubstring("decompressing demo")))
		})

		It("returns the error of a file that cannot be opened", func() {
			Expect(demo.ReadDemoFileHeaderFromFilePath(filepath.Join(GinkgoT().TempDir(), "missing.sdfz"))).Error().To(
				MatchError(fs.ErrNotExist))
		})
	})

	Context("ReadFileHeaderAndSetupScript", func() {
		It("reads the header and the setup script, but no statistics", func() {
			testDemo := newTestDemo()
			compressed, err := testDemo.compressed()
			Expect(err).NotTo(HaveOccurred())

			result, err := demo.ReadFileHeaderAndSetupScript(writeTempFile(compressed))

			Expect(err).NotTo(HaveOccurred())
			Expect(result.FileHeader).To(Equal(testDemo.header))
			Expect(result.SetupScript).To(Equal(testDemo.setupScript))
			Expect(result.TeamStats).To(BeNil())
		})

		It("skips a header extension of a newer engine", func() {
			testDemo := newTestDemo()
			testDemo.headerExtension = []byte{1, 2, 3, 4}
			testDemo.updateSizes()
			compressed, err := testDemo.compressed()
			Expect(err).NotTo(HaveOccurred())

			result, err := demo.ReadFileHeaderAndSetupScript(writeTempFile(compressed))

			Expect(err).NotTo(HaveOccurred())
			Expect(result.SetupScript).To(Equal(testDemo.setupScript))
		})

		It("reads a demo which is cut off after the setup script", func() {
			testDemo := newTestDemo()
			data, err := testDemo.encode()
			Expect(err).NotTo(HaveOccurred())
			compressed, err := compress(data[:int(testDemo.header.HeaderSize)+len(testDemo.setupScript)])
			Expect(err).NotTo(HaveOccurred())

			Expect(demo.ReadFileHeaderAndSetupScript(writeTempFile(compressed))).Error().NotTo(HaveOccurred())
		})

		It("rejects a demo which is cut off in the setup script", func() {
			testDemo := newTestDemo()
			data, err := testDemo.encode()
			Expect(err).NotTo(HaveOccurred())
			compressed, err := compress(data[:int(testDemo.header.HeaderSize)+len(testDemo.setupScript)-1])
			Expect(err).NotTo(HaveOccurred())

			Expect(demo.ReadFileHeaderAndSetupScript(writeTempFile(compressed))).Error().To(
				MatchError(ContainSubstring("reading setup script")))
		})

		It("rejects a setup script larger than MaxDecompressedSize", func() {
			testDemo := newTestDemo()
			testDemo.header.ScriptSize = int32(demo.MaxDecompressedSize)
			compressed, err := testDemo.compressed()
			Expect(err).NotTo(HaveOccurred())

			Expect(demo.ReadFileHeaderAndSetupScript(writeTempFile(compressed))).Error().To(
				MatchError("reading demo: larger than 4096 bytes"))
		})

		It("rejects an invalid setup script", func() {
			testDemo := newTestDemo()
			testDemo.setupScript = "[game]\x00"
			testDemo.updateSizes()
			compressed, err := testDemo.compressed()
			Expect(err).NotTo(HaveOccurred())

			Expect(demo.ReadFileHeaderAndSetupScript(writeTempFile(compressed))).Error().To(
				MatchError(ContainSubstring("NUL byte")))
		})

		It("rejects an invalid header", func() {
			testDemo := newTestDemo()
			testDemo.header.Version = 4
			compressed, err := testDemo.compressed()
			Expect(err).NotTo(HaveOccurred())

			Expect(demo.ReadFileHeaderAndSetupScript(writeTempFile(compressed))).Error().To(
				MatchError(ContainSubstring("invalid DemoFileHeader: version")))
		})

		It("returns the error of a file that cannot be opened", func() {
			Expect(demo.ReadFileHeaderAndSetupScript(filepath.Join(GinkgoT().TempDir(), "missing.sdfz"))).Error().To(
				MatchError(fs.ErrNotExist))
		})
	})

	Context("MarshalJSON", func() {
		It("writes the winning allyteams as an array of numbers", func() {
			testDemo := newTestDemo()
			testDemo.winningAllyTeams = []uint8{0, 1}

			data, err := json.Marshal(demo.Demo{WinningAllyTeams: testDemo.winningAllyTeams})

			Expect(err).NotTo(HaveOccurred())
			Expect(data).To(ContainSubstring(`"winningAllyTeams":[0,1]`))
		})

		It("writes missing winning allyteams as null", func() {
			data, err := json.Marshal(demo.Demo{})

			Expect(err).NotTo(HaveOccurred())
			Expect(data).To(ContainSubstring(`"winningAllyTeams":null`))
		})
	})
})

// writeTempFile writes data to a new file and returns its path.
func writeTempFile(data []byte) string {
	filePath := filepath.Join(GinkgoT().TempDir(), "test.sdfz")
	Expect(os.WriteFile(filePath, data, 0o600)).To(Succeed())
	return filePath
}
