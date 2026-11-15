package index_test

import (
	"os"
	"path/filepath"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/backbone81/bar-replay-analytics/internal/replay/index"
)

var _ = Describe("ReplayIndex", func() {
	// filePath is the index file, in a directory which does not exist before a spec creates it.
	var filePath string

	BeforeEach(func() {
		filePath = filepath.Join(GinkgoT().TempDir(), "leaderboard", "index.json")
	})

	It("starts empty for a missing file", func() {
		_, found := openIndex(filePath).Get("a")
		Expect(found).To(BeFalse())
	})

	It("starts empty for an empty file", func() {
		Expect(os.MkdirAll(filepath.Dir(filePath), 0o750)).To(Succeed())
		Expect(os.WriteFile(filePath, nil, 0o600)).To(Succeed())

		_, found := openIndex(filePath).Get("a")
		Expect(found).To(BeFalse())
	})

	It("keeps the entries it saves", func() {
		index := openIndex(filePath)
		index.Add("a", "a.sdfz")
		index.Add("b", "b.sdfz")
		Expect(index.Save()).To(Succeed())

		Expect(os.ReadFile(filePath)).To(MatchJSON(`{"a": "a.sdfz", "b": "b.sdfz"}`))
		fileName, found := openIndex(filePath).Get("b")
		Expect(found).To(BeTrue())
		Expect(fileName).To(Equal("b.sdfz"))
	})

	It("forgets a removed entry", func() {
		index := openIndex(filePath)
		index.Add("a", "a.sdfz")
		index.Remove("a")

		_, found := index.Get("a")
		Expect(found).To(BeFalse())
		Expect(index.Save()).To(Succeed())
		Expect(os.ReadFile(filePath)).To(MatchJSON(`{}`))
	})

	It("reports a file which is not JSON", func() {
		Expect(os.MkdirAll(filepath.Dir(filePath), 0o750)).To(Succeed())
		Expect(os.WriteFile(filePath, []byte("{"), 0o600)).To(Succeed())

		Expect(index.Open(filePath)).Error().To(HaveOccurred())
	})
})

// openIndex opens the index file filePath, and fails the spec on an error.
func openIndex(filePath string) *index.ReplayIndex {
	index, err := index.Open(filePath)
	Expect(err).NotTo(HaveOccurred())
	return index
}
