package barlive_test

import (
	"context"
	"net/http"
	"os"
	"path/filepath"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/backbone81/bar-replay-analytics-go/internal/barlive"
)

// testDemoFileName is a file name as the storage holds them, with spaces.
const testDemoFileName = "2026-10-28_16-27-53-947_Aethermoor Creek 1.0_2026.07.04.sdfz"

// testDemoPath is the path the test server receives for testDemoFileName.
const testDemoPath = testStoragePath + "/demos/2026-10-28_16-27-53-947_Aethermoor%20Creek%201.0_2026.07.04.sdfz"

// testDemoContent stands in for a replay file; the client does not look into it.
var testDemoContent = []byte("replay file content")

// serveDemo answers like the storage, with content.
func serveDemo(content []byte) http.HandlerFunc {
	return func(writer http.ResponseWriter, _ *http.Request) {
		// The storage really sends this content type.
		writer.Header().Set("Content-Type", "application/x-www-form-urlencoded")
		_, _ = writer.Write(content)
	}
}

// serveStorageNotFound answers like the storage for a missing file.
var serveStorageNotFound = serveString(http.StatusNotFound, "text/html; charset=UTF-8",
	"<html><h1>Not Found</h1><p>The resource could not be found.</p></html>")

var _ = Describe("DownloadDemo", func() {
	var server *testServer

	// tempDir is where DownloadDemo creates its temporary files.
	var tempDir string

	BeforeEach(func() {
		server = newTestServer()
		server.HandleFunc("GET "+testStoragePath+"/demos/", serveStorageNotFound)

		tempDir = GinkgoT().TempDir()
		GinkgoT().Setenv("TMPDIR", tempDir)
	})

	It("downloads a demo into a temporary file", func() {
		server.HandleFunc("GET "+testDemoPath, serveDemo(testDemoContent))

		path, err := server.client().DownloadDemo(context.Background(), testDemoFileName)

		Expect(err).NotTo(HaveOccurred())
		Expect(filepath.Dir(path)).To(Equal(tempDir))
		Expect(os.ReadFile(path)).To(Equal(testDemoContent))
		Expect(server.recorded()[0].Path).To(Equal(testDemoPath))
	})

	It("creates a new file for each download", func() {
		server.HandleFunc("GET "+testDemoPath, serveDemo(testDemoContent))
		client := server.client()

		first, err := client.DownloadDemo(context.Background(), testDemoFileName)
		Expect(err).NotTo(HaveOccurred())
		second, err := client.DownloadDemo(context.Background(), testDemoFileName)
		Expect(err).NotTo(HaveOccurred())

		Expect(second).NotTo(Equal(first))
		Expect(os.ReadDir(tempDir)).To(HaveLen(2))
	})

	It("is not rate limited", func() {
		server.HandleFunc("GET "+testDemoPath, serveDemo(testDemoContent))
		client := server.client(barlive.WithMaxRequestRate(1.0 / 3600))
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		for range 3 {
			Expect(client.DownloadDemo(ctx, testDemoFileName)).Error().NotTo(HaveOccurred())
		}

		Expect(server.recorded()).To(HaveLen(3))
	})

	It("reports a missing demo and creates no file", func() {
		Expect(server.client().DownloadDemo(context.Background(), testDemoFileName)).Error().To(HaveOccurred())
		Expect(os.ReadDir(tempDir)).To(BeEmpty())
	})

	It("reports another status and creates no file", func() {
		server.HandleFunc("GET "+testDemoPath, serveString(http.StatusServiceUnavailable, "text/plain", "try later"))

		Expect(server.client().DownloadDemo(context.Background(), testDemoFileName)).Error().To(HaveOccurred())
		Expect(os.ReadDir(tempDir)).To(BeEmpty())
	})

	It("removes the temporary file if the download breaks off", func() {
		server.HandleFunc("GET "+testDemoPath, func(writer http.ResponseWriter, _ *http.Request) {
			// Announce more than is sent, so the client sees an unexpected EOF.
			writer.Header().Set("Content-Length", "1000")
			_, _ = writer.Write(testDemoContent)
		})

		Expect(server.client().DownloadDemo(context.Background(), testDemoFileName)).Error().To(HaveOccurred())
		Expect(os.ReadDir(tempDir)).To(BeEmpty())
	})

	DescribeTable("rejects a file name with unexpected characters without a request",
		func(fileName string) {
			Expect(server.client().DownloadDemo(context.Background(), fileName)).Error().To(HaveOccurred())
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
