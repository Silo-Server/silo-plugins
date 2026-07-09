package catalog

import (
	"encoding/json"
	"testing"

	pluginv1 "github.com/Silo-Server/silo-plugin-sdk/pkg/pluginproto/silo/plugin/v1"
)

func TestBuildPackageFromRelease_MinimalManifestAndAssets(t *testing.T) {
	source := &SourceManifest{
		PluginId:       "silo.tmdb",
		Version:        "1.2.3",
		SiloApiVersion: "v1",
		Presentation:   catalogTestPresentation("https://github.com/Silo-Server/silo-plugin-tmdb"),
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
			{Name: "plugin-linux-amd64", BrowserDownloadURL: "https://example.invalid/tmdb/plugin-linux-amd64"},
			{Name: "plugin-linux-arm64", BrowserDownloadURL: "https://example.invalid/tmdb/plugin-linux-arm64"},
			{Name: "checksums.txt", BrowserDownloadURL: "https://example.invalid/tmdb/checksums.txt"},
			{Name: "notes.txt", BrowserDownloadURL: "https://example.invalid/tmdb/notes.txt"},
		},
	}

	pkg, err := BuildPackageFromRelease("Silo-Server/silo-plugin-tmdb", source, release)
	if err != nil {
		t.Fatalf("BuildPackageFromRelease() error = %v", err)
	}

	if pkg.RepoURL != "https://github.com/Silo-Server/silo-plugin-tmdb" {
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
		Presentation:   catalogTestPresentation("https://github.com/Silo-Server/silo-plugin-tmdb"),
		Capabilities: []*pluginv1.CapabilityDescriptor{
			{Type: "metadata_provider.v1", Id: "tmdb"},
		},
	}
	release := Release{
		TagName: "v1.2.3",
		Assets: []Asset{
			{Name: "plugin-linux-amd64", BrowserDownloadURL: "https://example.invalid/tmdb/plugin-linux-amd64"},
			{Name: "checksums.txt", BrowserDownloadURL: "https://example.invalid/tmdb/checksums.txt"},
		},
	}

	pkg, err := BuildPackageFromRelease("Silo-Server/silo-plugin-tmdb", source, release)
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
			{Name: "plugin-linux-amd64", BrowserDownloadURL: "https://example.invalid/tmdb/plugin-linux-amd64"},
			{Name: "checksums.txt", BrowserDownloadURL: "https://example.invalid/tmdb/checksums.txt"},
		},
	}

	if _, err := BuildPackageFromRelease("Silo-Server/silo-plugin-tmdb", source, release); err == nil {
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
			{Name: "plugin-linux-amd64", BrowserDownloadURL: "https://example.invalid/arr/plugin-linux-amd64"},
			{Name: "checksums.txt", BrowserDownloadURL: "https://example.invalid/arr/checksums.txt"},
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
		RepoURL: "https://github.com/Silo-Server/silo-plugin-tmdb",
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
