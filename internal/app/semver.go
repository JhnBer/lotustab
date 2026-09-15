package app

import (
	"strings"

	xsemver "golang.org/x/mod/semver"
)

// releaseIsNewer reports whether release is a valid semver strictly greater
// than current. Invalid values (including "dev") never count as an update.
func releaseIsNewer(release, current string) bool {
	rel := canonicalSemver(release)
	cur := canonicalSemver(current)
	if !xsemver.IsValid(rel) || !xsemver.IsValid(cur) {
		return false
	}
	return xsemver.Compare(rel, cur) > 0
}

func canonicalSemver(s string) string {
	s = strings.TrimSpace(s)
	if s == "" {
		return ""
	}
	if s[0] == 'v' || s[0] == 'V' {
		s = s[1:]
	}
	if s == "" {
		return ""
	}
	return "v" + s
}
