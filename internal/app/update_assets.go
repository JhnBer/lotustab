package app

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"strings"

	"github.com/wailsapp/wails/v3/pkg/application"
	"github.com/wailsapp/wails/v3/pkg/updater"
	"github.com/wailsapp/wails/v3/pkg/updater/providers/github"
)

const (
	githubReleasesRepo = "JhnBer/lotustab"
	checksumAssetName  = "SHA256SUMS"
	githubAPIBase      = "https://api.github.com"
)

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
//
// Private release assets are fetched from the releases assets API. The
// browser_download_url that the stock provider uses answers 404 for a
// private repository even when the asset is listed.
type liveGitHubProvider struct {
	token   func() string
	apiBase string
	client  *http.Client
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
		Repository:   githubReleasesRepo,
		AssetMatcher: matchLotusAsset,
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
	rel, err := inner.Check(ctx, req)
	if err != nil || rel == nil {
		return rel, err
	}
	if err := p.attachChecksum(ctx, rel); err != nil {
		return nil, err
	}
	return rel, nil
}

func (p *liveGitHubProvider) Download(ctx context.Context, r *updater.Release, dst io.Writer, onProgress func(written, total int64)) error {
	id, err := releaseAssetID(r)
	if err != nil {
		return err
	}
	body, err := p.openAsset(ctx, id)
	if err != nil {
		return fmt.Errorf("github: download: %w", err)
	}
	defer body.Close()

	total := r.Artifact.Size
	written := int64(0)
	buf := make([]byte, 64*1024)
	for {
		n, rerr := body.Read(buf)
		if n > 0 {
			if _, werr := dst.Write(buf[:n]); werr != nil {
				return werr
			}
			written += int64(n)
			if onProgress != nil {
				onProgress(written, total)
			}
		}
		if rerr == io.EOF {
			return nil
		}
		if rerr != nil {
			return rerr
		}
	}
}

func (p *liveGitHubProvider) attachChecksum(ctx context.Context, rel *updater.Release) error {
	tag, _ := rel.Metadata["github.release.tag"].(string)
	if tag == "" || rel.Artifact.Filename == "" {
		return nil
	}
	id, err := p.checksumAssetID(ctx, tag)
	if err != nil {
		return fmt.Errorf("github: load checksum sidecar: %w", err)
	}
	if id == 0 {
		return nil
	}
	body, err := p.openAsset(ctx, id)
	if err != nil {
		return fmt.Errorf("github: load checksum sidecar: %w", err)
	}
	defer body.Close()
	raw, err := io.ReadAll(io.LimitReader(body, 1<<20))
	if err != nil {
		return fmt.Errorf("github: load checksum sidecar: %w", err)
	}
	digest, err := parseChecksumLine(string(raw), rel.Artifact.Filename)
	if err != nil {
		return fmt.Errorf("github: load checksum sidecar: %w", err)
	}
	if digest != nil {
		rel.Verification = &updater.Verification{
			DigestAlgo: "sha256",
			Digest:     digest,
		}
	}
	return nil
}

func (p *liveGitHubProvider) checksumAssetID(ctx context.Context, tag string) (int64, error) {
	endpoint := fmt.Sprintf("%s/repos/%s/releases/tags/%s", p.baseURL(), githubReleasesRepo, url.PathEscape(tag))
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return 0, err
	}
	req.Header.Set("Accept", "application/vnd.github+json")
	p.setAPIHeaders(req)
	resp, err := p.do(req)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		snippet, _ := io.ReadAll(io.LimitReader(resp.Body, 512))
		return 0, fmt.Errorf("release %s: HTTP %d: %s", tag, resp.StatusCode, strings.TrimSpace(string(snippet)))
	}
	var rel struct {
		Assets []struct {
			ID   int64  `json:"id"`
			Name string `json:"name"`
		} `json:"assets"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&rel); err != nil {
		return 0, err
	}
	for _, asset := range rel.Assets {
		if asset.Name == checksumAssetName {
			return asset.ID, nil
		}
	}
	return 0, nil
}

func (p *liveGitHubProvider) openAsset(ctx context.Context, id int64) (io.ReadCloser, error) {
	endpoint := fmt.Sprintf("%s/repos/%s/releases/assets/%d", p.baseURL(), githubReleasesRepo, id)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/octet-stream")
	p.setAPIHeaders(req)
	resp, err := p.do(req)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		defer resp.Body.Close()
		snippet, _ := io.ReadAll(io.LimitReader(resp.Body, 512))
		return nil, fmt.Errorf("asset %d: HTTP %d: %s", id, resp.StatusCode, strings.TrimSpace(string(snippet)))
	}
	return resp.Body, nil
}

func (p *liveGitHubProvider) setAPIHeaders(req *http.Request) {
	req.Header.Set("X-GitHub-Api-Version", "2022-11-28")
	if tok := p.currentToken(); tok != "" {
		req.Header.Set("Authorization", "Bearer "+tok)
	}
}

func (p *liveGitHubProvider) baseURL() string {
	if p != nil && p.apiBase != "" {
		return strings.TrimRight(p.apiBase, "/")
	}
	return githubAPIBase
}

func (p *liveGitHubProvider) httpClient() *http.Client {
	if p != nil && p.client != nil {
		return p.client
	}
	return http.DefaultClient
}

// do follows redirects and drops Authorization when the host changes, so the
// PAT is not sent to the release-asset CDN.
func (p *liveGitHubProvider) do(req *http.Request) (*http.Response, error) {
	base := p.httpClient()
	client := *base
	prev := client.CheckRedirect
	client.CheckRedirect = func(r *http.Request, via []*http.Request) error {
		if len(via) > 0 && !strings.EqualFold(via[len(via)-1].URL.Host, r.URL.Host) {
			r.Header.Del("Authorization")
		}
		if prev != nil {
			return prev(r, via)
		}
		if len(via) >= 10 {
			return errors.New("stopped after 10 redirects")
		}
		return nil
	}
	return client.Do(req)
}

func releaseAssetID(rel *updater.Release) (int64, error) {
	if rel == nil || rel.Metadata == nil {
		return 0, errors.New("github: release missing asset id")
	}
	switch id := rel.Metadata["github.asset.id"].(type) {
	case int64:
		if id > 0 {
			return id, nil
		}
	case int:
		if id > 0 {
			return int64(id), nil
		}
	case float64:
		if id > 0 {
			return int64(id), nil
		}
	}
	return 0, errors.New("github: release missing asset id")
}

// parseChecksumLine extracts the digest for target from a shasum listing.
// Each line is "<hex>  <filename>".
func parseChecksumLine(body, target string) ([]byte, error) {
	for _, line := range strings.Split(body, "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		fields := strings.Fields(line)
		if len(fields) < 2 {
			continue
		}
		name := strings.TrimPrefix(fields[len(fields)-1], "*")
		name = strings.TrimPrefix(name, "./")
		if name != target {
			continue
		}
		digest, err := hex.DecodeString(fields[0])
		if err != nil {
			return nil, fmt.Errorf("malformed digest for %s: %w", target, err)
		}
		return digest, nil
	}
	return nil, nil
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
