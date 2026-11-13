package catalog

import (
	"encoding/json"
	"slices"
	"testing"

	pluginv1 "github.com/Silo-Server/silo-plugin-sdk/pkg/pluginproto/silo/plugin/v1"
)

func TestBuildPackageFromRelease_MinimalManifestAndAssets(t *testing.T) {
	source := &SourceManifest{
		PluginId:       "silo.tmdb",
		Version:        "1.2.3",
		SiloApiVersion: "v1",
		Presentation:   catalogTestPresentation("https://github.com/Silo-Server/silo-plugin-metadata-tmdb"),
		Capabilities: []*pluginv1.CapabilityDescriptor{
			{
				Type:        "metadata_provider.v1",
				Id:          "tmdb",
				DisplayName: "TMDB",
				Description: "TMDB metadata provider for Silo.",
			},
		},
	}

	release := Release{
		TagName: "v1.2.3",
		Assets: []Asset{
			{Name: "plugin-linux-amd64", BrowserDownloadURL: "https://github.com/Silo-Server/silo-plugin-metadata-tmdb/releases/download/v1.2.3/plugin-linux-amd64"},
			{Name: "plugin-linux-arm64", BrowserDownloadURL: "https://github.com/Silo-Server/silo-plugin-metadata-tmdb/releases/download/v1.2.3/plugin-linux-arm64"},
			{Name: "checksums.txt", BrowserDownloadURL: "https://github.com/Silo-Server/silo-plugin-metadata-tmdb/releases/download/v1.2.3/checksums.txt"},
			{Name: "notes.txt", BrowserDownloadURL: "https://github.com/Silo-Server/silo-plugin-metadata-tmdb/releases/download/v1.2.3/notes.txt"},
		},
	}

	pkg, err := BuildPackageFromRelease("Silo-Server/silo-plugin-metadata-tmdb", source, release)
	if err != nil {
		t.Fatalf("BuildPackageFromRelease() error = %v", err)
	}

	if pkg.RepoURL != "https://github.com/Silo-Server/silo-plugin-metadata-tmdb" {
		t.Fatalf("RepoURL = %q", pkg.RepoURL)
	}
	if pkg.Manifest.GetPluginId() != "silo.tmdb" {
		t.Fatalf("PluginID = %q", pkg.Manifest.GetPluginId())
	}
	if pkg.Manifest.GetVersion() != "1.2.3" {
		t.Fatalf("Version = %q", pkg.Manifest.GetVersion())
	}
	if pkg.Manifest.GetChecksum() != "" {
		t.Fatalf("Checksum = %q, want empty catalog checksum", pkg.Manifest.GetChecksum())
	}
	if got := len(pkg.Binaries); got != 2 {
		t.Fatalf("Binaries length = %d, want 2", got)
	}
	if pkg.Binaries["linux/amd64"].URL == "" {
		t.Fatal("expected linux/amd64 binary URL")
	}
}

func TestBuildPackageFromRelease_TagWinsOverManifestVersion(t *testing.T) {
	source := &SourceManifest{
		PluginId:       "silo.tmdb",
		Version:        "1.2.2",
		SiloApiVersion: "v1",
		Presentation:   catalogTestPresentation("https://github.com/Silo-Server/silo-plugin-metadata-tmdb"),
		Capabilities: []*pluginv1.CapabilityDescriptor{
			{Type: "metadata_provider.v1", Id: "tmdb"},
		},
	}
	release := Release{
		TagName: "v1.2.3",
		Assets: []Asset{
			{Name: "plugin-linux-amd64", BrowserDownloadURL: "https://github.com/Silo-Server/silo-plugin-metadata-tmdb/releases/download/v1.2.3/plugin-linux-amd64"},
			{Name: "checksums.txt", BrowserDownloadURL: "https://github.com/Silo-Server/silo-plugin-metadata-tmdb/releases/download/v1.2.3/checksums.txt"},
		},
	}

	pkg, err := BuildPackageFromRelease("Silo-Server/silo-plugin-metadata-tmdb", source, release)
	if err != nil {
		t.Fatalf("BuildPackageFromRelease() error = %v", err)
	}
	if pkg.Manifest.GetVersion() != "1.2.3" {
		t.Fatalf("Version = %q, want %q (tag should override manifest)", pkg.Manifest.GetVersion(), "1.2.3")
	}
}

func TestBuildPackageFromRelease_RequiresCompletePresentation(t *testing.T) {
	source := &SourceManifest{
		PluginId:       "silo.tmdb",
		Version:        "1.2.3",
		SiloApiVersion: "v1",
		Capabilities: []*pluginv1.CapabilityDescriptor{
			{Type: "metadata_provider.v1", Id: "tmdb"},
		},
	}
	release := Release{
		TagName: "v1.2.3",
		Assets: []Asset{
			{Name: "plugin-linux-amd64", BrowserDownloadURL: "https://github.com/Silo-Server/silo-plugin-metadata-tmdb/releases/download/v1.2.3/plugin-linux-amd64"},
			{Name: "checksums.txt", BrowserDownloadURL: "https://github.com/Silo-Server/silo-plugin-metadata-tmdb/releases/download/v1.2.3/checksums.txt"},
		},
	}

	if _, err := BuildPackageFromRelease("Silo-Server/silo-plugin-metadata-tmdb", source, release); err == nil {
		t.Fatal("BuildPackageFromRelease() accepted a manifest without presentation metadata")
	}
}

func TestBuildPackageFromRelease_PreservesManifestMetadataAndConfigSchema(t *testing.T) {
	source, err := DecodeSourceManifest([]byte(`{
	  "plugin_id": "silo.requests.arr",
	  "version": "0.1.0",
	  "checksum": "__CHECKSUM__",
	  "silo_api_version": "v1",
	  "presentation": {
	    "display_name": "Sonarr & Radarr Requests",
	    "summary": "Routes requests to Sonarr and Radarr.",
	    "description_markdown": "Routes approved requests.",
	    "setup_markdown": "Add a connection.",
	    "homepage_url": "https://github.com/Silo-Server/silo-plugins-requests-arr",
	    "source_url": "https://github.com/Silo-Server/silo-plugins-requests-arr",
	    "support_url": "https://github.com/Silo-Server/silo-plugins-requests-arr/issues",
	    "changelog_url": "https://github.com/Silo-Server/silo-plugins-requests-arr/releases",
	    "publisher_name": "Silo",
	    "publisher_url": "https://github.com/Silo-Server",
	    "license_spdx": "AGPL-3.0-only"
	  },
	  "supported_platforms": [{"os": "linux", "arch": "amd64"}],
	  "capabilities": [{
	    "type": "request_router.v1",
	    "id": "arr",
	    "display_name": "Sonarr / Radarr",
	    "metadata": {"default_priority": {"series": 5}},
	    "config_schema": [{
	      "key": "connection",
	      "json_schema": "{\"type\":\"object\",\"properties\":{\"service_kind\":{\"type\":\"string\"}}}",
	      "admin_form": {
	        "submit_label": "Save connection",
	        "fields": [{
	          "key": "service_kind",
	          "label": "Service",
	          "control": "ADMIN_FORM_CONTROL_SELECT",
	          "options": [{"value": "sonarr", "label": "Sonarr"}]
	        }]
	      }
	    }]
	  }],
	  "global_config_schema": [{
	    "key": "account",
	    "json_schema": "{\"type\":\"object\",\"properties\":{\"api_key\":{\"type\":\"string\"}}}",
	    "admin_form": {
	      "fields": [{
	        "key": "api_key",
	        "label": "API Key",
	        "control": "ADMIN_FORM_CONTROL_PASSWORD",
	        "secret": true
	      }]
	    }
	  }]
	}`))
	if err != nil {
		t.Fatalf("DecodeSourceManifest() error = %v", err)
	}
	release := Release{
		TagName: "v0.1.1",
		Assets: []Asset{
			{Name: "plugin-linux-amd64", BrowserDownloadURL: "https://github.com/Silo-Server/silo-plugins-requests-arr/releases/download/v0.1.1/plugin-linux-amd64"},
			{Name: "checksums.txt", BrowserDownloadURL: "https://github.com/Silo-Server/silo-plugins-requests-arr/releases/download/v0.1.1/checksums.txt"},
		},
	}

	pkg, err := BuildPackageFromRelease("Silo-Server/silo-plugins-requests-arr", source, release)
	if err != nil {
		t.Fatalf("BuildPackageFromRelease() error = %v", err)
	}

	if pkg.Manifest.GetVersion() != "0.1.1" {
		t.Fatalf("Version = %q, want tag version", pkg.Manifest.GetVersion())
	}
	if got := len(pkg.Manifest.GetSupportedPlatforms()); got != 1 {
		t.Fatalf("SupportedPlatforms length = %d, want 1", got)
	}
	capability := pkg.Manifest.GetCapabilities()[0]
	if capability.GetMetadata().AsMap()["default_priority"] == nil {
		t.Fatalf("metadata default_priority was not preserved: %v", capability.GetMetadata())
	}
	if got := len(capability.GetConfigSchema()); got != 1 {
		t.Fatalf("ConfigSchema length = %d, want 1", got)
	}
	field := capability.GetConfigSchema()[0].GetAdminForm().GetFields()[0]
	if field.GetControl() != pluginv1.AdminFormControl_ADMIN_FORM_CONTROL_SELECT {
		t.Fatalf("control = %v, want select", field.GetControl())
	}
	if got := len(pkg.Manifest.GetGlobalConfigSchema()); got != 1 {
		t.Fatalf("GlobalConfigSchema length = %d, want 1", got)
	}

	data, err := json.Marshal(RepositoryIndex{Plugins: []CatalogPackage{pkg}})
	if err != nil {
		t.Fatalf("Marshal catalog() error = %v", err)
	}
	var decoded struct {
		Plugins []struct {
			Manifest *pluginv1.PluginManifest `json:"manifest"`
		} `json:"plugins"`
	}
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("standard json decode like Silo catalog_service failed: %v\n%s", err, data)
	}
	decodedField := decoded.Plugins[0].Manifest.GetCapabilities()[0].GetConfigSchema()[0].GetAdminForm().GetFields()[0]
	if decodedField.GetControl() != pluginv1.AdminFormControl_ADMIN_FORM_CONTROL_SELECT {
		t.Fatalf("decoded control = %v, want select", decodedField.GetControl())
	}
}

func catalogTestPresentation(sourceURL string) *pluginv1.PluginPresentation {
	return &pluginv1.PluginPresentation{
		DisplayName:         "Test Plugin",
		Summary:             "Test summary.",
		DescriptionMarkdown: "Test description.",
		SetupMarkdown:       "Test setup.",
		HomepageUrl:         sourceURL,
		SourceUrl:           sourceURL,
		SupportUrl:          sourceURL + "/issues",
		ChangelogUrl:        sourceURL + "/releases",
		PublisherName:       "Silo",
		PublisherUrl:        "https://github.com/Silo-Server",
		LicenseSpdx:         "AGPL-3.0-only",
	}
}

func TestUpsertPackage_ReplacesExistingPluginAndSorts(t *testing.T) {
	index := RepositoryIndex{
		Plugins: []CatalogPackage{
			{
				Manifest: &pluginv1.PluginManifest{
					PluginId:       "silo.tvdb",
					Version:        "1.0.0",
					SiloApiVersion: "v1",
				},
			},
			{
				Manifest: &pluginv1.PluginManifest{
					PluginId:       "silo.tmdb",
					Version:        "1.0.0",
					SiloApiVersion: "v1",
				},
			},
		},
	}

	updated := CatalogPackage{
		Manifest: &pluginv1.PluginManifest{
			PluginId:       "silo.tmdb",
			Version:        "1.2.3",
			SiloApiVersion: "v1",
		},
		RepoURL: "https://github.com/Silo-Server/silo-plugin-metadata-tmdb",
	}

	index = UpsertPackage(index, updated)

	if len(index.Plugins) != 2 {
		t.Fatalf("Plugins length = %d, want 2", len(index.Plugins))
	}
	if index.Plugins[0].Manifest.GetPluginId() != "silo.tmdb" {
		t.Fatalf("Plugins[0].PluginID = %q", index.Plugins[0].Manifest.GetPluginId())
	}
	if index.Plugins[0].Manifest.GetVersion() != "1.2.3" {
		t.Fatalf("Plugins[0].Version = %q", index.Plugins[0].Manifest.GetVersion())
	}
	if index.Plugins[1].Manifest.GetPluginId() != "silo.tvdb" {
		t.Fatalf("Plugins[1].PluginID = %q", index.Plugins[1].Manifest.GetPluginId())
	}
}

func TestBuildPackageFromRelease_BundleURL(t *testing.T) {
	source := &SourceManifest{
		PluginId:       "silo.tmdb",
		Version:        "1.2.3",
		SiloApiVersion: "v1",
		Presentation:   catalogTestPresentation("https://github.com/Silo-Server/silo-plugin-metadata-tmdb"),
		Capabilities:   []*pluginv1.CapabilityDescriptor{{Type: "metadata_provider.v1", Id: "tmdb"}},
	}
	assets := []Asset{
		{Name: "plugin-linux-amd64", BrowserDownloadURL: "https://github.com/Silo-Server/silo-plugin-metadata-tmdb/releases/download/v1.2.3/plugin-linux-amd64"},
		{Name: "checksums.txt", BrowserDownloadURL: "https://github.com/Silo-Server/silo-plugin-metadata-tmdb/releases/download/v1.2.3/checksums.txt"},
	}

	unsigned, err := BuildPackageFromRelease("Silo-Server/silo-plugin-metadata-tmdb", source, Release{TagName: "v1.2.3", Assets: assets})
	if err != nil {
		t.Fatalf("BuildPackageFromRelease(unsigned) error = %v", err)
	}
	if unsigned.BundleURL != "" {
		t.Fatalf("unsigned BundleURL = %q, want empty", unsigned.BundleURL)
	}
	data, err := json.Marshal(unsigned)
	if err != nil {
		t.Fatal(err)
	}
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(data, &raw); err != nil {
		t.Fatal(err)
	}
	if _, ok := raw["bundle_url"]; ok {
		t.Fatalf("unsigned entry emits bundle_url: %s", data)
	}

	bundle := "https://github.com/Silo-Server/silo-plugin-metadata-tmdb/releases/download/v1.2.3/plugin.sigstore.json"
	signedAssets := append(slices.Clone(assets), Asset{Name: BundleAssetName, BrowserDownloadURL: bundle})
	signed, err := BuildPackageFromRelease("Silo-Server/silo-plugin-metadata-tmdb", source, Release{TagName: "v1.2.3", Assets: signedAssets})
	if err != nil {
		t.Fatalf("BuildPackageFromRelease(signed) error = %v", err)
	}
	if signed.BundleURL != bundle {
		t.Fatalf("BundleURL = %q, want %q", signed.BundleURL, bundle)
	}
	if len(signed.Binaries) != 1 {
		t.Fatalf("Binaries = %v, want only the plugin binary", signed.Binaries)
	}
	data, err = json.Marshal(signed)
	if err != nil {
		t.Fatal(err)
	}
	var decoded struct {
		BundleURL    string `json:"bundle_url"`
		ChecksumsURL string `json:"checksums_url"`
	}
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatal(err)
	}
	if decoded.BundleURL != bundle || decoded.ChecksumsURL == "" {
		t.Fatalf("encoded entry = %s, want bundle_url beside checksums_url", data)
	}
}

func TestCheckOfficialRepo(t *testing.T) {
	tests := []struct {
		name     string
		pluginID string
		repo     string
		wantErr  bool
	}{
		{"bound repository", "silo.tmdb", "Silo-Server/silo-plugin-metadata-tmdb", false},
		{"another official repository", "silo.tmdb", "Silo-Server/silo-plugin-metadata-tvdb", true},
		{"same name in another owner", "silo.tmdb", "Silo-Community/silo-plugin-metadata-tmdb", true},
		{"different spelling", "silo.tmdb", "silo-server/silo-plugin-metadata-tmdb", true},
		{"plugin not in the table", "silo.requests.arr", "Silo-Community/silo-plugins-requests-arr", true},
		{"empty plugin ID", "", "Silo-Server/silo-plugin-metadata-tmdb", true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := CheckOfficialRepo(tt.pluginID, tt.repo)
			if (err != nil) != tt.wantErr {
				t.Fatalf("CheckOfficialRepo(%q, %q) error = %v, wantErr %v", tt.pluginID, tt.repo, err, tt.wantErr)
			}
		})
	}
}

func TestCheckSignatureRatchet(t *testing.T) {
	const bundle = "https://github.com/Silo-Server/silo-plugin-metadata-tmdb/releases/download/v1.2.3/plugin.sigstore.json"
	signed := &CatalogPackage{Manifest: &pluginv1.PluginManifest{PluginId: "silo.tmdb"}, BundleURL: bundle}
	unsigned := &CatalogPackage{Manifest: &pluginv1.PluginManifest{PluginId: "silo.tvdb"}}

	tests := []struct {
		name      string
		current   *CatalogPackage
		bundleURL string
		wantErr   bool
	}{
		{"unsigned after signed", signed, "", true},
		{"signed after signed", signed, bundle, false},
		{"unsigned after unsigned", unsigned, "", false},
		{"signed after unsigned", unsigned, bundle, false},
		{"unsigned first release", nil, "", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := CheckSignatureRatchet(tt.current, tt.bundleURL)
			if (err != nil) != tt.wantErr {
				t.Fatalf("CheckSignatureRatchet() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestVerifyArgs(t *testing.T) {
	got := VerifyArgs("/tmp/v/binary-0", "/tmp/v/plugin.sigstore.json", "Silo-Server/silo-plugin-metadata-tmdb", "1.2.3")
	want := []string{
		"attestation", "verify", "/tmp/v/binary-0",
		"--bundle=/tmp/v/plugin.sigstore.json",
		"--repo=Silo-Server/silo-plugin-metadata-tmdb",
		"--cert-identity=https://github.com/Silo-Server/.github/.github/workflows/plugin-release.yml@refs/heads/main",
		"--cert-oidc-issuer=https://token.actions.githubusercontent.com",
		"--source-ref=refs/tags/v1.2.3",
		"--predicate-type=https://slsa.dev/provenance/v1",
		"--deny-self-hosted-runners",
		"--format=json",
	}
	if !slices.Equal(got, want) {
		t.Fatalf("VerifyArgs() =\n%q\nwant\n%q", got, want)
	}
}

func TestCheckVerifyOutput(t *testing.T) {
	const commitA = "29fd28771a006d9d68a311865a2b858c5afe301a"
	const commitB = "63837fb778d6a39f02be0ce76ff89f2bc8d74bdc"
	result := func(issuer, tsType, digest string) string {
		return `{"verificationResult":{"signature":{"certificate":{"certificateIssuer":"` + issuer +
			`","sourceRepositoryDigest":"` + digest + `"}},"verifiedTimestamps":[{"type":"` + tsType + `","uri":"https://rekor.sigstore.dev"}]}}`
	}
	const publicGood = "CN=sigstore-intermediate,O=sigstore.dev"
	tests := []struct {
		name       string
		out        string
		wantCommit string
		wantErr    bool
	}{
		{"public-good with transparency log", "[" + result(publicGood, "Tlog", commitA) + "]", commitA, false},
		{"two attestations, one commit", "[" + result(publicGood, "Tlog", commitA) + "," + result(publicGood, "Tlog", commitA) + "]", commitA, false},
		{"two attestations, two commits", "[" + result(publicGood, "Tlog", commitA) + "," + result(publicGood, "Tlog", commitB) + "]", "", true},
		{"no source commit", "[" + result(publicGood, "Tlog", "") + "]", "", true},
		{"short source commit", "[" + result(publicGood, "Tlog", "29fd287") + "]", "", true},
		{"GitHub private instance", "[" + result("CN=Fulcio Intermediate l2,O=GitHub\\, Inc.", "TimestampAuthority", commitA) + "]", "", true},
		{"no transparency-log entry", "[" + result(publicGood, "TimestampAuthority", commitA) + "]", "", true},
		{"no attestations", `[]`, "", true},
		{"not JSON", `verified`, "", true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			commit, err := CheckVerifyOutput([]byte(tt.out))
			if (err != nil) != tt.wantErr {
				t.Fatalf("CheckVerifyOutput() error = %v, wantErr %v", err, tt.wantErr)
			}
			if commit != tt.wantCommit {
				t.Fatalf("CheckVerifyOutput() commit = %q, want %q", commit, tt.wantCommit)
			}
		})
	}
}

func TestOfficialTableIsOneToOne(t *testing.T) {
	seen := map[string]string{}
	for pluginID, repo := range officialPluginRepos {
		if other, ok := seen[repo]; ok {
			t.Fatalf("%s is bound to both %s and %s", repo, other, pluginID)
		}
		seen[repo] = pluginID
		got, err := OfficialPluginID(repo)
		if err != nil || got != pluginID {
			t.Fatalf("OfficialPluginID(%q) = %q, %v; want %q", repo, got, err, pluginID)
		}
	}
	if _, err := OfficialPluginID("Silo-Community/silo-plugins-requests-arr"); err == nil {
		t.Fatal("OfficialPluginID accepted a repository outside the table")
	}
}

func TestValidateInputs(t *testing.T) {
	const repo = "Silo-Server/silo-plugin-metadata-tmdb"
	tests := []struct {
		name    string
		repo    string
		tag     string
		wantErr bool
	}{
		{"release tag", repo, "v1.2.3", false},
		{"suffixed tag", repo, "v1.2.3-hotfix.1", false},
		{"traversal in tag", repo, "../../../../attacker/evil/releases/tags/v9.0.0", true},
		{"traversal after version", repo, "v1.2.3/../../../attacker/evil", true},
		{"slash in suffix", repo, "v1.2.3-a/b", true},
		{"query in tag", repo, "v1.2.3?x=1", true},
		{"encoded slash", repo, "v1.2.3-%2F", true},
		{"no v prefix", repo, "1.2.3", true},
		{"leading zero", repo, "v01.2.3", true},
		{"missing patch", repo, "v1.2", true},
		{"empty tag", repo, "", true},
		{"three path segments", "Silo-Server/silo-plugin-metadata-tmdb/x", "v1.2.3", true},
		{"traversal in repo", "Silo-Server/../attacker", "v1.2.3", true},
		{"no owner", "/silo-plugin-metadata-tmdb", "v1.2.3", true},
		{"empty repo", "", "v1.2.3", true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateInputs(tt.repo, tt.tag)
			if (err != nil) != tt.wantErr {
				t.Fatalf("ValidateInputs(%q, %q) error = %v, wantErr %v", tt.repo, tt.tag, err, tt.wantErr)
			}
		})
	}
}

func TestValidateCommit(t *testing.T) {
	for _, sha := range []string{"", "29fd287", "29FD28771A006D9D68A311865A2B858C5AFE301A", "../../29fd28771a006d9d68a311865a2b858c5af"} {
		if err := ValidateCommit(sha); err == nil {
			t.Fatalf("ValidateCommit(%q) accepted an invalid SHA", sha)
		}
	}
	if err := ValidateCommit("29fd28771a006d9d68a311865a2b858c5afe301a"); err != nil {
		t.Fatalf("ValidateCommit() error = %v", err)
	}
}

func TestCheckRelease(t *testing.T) {
	tests := []struct {
		name    string
		release Release
		wantErr bool
	}{
		{"published release", Release{TagName: "v1.2.3"}, false},
		{"different tag", Release{TagName: "v9.0.0"}, true},
		{"draft", Release{TagName: "v1.2.3", Draft: true}, true},
		{"prerelease", Release{TagName: "v1.2.3", Prerelease: true}, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := CheckRelease("v1.2.3", tt.release)
			if (err != nil) != tt.wantErr {
				t.Fatalf("CheckRelease() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestParseReleaseAssets(t *testing.T) {
	const repo = "Silo-Server/silo-plugin-metadata-tmdb"
	const base = "https://github.com/" + repo + "/releases/download/v1.2.3/"
	valid := []Asset{
		{Name: "plugin-linux-amd64", BrowserDownloadURL: base + "plugin-linux-amd64"},
		{Name: "checksums.txt", BrowserDownloadURL: base + "checksums.txt"},
		{Name: BundleAssetName, BrowserDownloadURL: base + BundleAssetName},
		// Not catalogued, so its URL is not checked.
		{Name: "notes.txt", BrowserDownloadURL: "https://example.invalid/notes.txt"},
	}
	assets, err := ParseReleaseAssets(repo, Release{TagName: "v1.2.3", Assets: valid})
	if err != nil {
		t.Fatalf("ParseReleaseAssets() error = %v", err)
	}
	if assets.ChecksumsURL != base+"checksums.txt" || assets.BundleURL != base+BundleAssetName || assets.Binaries["linux/amd64"] != base+"plugin-linux-amd64" {
		t.Fatalf("ParseReleaseAssets() = %+v", assets)
	}

	tests := []struct {
		name  string
		asset Asset
	}{
		{"binary in another repository", Asset{Name: "plugin-linux-arm64", BrowserDownloadURL: "https://github.com/attacker/evil/releases/download/v1.2.3/plugin-linux-arm64"}},
		{"binary on another host", Asset{Name: "plugin-linux-arm64", BrowserDownloadURL: "https://example.invalid/plugin-linux-arm64"}},
		{"binary in another tag", Asset{Name: "plugin-linux-arm64", BrowserDownloadURL: "https://github.com/" + repo + "/releases/download/v1.0.0/plugin-linux-arm64"}},
		{"binary path traversal", Asset{Name: "plugin-linux-arm64", BrowserDownloadURL: base + "../../../../../attacker/evil/releases/download/v1.2.3/plugin-linux-arm64"}},
		{"binary under another name", Asset{Name: "plugin-linux-arm64", BrowserDownloadURL: base + "plugin-linux-amd64"}},
		{"plain-http checksums", Asset{Name: "checksums.txt", BrowserDownloadURL: "http://github.com/" + repo + "/releases/download/v1.2.3/checksums.txt"}},
		{"bundle in another repository", Asset{Name: BundleAssetName, BrowserDownloadURL: "https://github.com/attacker/evil/releases/download/v1.2.3/" + BundleAssetName}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			release := Release{TagName: "v1.2.3", Assets: append(slices.Clone(valid[:1]), tt.asset)}
			if _, err := ParseReleaseAssets(repo, release); err == nil {
				t.Fatalf("ParseReleaseAssets() accepted %+v", tt.asset)
			}
		})
	}
}

func TestCheckVersion(t *testing.T) {
	current := &CatalogPackage{Manifest: &pluginv1.PluginManifest{PluginId: "silo.tmdb", Version: "1.2.10"}}
	tests := []struct {
		name          string
		current       *CatalogPackage
		version       string
		allowRollback bool
		wantListed    bool
		wantErr       bool
	}{
		{"first release", nil, "0.1.0", false, false, false},
		{"newer patch, compared numerically", current, "1.2.11", false, false, false},
		{"newer minor", current, "1.3.0", false, false, false},
		{"same version is a no-op", current, "1.2.10", false, true, false},
		{"older patch, compared numerically", current, "1.2.9", false, false, true},
		{"older major", current, "0.9.0", false, false, true},
		{"suffixed version sorts before its release", current, "1.2.10-hotfix.1", false, false, true},
		{"rollback with override", current, "1.2.9", true, false, false},
		{"same version with override rewrites", current, "1.2.10", true, false, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			listed, err := CheckVersion(tt.current, tt.version, tt.allowRollback)
			if (err != nil) != tt.wantErr || listed != tt.wantListed {
				t.Fatalf("CheckVersion() = %v, %v; want listed %v, wantErr %v", listed, err, tt.wantListed, tt.wantErr)
			}
		})
	}
}
