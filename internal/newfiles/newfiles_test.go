package newfiles_test

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
	"testing/fstest"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/backbone81/bar-replay-analytics-go/internal/newfiles"
)

func TestNewFiles(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "NewFiles Suite")
}

var _ = Describe("Write", func() {
	files := fstest.MapFS{
		"a.md":          {Data: []byte("a")},
		"work/.keep":    {Data: []byte{}},
		"work/sub/b.md": {Data: []byte("b")},
	}

	It("creates every file and folder in an empty folder", func() {
		dir := GinkgoT().TempDir()

		created, err := newfiles.Write(files, dir)

		Expect(err).NotTo(HaveOccurred())
		Expect(created).To(ConsistOf(
			filepath.Join(dir, "a.md"),
			filepath.Join(dir, "work", ".keep"),
			filepath.Join(dir, "work", "sub", "b.md"),
		))
		Expect(os.ReadFile(filepath.Join(dir, "a.md"))).To(Equal([]byte("a")))
		Expect(os.ReadFile(filepath.Join(dir, "work", ".keep"))).To(BeEmpty())
		Expect(os.ReadFile(filepath.Join(dir, "work", "sub", "b.md"))).To(Equal([]byte("b")))
	})

	It("creates only the missing files and reports the existing ones without changing them", func() {
		dir := GinkgoT().TempDir()
		Expect(os.WriteFile(filepath.Join(dir, "a.md"), []byte("mine"), 0o600)).To(Succeed())

		created, err := newfiles.Write(files, dir)

		var existErr *newfiles.ExistError
		Expect(errors.As(err, &existErr)).To(BeTrue())
		Expect(existErr.FilePaths).To(Equal([]string{filepath.Join(dir, "a.md")}))
		Expect(created).To(ConsistOf(
			filepath.Join(dir, "work", ".keep"),
			filepath.Join(dir, "work", "sub", "b.md"),
		))
		Expect(os.ReadFile(filepath.Join(dir, "a.md"))).To(Equal([]byte("mine")))
	})
})
