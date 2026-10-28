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

			_, err = demo.Read(bytes.NewReader(data))

			Expect(err).To(MatchError(ContainSubstring("decompressing demo")))
		})

		It("rejects a gzip stream with a broken checksum", func() {
			compressed, err := newTestDemo().compressed()
			Expect(err).NotTo(HaveOccurred())
			compressed[len(compressed)-8] ^= 0xff // The CRC-32 is in the gzip trailer's first 4 of 8 bytes.

			_, err = demo.Read(bytes.NewReader(compressed))

			Expect(err).To(MatchError(gzip.ErrChecksum))
		})

		It("rejects a demo larger than MaxDecompressedSize", func() {
			compressed, err := compress(make([]byte, demo.MaxDecompressedSize+1))
			Expect(err).NotTo(HaveOccurred())

			_, err = demo.Read(bytes.NewReader(compressed))

			Expect(err).To(MatchError("reading demo: larger than 4096 bytes"))
		})

		DescribeTable("rejects an invalid demo",
			func(modify func(testDemo *testDemo), expectedError string) {
				testDemo := newTestDemo()
				modify(testDemo)

				compressed, err := testDemo.compressed()
				Expect(err).NotTo(HaveOccurred())

				_, err = demo.Read(bytes.NewReader(compressed))

				Expect(err).To(MatchError(ContainSubstring(expectedError)))
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
			_, err := demo.ReadUncompressed(bytes.NewReader(make([]byte, demo.MaxDecompressedSize+1)))
			Expect(err).To(MatchError("reading demo: larger than 4096 bytes"))
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
			_, err := demo.ReadFile(filepath.Join(GinkgoT().TempDir(), "missing.sdfz"))

			Expect(err).To(MatchError(fs.ErrNotExist))
		})

		It("returns the error of an invalid demo file", func() {
			filePath := filepath.Join(GinkgoT().TempDir(), "invalid.sdfz")
			Expect(os.WriteFile(filePath, []byte("not a demo"), 0o600)).To(Succeed())

			_, err := demo.ReadFile(filePath)

			Expect(err).To(MatchError(ContainSubstring("decompressing demo")))
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
