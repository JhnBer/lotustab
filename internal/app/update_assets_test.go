package app

import (
	"bytes"
	"context"
	"encoding/hex"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/wailsapp/wails/v3/pkg/updater"
	"github.com/wailsapp/wails/v3/pkg/updater/providers/github"
)

func TestMatchLotusAsset(t *testing.T) {
	assets := []github.ReleaseAsset{
		{Name: "SHA256SUMS"},
		{Name: "lotustab-darwin-universal-v0.0.9.dmg"},
		{Name: "lotustab-darwin-universal-v0.0.9.zip"},
		{Name: "lotustab-windows-amd64-v0.0.9.exe"},
		{Name: "lotustab-linux-amd64-v0.0.9.tar.gz"},
	}

	cases := []struct {
		plat, arch, want string
	}{
		{"darwin", "arm64", "lotustab-darwin-universal-v0.0.9.zip"},
		{"darwin", "amd64", "lotustab-darwin-universal-v0.0.9.zip"},
		{"windows", "amd64", "lotustab-windows-amd64-v0.0.9.exe"},
		{"linux", "amd64", "lotustab-linux-amd64-v0.0.9.tar.gz"},
		{"linux", "arm64", ""},
	}

	for _, tc := range cases {
		idx := matchLotusAsset(updater.CheckRequest{Platform: tc.plat, Arch: tc.arch}, assets)
		got := ""
		if idx >= 0 {
			got = assets[idx].Name
		}
		if got != tc.want {
			t.Fatalf("%s/%s: got %q, want %q", tc.plat, tc.arch, got, tc.want)
		}
	}
}

func TestMatchLotusAssetPrefersZipOverDmg(t *testing.T) {
	assets := []github.ReleaseAsset{
		{Name: "lotustab-darwin-universal-v1.0.0.dmg"},
		{Name: "lotustab-darwin-universal-v1.0.0.zip"},
	}
	idx := matchLotusAsset(updater.CheckRequest{Platform: "darwin", Arch: "arm64"}, assets)
	if idx != 1 {
		t.Fatalf("expected zip at index 1, got %d", idx)
	}
}

func TestParseChecksumLine(t *testing.T) {
	body := "" +
		"335637095b962a9ffb3f9d63705b8b62befcb21cfa0aedd1e4bc54fe5b79fc6e  app.dmg\n" +
		"1d84f09f5d4746e95ed83750c58c9eb9c0618772505bd8b650846f719a21cd7e  app.zip\n"
	got, err := parseChecksumLine(body, "app.zip")
	if err != nil {
		t.Fatal(err)
	}
	want, _ := hex.DecodeString("1d84f09f5d4746e95ed83750c58c9eb9c0618772505bd8b650846f719a21cd7e")
	if !bytes.Equal(got, want) {
		t.Fatalf("digest %x", got)
	}
}

func TestAttachChecksumAndDownloadUseAssetAPI(t *testing.T) {
	cdn := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "" {
			t.Errorf("CDN received Authorization")
		}
		switch r.URL.Path {
		case "/sums":
			io.WriteString(w, "1d84f09f5d4746e95ed83750c58c9eb9c0618772505bd8b650846f719a21cd7e  app.zip\n")
		case "/zip":
			io.WriteString(w, "zip-bytes")
		default:
			http.NotFound(w, r)
		}
	}))
	defer cdn.Close()

	api := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer secret" {
			t.Errorf("api auth: %q", r.Header.Get("Authorization"))
		}
		switch r.URL.Path {
		case "/repos/JhnBer/lotustab/releases/tags/v0.0.19":
			io.WriteString(w, `{"assets":[{"id":7,"name":"SHA256SUMS"},{"id":9,"name":"app.zip"}]}`)
		case "/repos/JhnBer/lotustab/releases/assets/7":
			if r.Header.Get("Accept") != "application/octet-stream" {
				t.Errorf("accept: %q", r.Header.Get("Accept"))
			}
			http.Redirect(w, r, cdn.URL+"/sums", http.StatusFound)
		case "/repos/JhnBer/lotustab/releases/assets/9":
			http.Redirect(w, r, cdn.URL+"/zip", http.StatusFound)
		default:
			http.NotFound(w, r)
		}
	}))
	defer api.Close()

	p := &liveGitHubProvider{token: func() string { return "secret" }, apiBase: api.URL}
	rel := &updater.Release{
		Artifact: updater.Artifact{Filename: "app.zip", Size: 9},
		Metadata: map[string]any{
			"github.release.tag": "v0.0.19",
			"github.asset.id":    int64(9),
		},
	}
	if err := p.attachChecksum(context.Background(), rel); err != nil {
		t.Fatal(err)
	}
	if rel.Verification == nil || rel.Verification.DigestAlgo != "sha256" {
		t.Fatalf("verification: %+v", rel.Verification)
	}

	var buf bytes.Buffer
	var written, total int64
	err := p.Download(context.Background(), rel, &buf, func(w, tot int64) {
		written, total = w, tot
	})
	if err != nil {
		t.Fatal(err)
	}
	if buf.String() != "zip-bytes" || written != 9 || total != 9 {
		t.Fatalf("download %q written %d/%d", buf.String(), written, total)
	}
}

func TestAboutServiceInfoDev(t *testing.T) {
	s := NewAboutService("dev", true, nil)
	info := s.Info()
	if info.Version != "dev" || !info.UpdatesEnabled || info.HasGitHubToken {
		t.Fatalf("unexpected info: %+v", info)
	}
	if _, err := s.Check(); err == nil {
		t.Fatal("expected check to fail without a running app")
	}
}
