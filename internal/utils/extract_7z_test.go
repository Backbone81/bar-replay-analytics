package utils_test

import (
	"context"
	"os"
	"path/filepath"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/backbone81/bar-replay-analytics/internal/utils"
)

var _ = Describe("Extract7z", func() {
	It("unpacks folders and files and leaves no temporary folder", func(ctx SpecContext) {
		archive := copyTestdata("engine.7z", GinkgoT().TempDir())
		dir := filepath.Join(filepath.Dir(archive), "engine")

		Expect(utils.Extract7z(ctx, archive, dir)).To(Succeed())

		Expect(os.ReadFile(filepath.Join(dir, "spring-headless"))).To(Equal([]byte("fake engine\n")))
		Expect(os.ReadFile(filepath.Join(dir, "pr-downloader"))).To(Equal([]byte("fake downloader\n")))
		Expect(os.ReadFile(filepath.Join(dir, "AI", "config.json"))).To(Equal([]byte("data\n")))
		Expect(fileNames(filepath.Dir(archive))).To(ConsistOf("engine.7z", "engine"))
	})

	It("replaces an existing folder with everything in it", func(ctx SpecContext) {
		archive := copyTestdata("engine.7z", GinkgoT().TempDir())
		dir := filepath.Join(filepath.Dir(archive), "engine")
		writeFile(filepath.Join(dir, "spring-headless"), "old engine, longer than the new one\n")
		writeFile(filepath.Join(dir, "stale"), "")

		Expect(utils.Extract7z(ctx, archive, dir)).To(Succeed())

		Expect(os.ReadFile(filepath.Join(dir, "spring-headless"))).To(Equal([]byte("fake engine\n")))
		Expect(filepath.Join(dir, "stale")).NotTo(BeAnExistingFile())
	})

	It("stops when the context is canceled and leaves no folder", func(ctx SpecContext) {
		archive := copyTestdata("engine.7z", GinkgoT().TempDir())
		cancelableCtx, cancel := context.WithCancel(ctx)
		cancel()

		Expect(utils.Extract7z(cancelableCtx, archive, filepath.Join(filepath.Dir(archive), "engine"))).
			To(MatchError(context.Canceled))

		Expect(fileNames(filepath.Dir(archive))).To(ConsistOf("engine.7z"))
	})

	It("rejects an entry that leaves the destination folder", func(ctx SpecContext) {
		dir := filepath.Join(GinkgoT().TempDir(), "inner")
		Expect(os.Mkdir(dir, 0o750)).To(Succeed())
		archive := copyTestdata("escaping.7z", dir)

		Expect(utils.Extract7z(ctx, archive, filepath.Join(dir, "escaping"))).ToNot(Succeed())

		Expect(filepath.Join(filepath.Dir(dir), "escaped")).NotTo(BeAnExistingFile())
		Expect(fileNames(dir)).To(ConsistOf("escaping.7z"))
	})

	It("rejects a symbolic link and leaves no folder", func(ctx SpecContext) {
		archive := copyTestdata("symlink.7z", GinkgoT().TempDir())

		Expect(utils.Extract7z(ctx, archive, filepath.Join(filepath.Dir(archive), "symlink"))).ToNot(Succeed())

		Expect(fileNames(filepath.Dir(archive))).To(ConsistOf("symlink.7z"))
	})

	It("fails on a file that is no 7z archive and leaves an existing folder as it was", func(ctx SpecContext) {
		archive := filepath.Join(GinkgoT().TempDir(), "engine.7z")
		writeFile(archive, "no archive")
		dir := filepath.Join(filepath.Dir(archive), "engine")
		writeFile(filepath.Join(dir, "spring-headless"), "old engine\n")

		Expect(utils.Extract7z(ctx, archive, dir)).ToNot(Succeed())

		Expect(os.ReadFile(filepath.Join(dir, "spring-headless"))).To(Equal([]byte("old engine\n")))
		Expect(fileNames(filepath.Dir(archive))).To(ConsistOf("engine.7z", "engine"))
	})
})
