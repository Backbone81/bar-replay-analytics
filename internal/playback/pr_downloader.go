package playback

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// prDownloader is the file name of the downloader of games and maps that ships with the engine.
var prDownloader = executableName("pr-downloader")

// prDownloaderErrorLines is the number of pr-downloader's last lines which an error of prDownload includes.
const prDownloaderErrorLines = 20

// prDownload downloads a map or game (kind "map" or "game") into the data dir with the pr-downloader next to engine
// and Config.PRDownloaderEnv. On failure the error holds its last lines.
func (p *Player) prDownload(ctx context.Context, engine string, kind string, name string) error {
	program := filepath.Join(filepath.Dir(engine), prDownloader)
	if !isFile(program) {
		return fmt.Errorf("the engine has no %s", prDownloader)
	}
	p.reportDownload(kind + " " + name)

	//nolint:gosec // The engine's pr-downloader; name passed validateReplayName and cannot be taken for an option.
	command := exec.CommandContext(ctx, program, "--filesystem-writepath", p.dataDir, "--download-"+kind, name)
	command.Env = append(os.Environ(), p.config.PRDownloaderEnv...)
	output, err := command.CombinedOutput()
	if err != nil {
		return fmt.Errorf("%s: %w; its last lines:\n%s", prDownloader, err, p.lastLines(output))
	}
	return nil
}

// lastLines returns the last prDownloaderErrorLines non-empty lines of output without the progress bars, which
// pr-downloader redraws with \r: "[Progress]  41% [=====     ] 13893524/34023778".
func (p *Player) lastLines(output []byte) string {
	var lines []string
	for _, line := range bytes.FieldsFunc(output, func(r rune) bool { return r == '\n' || r == '\r' }) {
		text := strings.TrimSpace(string(line))
		if text != "" && !strings.HasPrefix(text, "[Progress]") {
			lines = append(lines, text)
		}
	}
	return strings.Join(lines[max(0, len(lines)-prDownloaderErrorLines):], "\n")
}
