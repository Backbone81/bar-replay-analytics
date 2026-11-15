package utils_test

import (
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing/iotest"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/backbone81/bar-replay-analytics/internal/utils"
)

var _ = Describe("Files", func() {
	Context("WriteFile", func() {
		It("writes the data and leaves no temporary file", func() {
			dir := GinkgoT().TempDir()

			Expect(utils.WriteFile(filepath.Join(dir, "file"), []byte("content"), 0o640)).To(Succeed())

			Expect(os.ReadFile(filepath.Join(dir, "file"))).To(Equal([]byte("content")))
			Expect(fileNames(dir)).To(ConsistOf("file"))
		})

		It("replaces an existing file", func() {
			dir := GinkgoT().TempDir()
			writeFile(filepath.Join(dir, "file"), "an older and longer content")

			Expect(utils.WriteFile(filepath.Join(dir, "file"), []byte("new"), 0o644)).To(Succeed())

			Expect(os.ReadFile(filepath.Join(dir, "file"))).To(Equal([]byte("new")))
			Expect(fileNames(dir)).To(ConsistOf("file"))
		})
	})

	Context("WriteFileFrom", func() {
		It("writes everything the reader returns", func() {
			dir := GinkgoT().TempDir()

			Expect(utils.WriteFileFrom(filepath.Join(dir, "file"), strings.NewReader("content"), 0o755)).To(Succeed())

			Expect(os.ReadFile(filepath.Join(dir, "file"))).To(Equal([]byte("content")))
		})

		It("leaves an existing file as it was and no temporary file if the reader fails", func() {
			dir := GinkgoT().TempDir()
			writeFile(filepath.Join(dir, "file"), "old")
			reader := io.MultiReader(strings.NewReader("partial"), iotest.ErrReader(errTestRead))

			Expect(utils.WriteFileFrom(filepath.Join(dir, "file"), reader, 0o644)).To(MatchError(errTestRead))

			Expect(os.ReadFile(filepath.Join(dir, "file"))).To(Equal([]byte("old")))
			Expect(fileNames(dir)).To(ConsistOf("file"))
		})

		It("fails if the file is a folder and leaves no temporary file", func() {
			dir := GinkgoT().TempDir()
			Expect(os.Mkdir(filepath.Join(dir, "file"), 0o750)).To(Succeed())

			Expect(utils.WriteFileFrom(filepath.Join(dir, "file"), strings.NewReader("content"), 0o644)).
				ToNot(Succeed())

			Expect(fileNames(dir)).To(ConsistOf("file"))
		})

		It("fails if the folder does not exist", func() {
			dir := GinkgoT().TempDir()

			Expect(utils.WriteFileFrom(filepath.Join(dir, "missing", "file"), strings.NewReader("content"), 0o644)).
				To(MatchError(os.ErrNotExist))
		})
	})

	Context("CopyFile", func() {
		It("copies the file and leaves no temporary file", func() {
			dir := GinkgoT().TempDir()
			writeFile(filepath.Join(dir, "source"), "content")

			Expect(utils.CopyFile(filepath.Join(dir, "source"), filepath.Join(dir, "destination"))).To(Succeed())

			Expect(os.ReadFile(filepath.Join(dir, "destination"))).To(Equal([]byte("content")))
			Expect(fileNames(dir)).To(ConsistOf("source", "destination"))
		})

		It("fails without the source and leaves the destination as it was", func() {
			dir := GinkgoT().TempDir()
			writeFile(filepath.Join(dir, "destination"), "old")

			Expect(utils.CopyFile(filepath.Join(dir, "missing"), filepath.Join(dir, "destination"))).
				To(MatchError(os.ErrNotExist))

			Expect(os.ReadFile(filepath.Join(dir, "destination"))).To(Equal([]byte("old")))
			Expect(fileNames(dir)).To(ConsistOf("destination"))
		})
	})
})
