package main

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"maps"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"
	"time"

	"github.com/Silo-Server/silo-plugins/catalog"
)

const (
	githubAPIVersion = "2022-11-28"

	maxBundleBytes = 256 << 10
	maxBinaryBytes = 512 << 20
)

func main() {
	var repo, tag, manifestPath string
	var allowRollback, historyOnly, withdraw bool

	flag.StringVar(&repo, "repo", "", "GitHub repository in owner/name form")
	flag.StringVar(&tag, "tag", "", "Release tag to publish, v<major>.<minor>.<patch>")
	flag.StringVar(&manifestPath, "manifest", "manifest.json", "Path to the catalog manifest")
	flag.BoolVar(&allowRollback, "allow-rollback", false, "Make an older release current (deliberate rollback only)")
	flag.BoolVar(&historyOnly, "history-only", false, "Verify and record a signed release without changing the current catalog")
	flag.BoolVar(&withdraw, "withdraw", false, "Record a withdrawal in the current catalog")
	flag.Parse()

	if err := run(context.Background(), repo, tag, manifestPath, allowRollback, historyOnly, withdraw); err != nil {
		exitf("%v", err)
	}
}

// run applies every check that needs no network first, then fetches the
// release, ties it to one commit (the signed one, or the tag's commit for an
// unsigned release), reads manifest.json at that commit, and only then writes
// the catalog.
func run(ctx context.Context, repo, tag, manifestPath string, allowRollback, historyOnly, withdraw bool) error {
	releasePath, err := catalog.ReleasePath(filepath.Dir(manifestPath), repo, tag)
	if err != nil {
		return fmt.Errorf("refuse release: %w", err)
	}
	pluginID, err := catalog.OfficialPluginID(repo)
	if err != nil {
		return err
	}

	index, err := loadIndex(manifestPath)
	if err != nil {
		return fmt.Errorf("load catalog: %w", err)
	}
	current := catalog.FindPackage(index, pluginID)
	version := catalog.VersionFromTag(tag)
	if withdraw {
		if historyOnly || allowRollback {
			return fmt.Errorf("-withdraw cannot be combined with -history-only or -allow-rollback")
		}
		if current == nil || current.Manifest.GetVersion() != version {
			if _, err := os.Stat(releasePath); err != nil {
				return fmt.Errorf("withdraw %s %s: %w", pluginID, version, err)
			}
		}
		return writeIndex(manifestPath, catalog.WithdrawRelease(index, pluginID, version))
	}
	if !historyOnly {
		if index.IsWithdrawn(pluginID, version) {
			return fmt.Errorf("refuse release: %s %s has been withdrawn", pluginID, version)
		}
		if _, err := catalog.CheckVersion(current, version, allowRollback); err != nil {
			return fmt.Errorf("refuse release: %w", err)
		}
	}

	token := os.Getenv("GITHUB_TOKEN")
	client := &http.Client{Timeout: 30 * time.Second}

	release, err := fetchRelease(ctx, client, token, repo, tag)
	if err != nil {
		return fmt.Errorf("fetch release: %w", err)
	}
	if err := catalog.CheckRelease(tag, release); err != nil {
		return fmt.Errorf("refuse release: %w", err)
	}
	assets, err := catalog.ParseReleaseAssets(repo, release)
	if err != nil {
		return fmt.Errorf("refuse release: %w", err)
	}
	if err := catalog.CheckSignatureRatchet(current, assets.BundleURL); err != nil {
		return fmt.Errorf("refuse release: %w", err)
	}
	if historyOnly && assets.BundleURL == "" {
		return fmt.Errorf("refuse backfill: %s %s has no signature bundle", pluginID, version)
	}

	// Binaries are tens of megabytes, so downloads get more time than API calls.
	downloadClient := &http.Client{Timeout: 5 * time.Minute}
	var commit string
	if assets.BundleURL == "" {
		commit, err = resolveTagCommit(ctx, client, token, repo, tag)
		if err != nil {
			return fmt.Errorf("resolve tag %s: %w", tag, err)
		}
	}
	digests, signedCommit, err := readReleaseBinaries(ctx, downloadClient, repo, version, assets)
	if err != nil {
		return fmt.Errorf("refuse release: %w", err)
	}
	if signedCommit != "" {
		commit = signedCommit
	}

	sourceManifest, err := fetchSourceManifest(ctx, client, token, repo, commit)
	if err != nil {
		return fmt.Errorf("fetch source manifest: %w", err)
	}
	if err := catalog.CheckOfficialRepo(sourceManifest.GetPluginId(), repo); err != nil {
		return fmt.Errorf("refuse release: %w", err)
	}
	pkg, err := catalog.BuildPackageFromRelease(repo, sourceManifest, release)
	if err != nil {
		return fmt.Errorf("build catalog package: %w", err)
	}
	pkg.Tag, pkg.SourceCommit = tag, commit
	for platform, digest := range digests {
		binary := pkg.Binaries[platform]
		binary.Checksum = digest
		pkg.Binaries[platform] = binary
	}

	created, err := publishDescriptor(releasePath, pkg)
	if err != nil {
		return err
	}
	if historyOnly {
		return nil
	}
	index = catalog.UpsertPackage(index, pkg)
	if err := writeIndex(manifestPath, index); err != nil {
		if created {
			err = errors.Join(err, os.Remove(releasePath))
		}
		return fmt.Errorf("write catalog: %w", err)
	}
	return nil
}

// readReleaseBinaries computes the digests from the actual assets and verifies
// signed releases against the bundle before returning their source commit.
func readReleaseBinaries(ctx context.Context, client *http.Client, repo, version string, assets catalog.ReleaseAssets) (map[string]string, string, error) {
	if len(assets.Binaries) == 0 {
		return nil, "", fmt.Errorf("release has no plugin binaries")
	}
	if assets.ChecksumsURL == "" {
		return nil, "", fmt.Errorf("release has no checksums.txt")
	}
	dir, err := os.MkdirTemp("", "silo-plugin-verify-")
	if err != nil {
		return nil, "", err
	}
	defer os.RemoveAll(dir)

	bundlePath := filepath.Join(dir, catalog.BundleAssetName)
	if assets.BundleURL != "" {
		if _, err := download(ctx, client, assets.BundleURL, bundlePath, maxBundleBytes); err != nil {
			return nil, "", fmt.Errorf("download %s: %w", catalog.BundleAssetName, err)
		}
	}
	checksumsPath := filepath.Join(dir, "checksums.txt")
	if _, err := download(ctx, client, assets.ChecksumsURL, checksumsPath, 64<<10); err != nil {
		return nil, "", fmt.Errorf("download checksums.txt: %w", err)
	}
	checksums, err := os.ReadFile(checksumsPath)
	if err != nil {
		return nil, "", err
	}

	platforms := slices.Sorted(maps.Keys(assets.Binaries))
	digests := make(map[string]string, len(platforms))
	var commit string
	for i, platform := range platforms {
		binaryPath := filepath.Join(dir, fmt.Sprintf("binary-%d", i))
		digest, err := download(ctx, client, assets.Binaries[platform], binaryPath, maxBinaryBytes)
		if err != nil {
			return nil, "", fmt.Errorf("download %s binary: %w", platform, err)
		}
		if err := catalog.CheckReleaseDigest(string(checksums), "plugin-"+strings.ReplaceAll(platform, "/", "-"), digest); err != nil {
			return nil, "", err
		}
		digests[platform] = digest
		if assets.BundleURL == "" {
			continue
		}
		cmd := exec.CommandContext(ctx, "gh", catalog.VerifyArgs(binaryPath, bundlePath, repo, version)...)
		cmd.Stderr = os.Stderr
		out, err := cmd.Output()
		if err != nil {
			return nil, "", fmt.Errorf("%s binary: gh attestation verify: %w", platform, err)
		}
		binaryCommit, err := catalog.CheckVerifyOutput(out)
		if err != nil {
			return nil, "", fmt.Errorf("%s binary: %w", platform, err)
		}
		if commit != "" && binaryCommit != commit {
			return nil, "", fmt.Errorf("%s binary was built from %s, other binaries from %s", platform, binaryCommit, commit)
		}
		commit = binaryCommit
		fmt.Fprintf(os.Stderr, "Verified the %s binary of %s v%s, built from %s.\n", platform, repo, version, commit)
	}
	return digests, commit, nil
}

// resolveTagCommit returns the commit a tag points at, following annotated
// tags. An unsigned release has no certificate naming its commit, so this
// pins manifest.json to one commit rather than a ref a branch could shadow.
func resolveTagCommit(ctx context.Context, client *http.Client, token, repo, tag string) (string, error) {
	var ref struct {
		Object gitObject `json:"object"`
	}
	if err := githubJSON(ctx, client, token, "https://api.github.com/repos/"+repo+"/git/ref/tags/"+tag, &ref); err != nil {
		return "", err
	}
	object := ref.Object
	for range 5 {
		if err := catalog.ValidateCommit(object.SHA); err != nil {
			return "", err
		}
		switch object.Type {
		case "commit":
			return object.SHA, nil
		case "tag":
			var annotated struct {
				Object gitObject `json:"object"`
			}
			if err := githubJSON(ctx, client, token, "https://api.github.com/repos/"+repo+"/git/tags/"+object.SHA, &annotated); err != nil {
				return "", err
			}
			object = annotated.Object
		default:
			return "", fmt.Errorf("tag points at a %s, not a commit", object.Type)
		}
	}
	return "", fmt.Errorf("tag is nested more than 5 annotated tags deep")
}

type gitObject struct {
	SHA  string `json:"sha"`
	Type string `json:"type"`
}

func download(ctx context.Context, client *http.Client, url, path string, limit int64) (string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return "", fmt.Errorf("build request: %w", err)
	}
	req.Header.Set("User-Agent", "silo-plugins-catalog-updater")
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("GET %s: %w", url, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("GET %s: status %d", url, resp.StatusCode)
	}
	file, err := os.OpenFile(path, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o600)
	if err != nil {
		return "", err
	}
	hash := sha256.New()
	written, err := io.Copy(io.MultiWriter(file, hash), io.LimitReader(resp.Body, limit+1))
	if closeErr := file.Close(); err == nil {
		err = closeErr
	}
	if err != nil {
		return "", fmt.Errorf("read %s: %w", url, err)
	}
	if written > limit {
		return "", fmt.Errorf("GET %s: larger than %d bytes", url, limit)
	}
	return hex.EncodeToString(hash.Sum(nil)), nil
}

func fetchRelease(ctx context.Context, client *http.Client, token, repo, tag string) (catalog.Release, error) {
	var release catalog.Release
	if err := githubJSON(ctx, client, token, "https://api.github.com/repos/"+repo+"/releases/tags/"+tag, &release); err != nil {
		return catalog.Release{}, err
	}
	return release, nil
}

// fetchSourceManifest reads manifest.json at a validated commit SHA.
func fetchSourceManifest(ctx context.Context, client *http.Client, token, repo, commit string) (*catalog.SourceManifest, error) {
	url := "https://raw.githubusercontent.com/" + repo + "/" + commit + "/manifest.json"
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("build request: %w", err)
	}
	req.Header.Set("User-Agent", "silo-plugins-catalog-updater")
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("GET %s: %w", url, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 4<<10))
		return nil, fmt.Errorf("GET %s: status %d: %s", url, resp.StatusCode, strings.TrimSpace(string(body)))
	}
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read source manifest: %w", err)
	}
	return catalog.DecodeSourceManifest(body)
}

func githubJSON(ctx context.Context, client *http.Client, token, url string, dest any) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return fmt.Errorf("build request: %w", err)
	}
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("User-Agent", "silo-plugins-catalog-updater")
	req.Header.Set("X-GitHub-Api-Version", githubAPIVersion)
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}

	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("GET %s: %w", url, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 4<<10))
		return fmt.Errorf("GET %s: status %d: %s", url, resp.StatusCode, strings.TrimSpace(string(body)))
	}
	if err := json.NewDecoder(resp.Body).Decode(dest); err != nil {
		return fmt.Errorf("decode %s: %w", url, err)
	}
	return nil
}

func loadIndex(path string) (catalog.RepositoryIndex, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return catalog.RepositoryIndex{}, fmt.Errorf("read %s: %w", path, err)
	}
	var index catalog.RepositoryIndex
	if len(bytes.TrimSpace(data)) == 0 {
		return index, nil
	}
	if err := json.Unmarshal(data, &index); err != nil {
		return catalog.RepositoryIndex{}, fmt.Errorf("decode %s: %w", path, err)
	}
	return index, nil
}

func writeIndex(path string, index catalog.RepositoryIndex) error {
	data, err := json.MarshalIndent(index, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal catalog: %w", err)
	}
	data = append(data, '\n')
	return writeCatalogFile(path, data, os.Rename)
}

func publishDescriptor(path string, pkg catalog.CatalogPackage) (bool, error) {
	data, err := json.MarshalIndent(pkg, "", "  ")
	if err != nil {
		return false, err
	}
	data = append(data, '\n')
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return false, err
	}
	if err := writeCatalogFile(path, data, os.Link); err != nil {
		if !errors.Is(err, os.ErrExist) {
			return false, err
		}
		existing, err := os.ReadFile(path)
		if err != nil {
			return false, err
		}
		if !bytes.Equal(existing, data) {
			return false, fmt.Errorf("refuse to change release descriptor %s", path)
		}
		return false, nil
	}
	return true, nil
}

func writeCatalogFile(path string, data []byte, publish func(string, string) error) error {
	file, err := os.CreateTemp(filepath.Dir(path), ".catalog-")
	if err != nil {
		return err
	}
	defer os.Remove(file.Name())
	if err := file.Chmod(0o644); err != nil {
		_ = file.Close()
		return err
	}
	_, writeErr := file.Write(data)
	if err := errors.Join(writeErr, file.Close()); err != nil {
		return err
	}
	return publish(file.Name(), path)
}

func exitf(format string, args ...any) {
	fmt.Fprintf(os.Stderr, format+"\n", args...)
	os.Exit(1)
}
