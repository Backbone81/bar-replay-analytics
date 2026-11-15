package widget_test

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"io/fs"
	"path"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/backbone81/bar-replay-analytics-go/internal/replay/player/widget"
)

// expectedLuaFilesHash is the hash of the template's .lua files (see luaFilesHash) when the widget was last checked.
// When the widget changes, the test fails: decide whether the events it writes changed (then increase FORMAT_VERSION
// in the widget), then set expectedLuaFilesHash to the hash the test prints.
const expectedLuaFilesHash = "42abedcc494bd127fdc1437efa504fd9cd40ff6bb4c79b141c32a40dca619622"

var _ = Describe("FormatVersion", func() {
	It("was reconsidered when the widget changed", func() {
		hash := luaFilesHash(widget.Template)
		Expect(hash).To(Equal(expectedLuaFilesHash), `The analytics widget changed. If the events it writes changed (an
event, field or value added, removed or computed differently), increase FORMAT_VERSION in %s (now %d), so that
replay play plays replays with older events files again. Then set expectedLuaFilesHash in version_test.go to %q.`,
			widget.FileName, widget.FormatVersion, hash)
	})
})

// luaFilesHash returns the SHA-256, as hex, of the .lua files in template, which decide what the widget writes: each
// file's path and content, sorted by path, with CRLF as LF, so that a Windows checkout (.gitattributes checks them out
// with CRLF) and one with LF hash the same. README, license and dot files do not count.
func luaFilesHash(template fs.FS) string {
	hash := sha256.New()
	err := fs.WalkDir(template, ".", func(filePath string, entry fs.DirEntry, err error) error {
		if err != nil || entry.IsDir() || path.Ext(filePath) != ".lua" {
			return err
		}
		data, err := fs.ReadFile(template, filePath)
		if err != nil {
			return err
		}
		data = bytes.ReplaceAll(data, []byte("\r\n"), []byte("\n"))
		hash.Write([]byte(filePath))
		hash.Write([]byte{0})
		hash.Write(data)
		hash.Write([]byte{0})
		return nil
	})
	Expect(err).NotTo(HaveOccurred())
	return hex.EncodeToString(hash.Sum(nil))
}
