package downloader_test

import (
	"context"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/backbone81/bar-replay-analytics/internal/barlive"
	"github.com/backbone81/bar-replay-analytics/internal/barlive/barlivetest"
	replaydownloader2 "github.com/backbone81/bar-replay-analytics/internal/replay/downloader"
	"github.com/backbone81/bar-replay-analytics/internal/replay/index"
)

// The replays the specs use, with the file names of the storage.
const (
	duelID1   = "00000000000000000000000000000001"
	duelFile1 = "2026-11-01_10-00-00-000_Argent Strata 1.1_2026.07.04.sdfz"
	duelID2   = "00000000000000000000000000000002"
	duelFile2 = "2026-11-02_10-00-00-000_Devil's Postpiles 1.1.1_2026.07.04.sdfz"
	teamID    = "00000000000000000000000000000003"
	teamFile  = "2026-11-03_10-00-00-000_Argent Strata 1.1_2026.07.04.sdfz"
)

// indexFileName is the file of the index in the download directory.
const indexFileName = "index.json"

var _ = Describe("Downloader", func() {
	var fake *barlivetest.Fake

	// dir is the download directory, which does not exist before a spec creates it.
	var dir string

	BeforeEach(func() {
		fake = barlivetest.NewFake()
		DeferCleanup(fake.Close)
		fake.Leaderboards[barlive.GameTypeDuel] = []barlive.LeaderboardPlayer{
			{ID: 1, Name: "Alice", Rating: 50},
			{ID: 2, Name: "Bob", Rating: 40},
			{ID: 3, Name: "Carol", Rating: 30},
		}
		dir = filepath.Join(GinkgoT().TempDir(), "leaderboard")
	})

	Context("searching", func() {
		It("searches the duel replays of each top player in bar-db", func() {
			before := time.Now().UTC()
			Expect(runDownloader(fake, dir, replaydownloader2.WithDays(7))).To(Succeed())
			after := time.Now().UTC()

			searches := fake.RequestsTo("/replays")
			Expect(searches).To(HaveLen(3))
			for i, player := range []string{"Alice", "Bob", "Carol"} {
				query := searches[i].Query
				Expect(query["date"]).To(HaveLen(2))
				// The time is not injectable; a spec running over midnight may see either day.
				Expect(query["date"][0]).To(BeElementOf(
					before.AddDate(0, 0, -7).Format(time.DateOnly), after.AddDate(0, 0, -7).Format(time.DateOnly)))
				Expect(query["date"][1]).To(BeElementOf(
					before.AddDate(0, 0, 1).Format(time.DateOnly), after.AddDate(0, 0, 1).Format(time.DateOnly)))
				delete(query, "date")
				Expect(query).To(Equal(url.Values{
					"page":          {"1"},
					"limit":         {"100"},
					"preset":        {"duel"},
					"endedNormally": {"true"},
					"hasBots":       {"false"},
					"players":       {player},
				}))
			}
		})

		It("selects the maps in bar-db", func() {
			Expect(runDownloader(fake, dir,
				replaydownloader2.WithMap("Argent Strata 1.1"),
				replaydownloader2.WithMap("Cape_Violet_V1"),
			)).To(Succeed())

			Expect(fake.RequestsTo("/replays")[0].Query["maps"]).To(Equal([]string{"Argent Strata 1.1", "Cape_Violet_V1"}))
		})

		It("searches the replays of the given number of top players only", func() {
			Expect(runDownloader(fake, dir, replaydownloader2.WithPlayers(2))).To(Succeed())

			searches := fake.RequestsTo("/replays")
			Expect(searches).To(HaveLen(2))
			Expect(searches[0].Query.Get("players")).To(Equal("Alice"))
			Expect(searches[1].Query.Get("players")).To(Equal("Bob"))
		})

		It("searches the leaderboard of the game type", func() {
			fake.Leaderboards[barlive.GameTypeSmallTeam] = []barlive.LeaderboardPlayer{{ID: 4, Name: "Dave", Rating: 60}}

			Expect(runDownloader(fake, dir, replaydownloader2.WithGameType(barlive.GameTypeSmallTeam))).To(Succeed())

			searches := fake.RequestsTo("/replays")
			Expect(searches).To(HaveLen(1))
			Expect(searches[0].Query.Get("players")).To(Equal("Dave"))
			Expect(searches[0].Query.Get("preset")).To(Equal("team"))
		})

		It("requests the next page while the pages are full", func() {
			// 2v2 games, which the duel search drops, so the spec needs no file names for them.
			for i := range barlive.MaxReplaysLimit + 1 {
				id := fmt.Sprintf("%032x", 1000+i)
				fake.Replays["Alice"] = append(fake.Replays["Alice"], barlivetest.ReplaySummary(id, 2, 2))
			}

			Expect(runDownloader(fake, dir, replaydownloader2.WithPlayers(1))).To(Succeed())

			searches := fake.RequestsTo("/replays")
			Expect(searches).To(HaveLen(2))
			Expect(searches[1].Query.Get("page")).To(Equal("2"))
		})

		It("reports a game type the season has no leaderboard for", func() {
			fake.GameTypes = []barlive.GameType{barlive.GameTypeDuel}

			Expect(runDownloader(fake, dir, replaydownloader2.WithGameType(barlive.GameTypeFFA))).To(HaveOccurred())

			Expect(fake.RequestsTo("/replays")).To(BeEmpty())
		})
	})

	Context("downloading", func() {
		It("downloads the replays into the directory and records them in the index", func() {
			Expect(fake.AddReplay(duelID1, duelFile1, []int{1, 1}, "Alice")).To(Succeed())
			Expect(fake.AddReplay(duelID2, duelFile2, []int{1, 1}, "Bob")).To(Succeed())

			Expect(runDownloader(fake, dir)).To(Succeed())

			Expect(os.ReadFile(filepath.Join(dir, duelFile1))).To(Equal(fake.Demos[duelFile1]))
			Expect(os.ReadFile(filepath.Join(dir, duelFile2))).To(Equal(fake.Demos[duelFile2]))
			Expect(readIndex(dir)).To(MatchJSON(fmt.Sprintf(`{%q: %q, %q: %q}`, duelID1, duelFile1, duelID2, duelFile2)))
		})

		It("skips the replays whose allyteams do not have the shape of the game type", func() {
			Expect(fake.AddReplay(duelID1, duelFile1, []int{1, 1}, "Alice")).To(Succeed())
			Expect(fake.AddReplay(teamID, teamFile, []int{2, 2}, "Alice")).To(Succeed())

			Expect(runDownloader(fake, dir)).To(Succeed())

			Expect(fake.RequestsTo("/replays/" + teamID)).To(BeEmpty())
			Expect(filepath.Join(dir, teamFile)).NotTo(BeAnExistingFile())
			Expect(filepath.Join(dir, duelFile1)).To(BeAnExistingFile())
		})

		It("downloads a game of two top players once", func() {
			Expect(fake.AddReplay(duelID1, duelFile1, []int{1, 1}, "Alice", "Bob")).To(Succeed())

			Expect(runDownloader(fake, dir)).To(Succeed())

			Expect(fake.StorageRequests()).To(Equal(1))
			Expect(fake.RequestsTo("/replays/" + duelID1)).To(HaveLen(1))
		})

		It("sends no request for a replay in the index whose file exists", func() {
			Expect(fake.AddReplay(duelID1, duelFile1, []int{1, 1}, "Alice")).To(Succeed())
			Expect(runDownloader(fake, dir)).To(Succeed())
			requestsBefore := len(fake.RequestsTo("/replays/" + duelID1))

			Expect(runDownloader(fake, dir)).To(Succeed())

			Expect(fake.RequestsTo("/replays/" + duelID1)).To(HaveLen(requestsBefore))
			Expect(fake.StorageRequests()).To(Equal(1))
		})

		It("downloads a replay in the index again whose file was removed", func() {
			Expect(fake.AddReplay(duelID1, duelFile1, []int{1, 1}, "Alice")).To(Succeed())
			Expect(runDownloader(fake, dir)).To(Succeed())
			Expect(os.Remove(filepath.Join(dir, duelFile1))).To(Succeed())

			Expect(runDownloader(fake, dir)).To(Succeed())

			Expect(fake.StorageRequests()).To(Equal(2))
			Expect(filepath.Join(dir, duelFile1)).To(BeAnExistingFile())
			Expect(readIndex(dir)).To(MatchJSON(fmt.Sprintf(`{%q: %q}`, duelID1, duelFile1)))
		})

		It("records a file added by hand without downloading it", func() {
			Expect(fake.AddReplay(duelID1, duelFile1, []int{1, 1}, "Alice")).To(Succeed())
			Expect(os.MkdirAll(dir, 0o750)).To(Succeed())
			Expect(os.WriteFile(filepath.Join(dir, duelFile1), fake.Demos[duelFile1], 0o600)).To(Succeed())

			Expect(runDownloader(fake, dir)).To(Succeed())

			Expect(fake.StorageRequests()).To(BeZero())
			Expect(readIndex(dir)).To(MatchJSON(fmt.Sprintf(`{%q: %q}`, duelID1, duelFile1)))
		})

		It("uses the index it is given", func() {
			Expect(fake.AddReplay(duelID1, duelFile1, []int{1, 1}, "Alice")).To(Succeed())
			indexPath := filepath.Join(GinkgoT().TempDir(), "other", indexFileName)
			index, err := index.Open(indexPath)
			Expect(err).NotTo(HaveOccurred())

			Expect(runDownloader(fake, dir, replaydownloader2.WithIndex(index))).To(Succeed())

			fileName, found := index.Get(duelID1)
			Expect(found).To(BeTrue())
			Expect(fileName).To(Equal(duelFile1))
			Expect(os.ReadFile(indexPath)).To(MatchJSON(fmt.Sprintf(`{%q: %q}`, duelID1, duelFile1)))
			Expect(filepath.Join(dir, indexFileName)).NotTo(BeAnExistingFile())
		})
	})

	Context("dry run", func() {
		It("downloads nothing and writes no index", func() {
			Expect(fake.AddReplay(duelID1, duelFile1, []int{1, 1}, "Alice")).To(Succeed())

			Expect(runDownloader(fake, dir, replaydownloader2.WithDryRun())).To(Succeed())

			Expect(fake.StorageRequests()).To(BeZero())
			Expect(dir).NotTo(BeADirectory())
		})
	})

	Context("errors", func() {
		It("stops only the player whose replay fails and reports it at the end", func() {
			Expect(fake.AddReplay(duelID1, duelFile1, []int{1, 1}, "Alice")).To(Succeed())
			Expect(fake.AddReplay(duelID2, duelFile2, []int{1, 1}, "Alice", "Bob")).To(Succeed())
			delete(fake.Demos, duelFile1)

			Expect(runDownloader(fake, dir)).To(HaveOccurred())

			// Alice's newest replay failed, so her next one was left for Bob.
			Expect(fake.RequestsTo("/replays/" + duelID2)).To(HaveLen(1))
			Expect(filepath.Join(dir, duelFile2)).To(BeAnExistingFile())
			Expect(readIndex(dir)).To(MatchJSON(fmt.Sprintf(`{%q: %q}`, duelID2, duelFile2)))
		})

		It("rejects a file name from bar-db which leaves the directory", func() {
			Expect(fake.AddReplay(duelID1, "../escaped.sdfz", []int{1, 1}, "Alice")).To(Succeed())

			Expect(runDownloader(fake, dir)).To(MatchError(ContainSubstring("unexpected characters")))

			Expect(fake.StorageRequests()).To(BeZero())
			Expect(readIndex(dir)).To(MatchJSON(`{}`))
		})

		It("stops at a cancelled context", func() {
			downloader := newDownloader(fake, dir)
			ctx, cancel := context.WithCancel(context.Background())
			cancel()

			Expect(downloader.Run(ctx)).To(MatchError(context.Canceled))
		})

		It("reports an index file which is not JSON", func() {
			Expect(os.MkdirAll(dir, 0o750)).To(Succeed())
			Expect(os.WriteFile(filepath.Join(dir, indexFileName), []byte("{"), 0o600)).To(Succeed())

			Expect(replaydownloader2.New(
				replaydownloader2.WithClient(fake.Client()),
				replaydownloader2.WithDir(dir),
				replaydownloader2.WithGameType(barlive.GameTypeDuel),
			)).Error().To(HaveOccurred())
		})

		DescribeTable("rejects an invalid configuration",
			func(option replaydownloader2.Option) {
				Expect(replaydownloader2.New(
					replaydownloader2.WithClient(fake.Client()),
					replaydownloader2.WithDir(dir),
					replaydownloader2.WithGameType(barlive.GameTypeDuel),
					option,
				)).Error().To(HaveOccurred())
			},
			Entry("no players", replaydownloader2.WithPlayers(0)),
			Entry("no days", replaydownloader2.WithDays(0)),
			Entry("no game type", replaydownloader2.WithGameType("")),
			Entry("no logger", replaydownloader2.WithLogger(nil)),
		)
	})
})

// newDownloader returns a downloader of duel replays from fake into dir, configured further by options, and fails the
// spec on an error.
func newDownloader(
	fake *barlivetest.Fake, dir string, options ...replaydownloader2.Option,
) *replaydownloader2.Downloader {
	downloader, err := replaydownloader2.New(append([]replaydownloader2.Option{
		replaydownloader2.WithClient(fake.Client()),
		replaydownloader2.WithDir(dir),
		replaydownloader2.WithGameType(barlive.GameTypeDuel),
	}, options...)...)
	Expect(err).NotTo(HaveOccurred())
	return downloader
}

// runDownloader runs a downloader as newDownloader returns it and closes it, fails the spec if Close fails, and returns
// the error of Run.
func runDownloader(fake *barlivetest.Fake, dir string, options ...replaydownloader2.Option) error {
	downloader := newDownloader(fake, dir, options...)
	err := downloader.Run(context.Background())
	Expect(downloader.Close()).To(Succeed())
	return err
}

// readIndex returns the content of the index file in dir, and fails the spec if it cannot be read.
func readIndex(dir string) string {
	data, err := os.ReadFile(filepath.Join(dir, indexFileName))
	Expect(err).NotTo(HaveOccurred())
	return string(data)
}
