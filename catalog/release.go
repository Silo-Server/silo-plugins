package catalog

import (
	"fmt"
	"net/url"
	"regexp"
	"strings"

	"golang.org/x/mod/semver"
)

var (
	repoPattern   = regexp.MustCompile(`^[A-Za-z0-9-]+/[A-Za-z0-9._-]+$`)
	tagPattern    = regexp.MustCompile(`^v(0|[1-9]\d*)\.(0|[1-9]\d*)\.(0|[1-9]\d*)(-[0-9A-Za-z.-]+)?$`)
	commitPattern = regexp.MustCompile(`^[0-9a-f]{40}$`)
)

// ValidateInputs checks the caller-supplied repository and tag before either
// goes into a URL. Both come from a dispatch payload; a tag such as
// "../../other/repo/..." would otherwise make the release and manifest fetches
// resolve to another repository.
func ValidateInputs(repo, tag string) error {
	if !repoPattern.MatchString(repo) {
		return fmt.Errorf("repository %q is not in owner/name form", repo)
	}
	// The pattern keeps URL metacharacters out; semver.IsValid also rejects
	// suffixes CheckVersion could not order, such as a leading-zero number.
	if !tagPattern.MatchString(tag) || !semver.IsValid(tag) {
		return fmt.Errorf("tag %q is not a v<major>.<minor>.<patch> release tag", tag)
	}
	return nil
}

// ValidateCommit checks a commit SHA before it goes into a URL.
func ValidateCommit(sha string) error {
	if !commitPattern.MatchString(sha) {
		return fmt.Errorf("commit %q is not a full lowercase SHA-1", sha)
	}
	return nil
}

// VersionFromTag returns the catalog version a validated tag names.
func VersionFromTag(tag string) string {
	return strings.TrimPrefix(tag, "v")
}

// FindPackage returns the catalog entry for pluginID, or nil.
func FindPackage(index RepositoryIndex, pluginID string) *CatalogPackage {
	for i := range index.Plugins {
		if packagePluginID(index.Plugins[i]) == pluginID {
			return &index.Plugins[i]
		}
	}
	return nil
}

// CheckVersion refuses a release that does not advance the plugin's catalog
// version, so a dispatch cannot replace the current entry with an older
// release. alreadyListed reports a re-run for the listed version, which the
// caller treats as a successful no-op. allowRollback, set only for a
// deliberate manual run, accepts an older version and rewrites an equal one.
func CheckVersion(current *CatalogPackage, version string, allowRollback bool) (alreadyListed bool, err error) {
	if current == nil || current.Manifest == nil {
		return false, nil
	}
	listed := current.Manifest.GetVersion()
	switch cmp := semver.Compare("v"+version, "v"+listed); {
	case cmp > 0, allowRollback:
		return false, nil
	case cmp == 0:
		return true, nil
	default:
		return false, fmt.Errorf("version %s is older than the listed version %s", version, listed)
	}
}

// CheckRelease refuses a release that is not the published, final release for
// tag.
func CheckRelease(tag string, release Release) error {
	if release.TagName != tag {
		return fmt.Errorf("release tag %q does not match requested tag %q", release.TagName, tag)
	}
	if release.Draft {
		return fmt.Errorf("release %s is a draft", tag)
	}
	if release.Prerelease {
		return fmt.Errorf("release %s is a prerelease", tag)
	}
	return nil
}

// ReleaseAssets are the release assets the catalog records.
type ReleaseAssets struct {
	ChecksumsURL string
	BundleURL    string
	// Binaries maps "<os>/<arch>" to the binary's download URL.
	Binaries map[string]string
}

// ParseReleaseAssets picks the catalogued assets out of a release of repo.
// Every catalogued URL must be the asset's own download URL in that
// repository and tag, so an entry cannot point at another repository or host.
func ParseReleaseAssets(repo string, release Release) (ReleaseAssets, error) {
	assets := ReleaseAssets{Binaries: map[string]string{}}
	prefix := "https://github.com/" + repo + "/releases/download/" + url.PathEscape(release.TagName) + "/"
	for _, asset := range release.Assets {
		platform, isBinary := "", false
		switch {
		case asset.Name == "checksums.txt", asset.Name == BundleAssetName:
		case strings.HasPrefix(asset.Name, "plugin-"):
			platform, isBinary = platformKeyFromAssetName(asset.Name)
			if !isBinary {
				continue
			}
		default:
			continue
		}
		if want := prefix + url.PathEscape(asset.Name); asset.BrowserDownloadURL != want {
			return ReleaseAssets{}, fmt.Errorf("asset %q has download URL %q, want %q", asset.Name, asset.BrowserDownloadURL, want)
		}
		switch {
		case isBinary:
			assets.Binaries[platform] = asset.BrowserDownloadURL
		case asset.Name == BundleAssetName:
			assets.BundleURL = asset.BrowserDownloadURL
		default:
			assets.ChecksumsURL = asset.BrowserDownloadURL
		}
	}
	return assets, nil
}
