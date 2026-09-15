package app

import "testing"

func TestReleaseIsNewer(t *testing.T) {
	cases := []struct {
		release, current string
		want             bool
	}{
		{"0.0.9", "0.0.8", true},
		{"v0.0.9", "0.0.8", true},
		{"0.0.8", "0.0.8", false},
		{"v0.0.8", "v0.0.8", false},
		{"0.0.7", "0.0.8", false},
		{"1.0.0", "0.9.9", true},
		{"1.0.0-rc.1", "1.0.0", false},
		{"1.0.0", "1.0.0-rc.1", true},
		{"0.0.9", "dev", false},
		{"v0.0.9", "dev", false},
		{"dev", "0.0.8", false},
		{"", "0.0.8", false},
		{"0.0.9", "", false},
	}
	for _, tc := range cases {
		got := releaseIsNewer(tc.release, tc.current)
		if got != tc.want {
			t.Fatalf("releaseIsNewer(%q, %q) = %v, want %v", tc.release, tc.current, got, tc.want)
		}
	}
}
