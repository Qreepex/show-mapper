// Package updater implements the optional self-update feature: check the
// configured GitHub repository for a newer release and (on explicit user
// action) download + install it in place. Pure Go, no CGO.
//
// The release asset names produced by .github/workflows/release.yml
// (show-mapper_<ver>_<os>_<arch>.* + checksums.txt) are discovered and
// checksum-verified by github.com/rhysd/go-github-selfupdate.
//
// The feature activates only when `updates.repo` ("owner/name") is set in the
// config — enable it once the module path points at a real public repo.
package updater

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/blang/semver"
	selfupdate "github.com/rhysd/go-github-selfupdate/selfupdate"

	"github.com/Qreepex/show-mapper/internal/version"
)

// UpdateStatus is the (cacheable, JSON/WS-broadcast) state of update knowledge.
type UpdateStatus struct {
	Current       string     `json:"current"`    // running version ("dev" for dev builds)
	Repo          string     `json:"repo"`       // configured update source ("" = feature off)
	Configured    bool       `json:"configured"` // repo set?
	LatestVersion string     `json:"latestVersion,omitempty"`
	LatestURL     string     `json:"latestURL,omitempty"`    // release page
	ReleaseNotes  string     `json:"releaseNotes,omitempty"` // truncated
	PublishedAt   *time.Time `json:"publishedAt,omitempty"`
	Available     bool       `json:"available"` // newer than Current?
	CheckedAt     time.Time  `json:"checkedAt"`
	Error         string     `json:"error,omitempty"` // last check failure (Release asset missing, offline, …)

	// assetURL/hasAsset are the selected release asset, kept for Apply().
	assetURL string
	hasAsset bool
}

const maxNotes = 2000

// BaseStatus describes the configured-but-unchecked state.
func BaseStatus(repo string) UpdateStatus {
	return UpdateStatus{
		Current:    version.Version,
		Repo:       repo,
		Configured: repo != "",
	}
}

// Check queries GitHub for the latest release with a matching OS/arch asset.
func Check(repo string) UpdateStatus {
	st := BaseStatus(repo)
	st.CheckedAt = time.Now()
	if repo == "" {
		st.Error = "updates.repo not configured (hint: set it to your GitHub \"owner/repo\")"
		return st
	}

	latest, found, err := selfupdate.DetectLatest(repo)
	if err != nil {
		st.Error = err.Error()
		return st
	}
	if !found {
		st.Error = fmt.Sprintf("no released asset matching this OS/arch detected in %s", repo)
		return st
	}

	st.LatestVersion = latest.Version.String()
	st.LatestURL = latest.URL
	st.ReleaseNotes = latest.ReleaseNotes
	if len(st.ReleaseNotes) > maxNotes {
		st.ReleaseNotes = st.ReleaseNotes[:maxNotes] + "…"
	}
	st.PublishedAt = latest.PublishedAt
	if cur := currentSemver(version.Version); cur != nil {
		st.Available = latest.Version.GT(*cur)
	}
	st.assetURL = latest.AssetURL
	st.hasAsset = latest.AssetByteSize > 0 && latest.AssetURL != ""
	return st
}

// Apply downloads and swaps in the currently detected release binary.
// On Windows the running .exe is renamed aside (show-mapper.old.exe) so the
// new one can take its place — delete the *.old* file later. Restart the
// app to run the new version.
func Apply(st UpdateStatus) error {
	if !st.hasAsset || st.assetURL == "" {
		return fmt.Errorf("no downloadable asset known — run a check first")
	}
	exe, err := os.Executable()
	if err != nil {
		return err
	}
	exe, err = filepath.EvalSymlinks(exe)
	if err != nil {
		return err
	}
	return selfupdate.UpdateTo(st.assetURL, exe)
}

// ---------------------------------------------------------------------------

func currentSemver(v string) *semver.Version {
	sv, err := semver.ParseTolerant(strings.TrimSpace(v))
	if err != nil {
		return nil // dev/nightly builds don't compare
	}
	return &sv
}
