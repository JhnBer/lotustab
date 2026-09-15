package app

import (
	"context"
	"fmt"
	"io"
	"log"
	"strings"

	"github.com/wailsapp/wails/v3/pkg/application"
	"github.com/wailsapp/wails/v3/pkg/updater"
	"github.com/wailsapp/wails/v3/pkg/updater/providers/github"
)

const githubReleasesRepo = "JhnBer/lodaap"

var _ updater.Provider = (*liveGitHubProvider)(nil)

// InitUpdater configures app.Updater against GitHub Releases.
// token is called on every Check/Download so a saved or deleted PAT applies
// without restarting the app.
func InitUpdater(appl *application.App, version string, token func() string) error {
	if appl == nil {
		return fmt.Errorf("application is not running")
	}
	if err := appl.Updater.Init(updater.Config{
		CurrentVersion: version,
		Window:         updater.WindowNone,
		Providers:      []updater.Provider{&liveGitHubProvider{token: token}},
	}); err != nil {
		return err
	}
	log.Printf("updater: github %s (version %s)", githubReleasesRepo, version)
	return nil
}

// liveGitHubProvider rebuilds the GitHub client with the current keychain
// token on each request so Save/Delete take effect immediately.
type liveGitHubProvider struct {
	token func() string
}

func (p *liveGitHubProvider) Name() string { return "github" }

func (p *liveGitHubProvider) currentToken() string {
	if p == nil || p.token == nil {
		return ""
	}
	return p.token()
}

func (p *liveGitHubProvider) inner() (updater.Provider, error) {
	cfg := github.Config{
		Repository:    githubReleasesRepo,
		ChecksumAsset: "SHA256SUMS",
		AssetMatcher:  matchLotusAsset,
	}
	if tok := p.currentToken(); tok != "" {
		cfg.Token = tok
	}
	return github.New(cfg)
}

func (p *liveGitHubProvider) Check(ctx context.Context, req updater.CheckRequest) (*updater.Release, error) {
	inner, err := p.inner()
	if err != nil {
		return nil, err
	}
	auth := "public"
	if p.currentToken() != "" {
		auth = "token"
	}
	log.Printf("updater: github %s (version %s, auth %s)", githubReleasesRepo, req.CurrentVersion, auth)
	return inner.Check(ctx, req)
}

func (p *liveGitHubProvider) Download(ctx context.Context, r *updater.Release, dst io.Writer, onProgress func(written, total int64)) error {
	inner, err := p.inner()
	if err != nil {
		return err
	}
	return inner.Download(ctx, r, dst, onProgress)
}

// matchLotusAsset picks an installable GitHub asset for the running platform.
// DefaultAssetMatcher misses darwin universal builds (no arm64/amd64 in the
// filename) and can select a .dmg, which the updater cannot unpack.
func matchLotusAsset(req updater.CheckRequest, assets []github.ReleaseAsset) int {
	plat := strings.ToLower(req.Platform)
	arch := strings.ToLower(req.Arch)

	best := -1
	bestRank := 99
	for i, a := range assets {
		name := strings.ToLower(a.Name)
		if skipReleaseAsset(name) {
			continue
		}
		if plat != "" && !strings.Contains(name, plat) {
			continue
		}
		if arch != "" && !archMatches(name, plat, arch) {
			continue
		}
		rank := assetRank(name)
		if rank < bestRank {
			best = i
			bestRank = rank
		}
	}
	return best
}

func skipReleaseAsset(name string) bool {
	if strings.HasSuffix(name, ".sig") || strings.HasSuffix(name, ".asc") || strings.HasSuffix(name, ".dmg") {
		return true
	}
	if strings.Contains(name, "-installer.") || strings.Contains(name, "_installer.") || name == "installer.exe" {
		return true
	}
	return isChecksumAssetName(name)
}

func archMatches(name, plat, arch string) bool {
	if containsArchToken(name, arch) {
		return true
	}
	return plat == "darwin" && strings.Contains(name, "universal")
}

func containsArchToken(name, arch string) bool {
	if strings.Contains(name, arch) {
		return true
	}
	if arch == "amd64" && (strings.Contains(name, "x86_64") || strings.Contains(name, "x64")) {
		return true
	}
	if arch == "arm64" && strings.Contains(name, "aarch64") {
		return true
	}
	if arch == "386" {
		if strings.Contains(name, "x86_64") || strings.Contains(name, "x64") || strings.Contains(name, "amd64") {
			return false
		}
		return strings.Contains(name, "i386") || strings.Contains(name, "ia32") || strings.Contains(name, "x86")
	}
	return false
}

func isChecksumAssetName(lowerName string) bool {
	for _, ext := range []string{".sha256", ".sha512", ".sums", ".checksum", ".checksums"} {
		if strings.HasSuffix(lowerName, ext) {
			return true
		}
	}
	for _, token := range []string{"checksums", "sha256sums", "sha512sums"} {
		if lowerName == token || strings.HasPrefix(lowerName, token+".") {
			return true
		}
	}
	return false
}

func assetRank(name string) int {
	switch {
	case strings.HasSuffix(name, ".zip"):
		return 0
	case strings.HasSuffix(name, ".tar.gz"), strings.HasSuffix(name, ".tgz"):
		return 1
	case strings.HasSuffix(name, ".exe"):
		return 2
	default:
		return 3
	}
}
