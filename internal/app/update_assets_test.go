package app

import (
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
