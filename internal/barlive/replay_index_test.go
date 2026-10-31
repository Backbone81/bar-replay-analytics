package barlive_test

import (
	"context"
	"fmt"
	"io/fs"
	"net/http"
	"os"
	"path/filepath"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/backbone81/bar-replay-analytics-go/internal/barlive"
)

var _ = Describe("ReplayIndex", func() {
	var server *testServer

	// dir is the indexed directory.
	var dir string

	// demoFile is the content the storage serves for testDemoFileName.
	var demoFile []byte

	BeforeEach(func() {
		server = newTestServer()
		server.HandleFunc("GET /replays/"+testGameID, serveString(http.StatusOK, "application/json",
			fmt.Sprintf(`{"id":%q,"fileName":%q}`, testGameID, testDemoFileName)))
		demoFile = testDemoFile(testGameID)
		server.HandleFunc("GET "+testDemoPath, serveDemo(demoFile))
		dir = filepath.Join(GinkgoT().TempDir(), "leaderboard")
	})

	It("starts empty for a missing directory", func() {
		index := openIndex(dir)

		Expect(index.Has(testGameID)).To(BeFalse())
		Expect(index.Len()).To(BeZero())
	})

	It("downloads a missing replay, records it and keeps the record", func() {
		index := openIndex(dir)
		Expect(os.MkdirAll(dir, 0o750)).To(Succeed())

		result, err := index.Fetch(context.Background(), server.client(), testGameID)

		Expect(err).NotTo(HaveOccurred())
		Expect(result).To(Equal(barlive.FetchResult{FileName: testDemoFileName, Downloaded: true}))
		Expect(os.ReadFile(filepath.Join(dir, testDemoFileName))).To(Equal(demoFile))
		Expect(index.Has(testGameID)).To(BeTrue())

		Expect(index.Save()).To(Succeed())
		Expect(os.ReadFile(filepath.Join(dir, barlive.ReplayIndexFileName))).To(MatchJSON(
			fmt.Sprintf(`{%q: %q}`, testGameID, testDemoFileName)))
		Expect(openIndex(dir).Has(testGameID)).To(BeTrue())
	})

	It("sends no request for a recorded replay whose file exists", func() {
		Expect(os.MkdirAll(dir, 0o750)).To(Succeed())
		first := openIndex(dir)
		Expect(first.Fetch(context.Background(), server.client(), testGameID)).Error().NotTo(HaveOccurred())
		Expect(first.Save()).To(Succeed())
		index := openIndex(dir)
		recordedBefore := len(server.recorded())

		result, err := index.Fetch(context.Background(), server.client(), testGameID)

		Expect(err).NotTo(HaveOccurred())
		Expect(result).To(Equal(barlive.FetchResult{FileName: testDemoFileName}))
		Expect(server.recorded()).To(HaveLen(recordedBefore))
	})

	It("downloads a recorded replay again whose file was removed", func() {
		Expect(os.MkdirAll(dir, 0o750)).To(Succeed())
		index := openIndex(dir)
		Expect(index.Fetch(context.Background(), server.client(), testGameID)).Error().NotTo(HaveOccurred())
		Expect(index.Save()).To(Succeed())
		Expect(os.Remove(filepath.Join(dir, testDemoFileName))).To(Succeed())

		index = openIndex(dir)
		Expect(index.Has(testGameID)).To(BeFalse())
		result, err := index.Fetch(context.Background(), server.client(), testGameID)

		Expect(err).NotTo(HaveOccurred())
		Expect(result.Downloaded).To(BeTrue())
		Expect(storageRequests(server)).To(Equal(2))
	})

	It("records a file added by hand without downloading it", func() {
		Expect(os.MkdirAll(dir, 0o750)).To(Succeed())
		Expect(os.WriteFile(filepath.Join(dir, testDemoFileName), demoFile, 0o600)).To(Succeed())
		index := openIndex(dir)

		result, err := index.Fetch(context.Background(), server.client(), testGameID)

		Expect(err).NotTo(HaveOccurred())
		Expect(result).To(Equal(barlive.FetchResult{FileName: testDemoFileName}))
		Expect(index.Has(testGameID)).To(BeTrue())
		Expect(storageRequests(server)).To(BeZero())
	})

	It("reports a file of the same name holding another game and keeps it", func() {
		Expect(os.MkdirAll(dir, 0o750)).To(Succeed())
		otherGame := testDemoFile("ffffffffffffffffffffffffffffffff")
		Expect(os.WriteFile(filepath.Join(dir, testDemoFileName), otherGame, 0o600)).To(Succeed())
		index := openIndex(dir)

		Expect(index.Fetch(context.Background(), server.client(), testGameID)).Error().To(MatchError(fs.ErrExist))

		Expect(index.Has(testGameID)).To(BeFalse())
		Expect(os.ReadFile(filepath.Join(dir, testDemoFileName))).To(Equal(otherGame))
		Expect(storageRequests(server)).To(BeZero())
	})

	It("drops the entries of removed files when it saves", func() {
		Expect(os.MkdirAll(dir, 0o750)).To(Succeed())
		Expect(os.WriteFile(filepath.Join(dir, barlive.ReplayIndexFileName),
			[]byte(`{"00000000000000000000000000000009": "removed.sdfz"}`), 0o600)).To(Succeed())

		Expect(openIndex(dir).Save()).To(Succeed())

		Expect(os.ReadFile(filepath.Join(dir, barlive.ReplayIndexFileName))).To(MatchJSON(`{}`))
	})

	It("writes the index readable by everyone", func() {
		Expect(openIndex(dir).Save()).To(Succeed())

		info, err := os.Stat(filepath.Join(dir, barlive.ReplayIndexFileName))
		Expect(err).NotTo(HaveOccurred())
		Expect(info.Mode().Perm()).To(Equal(os.FileMode(0o644)))
	})

	It("creates the directory when it saves", func() {
		Expect(openIndex(dir).Save()).To(Succeed())

		Expect(os.ReadFile(filepath.Join(dir, barlive.ReplayIndexFileName))).To(MatchJSON(`{}`))
		Expect(os.ReadDir(dir)).To(HaveLen(1))
	})

	It("reports an index file which is not JSON", func() {
		Expect(os.MkdirAll(dir, 0o750)).To(Succeed())
		Expect(os.WriteFile(filepath.Join(dir, barlive.ReplayIndexFileName), []byte("{"), 0o600)).To(Succeed())

		Expect(barlive.OpenReplayIndex(dir)).Error().To(MatchError(ContainSubstring("remove the file")))
	})

	It("reports a failed lookup and records nothing", func() {
		Expect(os.MkdirAll(dir, 0o750)).To(Succeed())
		index := openIndex(dir)

		Expect(index.Fetch(context.Background(), server.client(), "00000000000000000000000000000009")).Error().
			To(HaveOccurred())
		Expect(index.Len()).To(BeZero())
	})
})

// openIndex opens the index of dir, and fails the spec on an error.
func openIndex(dir string) *barlive.ReplayIndex {
	index, err := barlive.OpenReplayIndex(dir)
	Expect(err).NotTo(HaveOccurred())
	return index
}

// storageRequests returns the number of requests server received for testDemoPath.
func storageRequests(server *testServer) int {
	count := 0
	for _, request := range server.recorded() {
		if request.Path == testDemoPath {
			count++
		}
	}
	return count
}
