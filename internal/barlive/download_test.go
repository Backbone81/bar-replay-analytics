package barlive_test

import (
	"bytes"
	"compress/gzip"
	"context"
	"encoding/binary"
	"encoding/hex"
	"io/fs"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/backbone81/bar-replay-analytics-go/internal/barlive"
	"github.com/backbone81/bar-replay-analytics-go/internal/demo"
)

// testDemoFileName is a file name as the storage holds them, with spaces.
const testDemoFileName = "2026-10-28_16-27-53-947_Aethermoor Creek 1.0_2026.07.04.sdfz"

// testDemoPath is the path the test server receives for testDemoFileName.
const testDemoPath = testStoragePath + "/demos/2026-10-28_16-27-53-947_Aethermoor%20Creek%201.0_2026.07.04.sdfz"

// testGameID is the game in the demo file testDemoFile returns, as bar-db names the replay.
const testGameID = "0102030405060708090a0b0c0d0e0f10"

// testDemoContent stands in for a replay file where the client does not look into it.
var testDemoContent = []byte("replay file content")

// testDemoFile returns a gzip-compressed demo file of the game gameID with a valid header followed by filler, which is
// all DownloadDemoIntoDir reads.
func testDemoFile(gameID string) []byte {
	header := demo.DemoFileHeader{
		Version:            demo.DemofileVersion,
		HeaderSize:         int32(binary.Size(demo.DemoFileHeader{})),
		ScriptSize:         1,
		PlayerStatElemSize: int32(binary.Size(demo.PlayerStatistics{})),
		TeamStatElemSize:   int32(binary.Size(demo.TeamStatistics{})),
		TeamStatPeriod:     15,
	}
	copy(header.Magic[:], demo.DemofileMagic)
	copy(header.VersionString[:], "2026.07.04")
	id, err := hex.DecodeString(gameID)
	Expect(err).NotTo(HaveOccurred())
	copy(header.GameID[:], id)

	var buffer bytes.Buffer
	gzipWriter := gzip.NewWriter(&buffer)
	Expect(binary.Write(gzipWriter, demo.ByteOrder, header)).To(Succeed())
	Expect(gzipWriter.Write([]byte("rest of the demo"))).Error().NotTo(HaveOccurred())
	Expect(gzipWriter.Close()).To(Succeed())
	return buffer.Bytes()
}

// serveDemo answers like the storage, with content.
func serveDemo(content []byte) http.HandlerFunc {
	return func(writer http.ResponseWriter, _ *http.Request) {
		// The storage really sends this content type.
		writer.Header().Set("Content-Type", "application/x-www-form-urlencoded")
		_, _ = writer.Write(content)
	}
}

// serveBrokenDemo announces more than it sends, so the client sees an unexpected EOF.
func serveBrokenDemo(writer http.ResponseWriter, _ *http.Request) {
	writer.Header().Set("Content-Length", "1000")
	_, _ = writer.Write(testDemoContent)
}

// serveStorageNotFound answers like the storage for a missing file.
var serveStorageNotFound = serveString(http.StatusNotFound, "text/html; charset=UTF-8",
	"<html><h1>Not Found</h1><p>The resource could not be found.</p></html>")

var _ = Describe("Download", func() {
	var server *testServer

	BeforeEach(func() {
		server = newTestServer()
		server.HandleFunc("GET "+testStoragePath+"/demos/", serveStorageNotFound)
	})

	Context("DownloadDemo", func() {
		It("writes the demo to the writer", func() {
			server.HandleFunc("GET "+testDemoPath, serveDemo(testDemoContent))
			var buffer bytes.Buffer

			Expect(server.client().DownloadDemo(context.Background(), testDemoFileName, &buffer)).To(Succeed())

			Expect(buffer.Bytes()).To(Equal(testDemoContent))
			Expect(server.recorded()[0].Path).To(Equal(testDemoPath))
		})

		It("reports the download to OnRequest", func() {
			server.HandleFunc("GET "+testDemoPath, serveDemo(testDemoContent))
			recorder := &requestRecorder{}
			client := server.client(barlive.WithOnRequest(recorder.record))

			Expect(client.DownloadDemo(context.Background(), testDemoFileName, &bytes.Buffer{})).To(Succeed())

			Expect(recorder.recordedURLs()).To(Equal([]string{server.server.URL + testDemoPath}))
		})

		It("accepts an apostrophe, which map names hold", func() {
			const fileName = "2026-10-28_14-02-51-887_Devil's Postpiles 1.1.1_2026.07.04.sdfz"
			server.HandleFunc("GET "+testStoragePath+"/demos/"+url.PathEscape(fileName), serveDemo(testDemoContent))
			var buffer bytes.Buffer

			Expect(server.client().DownloadDemo(context.Background(), fileName, &buffer)).To(Succeed())

			Expect(buffer.Bytes()).To(Equal(testDemoContent))
		})

		It("is not rate limited", func() {
			server.HandleFunc("GET "+testDemoPath, serveDemo(testDemoContent))
			client := server.client(barlive.WithMaxRequestRate(1.0 / 3600))
			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()

			for range 3 {
				Expect(client.DownloadDemo(ctx, testDemoFileName, &bytes.Buffer{})).To(Succeed())
			}

			Expect(server.recorded()).To(HaveLen(3))
		})

		It("reports a missing demo", func() {
			var buffer bytes.Buffer

			Expect(server.client().DownloadDemo(context.Background(), testDemoFileName, &buffer)).NotTo(Succeed())
			Expect(buffer.Len()).To(BeZero())
		})

		It("reports another status", func() {
			server.HandleFunc("GET "+testDemoPath, serveString(http.StatusServiceUnavailable, "text/plain", "try later"))

			Expect(server.client().DownloadDemo(context.Background(), testDemoFileName, &bytes.Buffer{})).NotTo(Succeed())
		})

		It("reports a download which breaks off", func() {
			server.HandleFunc("GET "+testDemoPath, serveBrokenDemo)

			Expect(server.client().DownloadDemo(context.Background(), testDemoFileName, &bytes.Buffer{})).NotTo(Succeed())
		})

		DescribeTable("rejects a file name with unexpected characters without a request",
			func(fileName string) {
				Expect(server.client().DownloadDemo(context.Background(), fileName, &bytes.Buffer{})).NotTo(Succeed())
				Expect(server.recorded()).To(BeEmpty())
			},
			Entry("empty", ""),
			Entry("current directory", "."),
			Entry("parent directory", ".."),
			Entry("path", "demos/a.sdfz"),
			Entry("parent path", "../a.sdfz"),
			Entry("absolute path", "/tmp/a.sdfz"),
			Entry("Windows path", `demos\a.sdfz`),
			Entry("Windows drive", "C:a.sdfz"),
			Entry("query", "a.sdfz?b"),
			Entry("fragment", "a.sdfz#b"),
			Entry("percent", "a%2Fb.sdfz"),
			Entry("newline", "a\nb.sdfz"),
			Entry("non-ASCII letter", "Ä.sdfz"),
			Entry("plus", "a+b.sdfz"),
		)
	})

	Context("DownloadDemoIntoDir", func() {
		// dir is where DownloadDemoIntoDir stores the demo files.
		var dir string

		// replay is the replay to download, whose demo file the storage serves as testDemoPath.
		var replay *barlive.Replay

		BeforeEach(func() {
			dir = GinkgoT().TempDir()
			replay = &barlive.Replay{
				ID:       testGameID,
				FileName: testDemoFileName,
			}
		})

		It("stores the demo file under its name", func() {
			content := testDemoFile(testGameID)
			server.HandleFunc("GET "+testDemoPath, serveDemo(content))

			path, err := server.client().DownloadDemoIntoDir(context.Background(), replay, dir)

			Expect(err).NotTo(HaveOccurred())
			Expect(path).To(Equal(filepath.Join(dir, testDemoFileName)))
			Expect(os.ReadFile(path)).To(Equal(content))
			Expect(os.ReadDir(dir)).To(HaveLen(1))
		})

		It("replaces a .part file left behind by a killed process", func() {
			content := testDemoFile(testGameID)
			server.HandleFunc("GET "+testDemoPath, serveDemo(content))
			partPath := filepath.Join(dir, testDemoFileName+".part")
			Expect(os.WriteFile(partPath, []byte("a longer partial download from before"), 0o600)).To(Succeed())

			path, err := server.client().DownloadDemoIntoDir(context.Background(), replay, dir)

			Expect(err).NotTo(HaveOccurred())
			Expect(os.ReadFile(path)).To(Equal(content))
			Expect(partPath).NotTo(BeAnExistingFile())
		})

		It("does not overwrite an existing file and sends no request", func() {
			path := filepath.Join(dir, testDemoFileName)
			Expect(os.WriteFile(path, testDemoContent, 0o600)).To(Succeed())

			Expect(server.client().DownloadDemoIntoDir(context.Background(), replay, dir)).Error().To(MatchError(fs.ErrExist))

			Expect(os.ReadFile(path)).To(Equal(testDemoContent))
			Expect(server.recorded()).To(BeEmpty())
		})

		It("rejects a demo file of another game and keeps no file", func() {
			server.HandleFunc("GET "+testDemoPath, serveDemo(testDemoFile("ffffffffffffffffffffffffffffffff")))

			Expect(server.client().DownloadDemoIntoDir(context.Background(), replay, dir)).Error().To(
				MatchError(ContainSubstring("holds game ffffffffffffffffffffffffffffffff")))
			Expect(os.ReadDir(dir)).To(BeEmpty())
		})

		It("rejects a file which is no demo and keeps no file", func() {
			server.HandleFunc("GET "+testDemoPath, serveDemo(testDemoContent))

			Expect(server.client().DownloadDemoIntoDir(context.Background(), replay, dir)).Error().To(HaveOccurred())
			Expect(os.ReadDir(dir)).To(BeEmpty())
		})

		It("keeps no file of a missing demo", func() {
			Expect(server.client().DownloadDemoIntoDir(context.Background(), replay, dir)).Error().To(HaveOccurred())
			Expect(os.ReadDir(dir)).To(BeEmpty())
		})

		It("keeps no file of a download which breaks off", func() {
			server.HandleFunc("GET "+testDemoPath, serveBrokenDemo)

			Expect(server.client().DownloadDemoIntoDir(context.Background(), replay, dir)).Error().To(HaveOccurred())
			Expect(os.ReadDir(dir)).To(BeEmpty())
		})

		It("keeps no file if the context is cancelled", func() {
			ctx, cancel := context.WithCancel(context.Background())
			cancel()

			Expect(server.client().DownloadDemoIntoDir(ctx, replay, dir)).Error().To(MatchError(context.Canceled))
			Expect(os.ReadDir(dir)).To(BeEmpty())
		})

		It("rejects a file name which would leave the directory without a request", func() {
			replay.FileName = "../" + testDemoFileName

			Expect(server.client().DownloadDemoIntoDir(context.Background(), replay, dir)).Error().To(HaveOccurred())
			Expect(server.recorded()).To(BeEmpty())
			Expect(os.ReadDir(dir)).To(BeEmpty())
		})
	})
})
