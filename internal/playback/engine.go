package playback

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
)

// springHeadless is the file name of the headless engine, which plays replays without a window.
var springHeadless = executableName("spring-headless")

// sevenZipNames are the names 7-Zip's command line tool goes by, in the order they are looked for on PATH.
var sevenZipNames = []string{"7z", "7zz", "7za"}

// EngineFolderName returns the folder name in engine/ which the BAR lobby uses for engineVersion, so a BAR install's
// data dir works. It follows BYAR-Chobby gui_steam_coop_handler.lua, GetEnginePath: "2025.01.6" is
// "rel2501.2025.01.6", "2026.07.04" is "recoil_2026.07.04", and older versions are lower case with " BAR105" as
// " bar".
func EngineFolderName(engineVersion string) string {
	if strings.HasPrefix(engineVersion, "2025.01") {
		return strings.ToLower("rel" + engineVersion[2:4] + engineVersion[5:7] + "." + engineVersion)
	}
	if isRecoilTag(engineVersion) {
		return strings.ToLower("recoil_" + engineVersion)
	}
	return strings.ToLower(strings.ReplaceAll(engineVersion, " BAR105", " bar"))
}

// isRecoilTag tells a RecoilEngine release tag ("2026.07.04") from an older Spring engine version
// ("105.1.1-2590-gb9462a0 BAR105").
func isRecoilTag(engineVersion string) bool {
	return strings.HasPrefix(engineVersion, "2")
}

// engineURL returns urlTemplate with the placeholders of Config.EngineCDNURL replaced for engineVersion. The tag is
// escaped for a query and a path alike: a space as %20, not +. BAR's CDN and RecoilEngine's releases name the
// platforms as Go does (linux, windows).
func (p *Player) engineURL(urlTemplate string, engineVersion string) string {
	tag := strings.ReplaceAll(url.QueryEscape(engineVersion), "+", "%20")
	return strings.NewReplacer("{platform}", runtime.GOOS, "{tag}", tag).Replace(urlTemplate)
}

// findOrDownloadEngine returns the path of spring-headless in the data dir's engine/<EngineFolderName>, downloaded if
// missing.
func (p *Player) findOrDownloadEngine(ctx context.Context, engineVersion string) (string, error) {
	engineDir := filepath.Join(p.dataDir, "engine", EngineFolderName(engineVersion))
	engine := filepath.Join(engineDir, springHeadless)
	if isFile(engine) {
		return engine, nil
	}
	if _, err := os.Stat(engineDir); err == nil {
		return "", fmt.Errorf("%s exists but has no %s; remove it to download it again", engineDir, springHeadless)
	}
	if err := p.downloadEngine(ctx, engineVersion, engineDir); err != nil {
		return "", err
	}
	return engine, nil
}

// downloadEngine downloads the engine archive and unpacks it into engineDir, which must not exist yet. It unpacks next
// to engineDir and renames at the end, so an interrupted download leaves no half engine behind.
func (p *Player) downloadEngine(ctx context.Context, engineVersion string, engineDir string) error {
	sevenZip, err := p.findSevenZip()
	if err != nil {
		return err
	}

	urls := []string{
		p.engineURL(p.config.EngineCDNURL, engineVersion),
	}
	if isRecoilTag(engineVersion) && p.config.EngineReleaseURL != "" {
		urls = append(urls, p.engineURL(p.config.EngineReleaseURL, engineVersion))
	}

	if err := os.MkdirAll(filepath.Dir(engineDir), dirPermissions); err != nil {
		return err
	}

	tempDir, err := os.MkdirTemp(filepath.Dir(engineDir), filepath.Base(engineDir)+"-*")
	if err != nil {
		return err
	}
	defer os.RemoveAll(tempDir) //nolint:errcheck

	archive := filepath.Join(tempDir, "engine.7z")
	var downloadErrs []error
	for _, engineURL := range urls {
		err := p.downloadFile(ctx, engineURL, archive)
		if err == nil {
			downloadErrs = nil
			break
		}
		downloadErrs = append(downloadErrs, fmt.Errorf("GET %s: %w", engineURL, err))
	}
	if len(downloadErrs) != 0 {
		return fmt.Errorf("downloading engine %s (you can unpack it into %s): %w",
			engineVersion, engineDir, errors.Join(downloadErrs...))
	}

	unpacked := filepath.Join(tempDir, "engine")

	//nolint:gosec // sevenZip is the configured or found 7-Zip, the paths are the package's own.
	output, err := exec.CommandContext(ctx, sevenZip, "x", "-y", "-o"+unpacked, archive).CombinedOutput()
	if err != nil {
		return fmt.Errorf("unpacking %s: %w\n%s", archive, err, output)
	}

	if !isFile(filepath.Join(unpacked, springHeadless)) {
		return fmt.Errorf("the archive of engine %s has no %s", engineVersion, springHeadless)
	}

	// 7-Zip may create it private, like the temporary folder around it.
	if err := os.Chmod(unpacked, dirPermissions); err != nil {
		return err
	}
	return os.Rename(unpacked, engineDir)
}

// findSevenZip returns Config.SevenZip, or the first of sevenZipNames on PATH.
func (p *Player) findSevenZip() (string, error) {
	if p.config.SevenZip != "" {
		return p.config.SevenZip, nil
	}
	for _, name := range sevenZipNames {
		if sevenZip, err := exec.LookPath(name); err == nil {
			return sevenZip, nil
		}
	}
	return "", errors.New("unpacking the engine needs 7-Zip: install 7z (p7zip-full or 7zip)")
}

// downloadFile downloads fileURL to filePath, which it creates or truncates.
func (p *Player) downloadFile(ctx context.Context, fileURL string, filePath string) error {
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, fileURL, nil)
	if err != nil {
		return err
	}

	p.reportDownload(fileURL)
	response, err := p.httpClient.Do(request)
	if err != nil {
		return err
	}
	defer response.Body.Close() //nolint:errcheck

	if response.StatusCode != http.StatusOK {
		return fmt.Errorf("unexpected HTTP status %s", response.Status)
	}

	//nolint:gosec // filePath is in a temporary folder of the package; see filePermissions.
	file, err := os.OpenFile(filePath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, filePermissions)
	if err != nil {
		return err
	}
	_, err = io.Copy(file, response.Body)
	return errors.Join(err, file.Close())
}

// isFile reports whether filePath is a regular file, following links.
func isFile(filePath string) bool {
	info, err := os.Stat(filePath)
	return err == nil && info.Mode().IsRegular()
}
