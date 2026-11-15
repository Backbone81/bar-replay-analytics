package utils_test

import (
	"bytes"
	"io/fs"
	"log/slog"
	"os"
	"path/filepath"
	"testing/fstest"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/backbone81/bar-replay-analytics/internal/utils"
)

var _ = Describe("WriteTemplate", func() {
	template := fstest.MapFS{
		"a.md":          {Data: []byte("a")},
		"empty":         {Mode: fs.ModeDir},
		"work/.keep":    {Data: []byte{}},
		"work/sub/b.md": {Data: []byte("b")},
	}
	discardLog := slog.New(slog.DiscardHandler)

	It("writes every file and folder into a folder that does not exist yet", func() {
		dir := filepath.Join(GinkgoT().TempDir(), "new")

		Expect(utils.WriteTemplate(discardLog, template, dir)).To(Succeed())

		Expect(os.ReadFile(filepath.Join(dir, "a.md"))).To(Equal([]byte("a")))
		Expect(filepath.Join(dir, "empty")).To(BeADirectory())
		Expect(os.ReadFile(filepath.Join(dir, "work", ".keep"))).To(BeEmpty())
		Expect(os.ReadFile(filepath.Join(dir, "work", "sub", "b.md"))).To(Equal([]byte("b")))
	})

	It("logs each file before creating it", func() {
		dir := GinkgoT().TempDir()
		var output bytes.Buffer

		Expect(utils.WriteTemplate(slog.New(slog.NewTextHandler(&output, nil)), template, dir)).To(Succeed())

		Expect(output.String()).To(ContainSubstring(filepath.Join(dir, "a.md")))
		Expect(output.String()).To(ContainSubstring(filepath.Join(dir, "work", "sub", "b.md")))
	})

	It("stops at a file that exists and leaves it unchanged", func() {
		dir := GinkgoT().TempDir()
		existing := filepath.Join(dir, "a.md")
		Expect(os.WriteFile(existing, []byte("mine"), 0o600)).To(Succeed())

		err := utils.WriteTemplate(discardLog, template, dir)

		Expect(err).To(MatchError(fs.ErrExist))
		Expect(err).To(MatchError(ContainSubstring(existing)))
		Expect(os.ReadFile(existing)).To(Equal([]byte("mine")))
		Expect(filepath.Join(dir, "work")).NotTo(BeAnExistingFile())
	})
})
