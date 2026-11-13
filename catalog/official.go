package catalog

import "fmt"

// officialPluginRepos binds each official plugin ID to the one GitHub
// repository, in owner/name form, allowed to publish it. Every plugin
// repository holds a token that can dispatch a catalog update, so without this
// binding one compromised repository could overwrite another plugin's entry.
//
// Silo server keeps its own copy of this table; add a first-party plugin to
// both.
var officialPluginRepos = map[string]string{
	"silo.audiobook-metadata":    "Silo-Server/silo-plugin-metadata-audiobook",
	"silo.auth.ldap":             "Silo-Server/silo-plugin-auth-ldap",
	"silo.auth.oidc":             "Silo-Server/silo-plugin-auth-oidc",
	"silo.autoscan.arr":          "Silo-Server/silo-plugin-autoscan-arr",
	"silo.ebook-metadata":        "Silo-Server/silo-plugin-metadata-ebook",
	"silo.manga-metadata":        "Silo-Server/silo-plugin-metadata-manga",
	"silo.mdblist":               "Silo-Server/silo-plugin-metadata-mdblist",
	"silo.sportarr":              "Silo-Server/silo-plugin-metadata-sportarr",
	"silo.theintrodb":            "Silo-Server/silo-plugin-markers-theintrodb",
	"silo.tmdb":                  "Silo-Server/silo-plugin-metadata-tmdb",
	"silo.tvdb":                  "Silo-Server/silo-plugin-metadata-tvdb",
	"silo.watchprovider.floppy":  "Silo-Server/silo-plugin-watchprovider-floppy",
	"silo.watchprovider.mdblist": "Silo-Server/silo-plugin-watchprovider-mdblist",
	"silo.watchprovider.simkl":   "Silo-Server/silo-plugin-watchprovider-simkl",
	// The repository is archived and the plugin is not in the catalog. The ID
	// stays bound so no other repository can publish under it.
	"silo.watchprovider.trakt": "Silo-Server/silo-plugin-watchprovider-trakt",
}

// CheckOfficialRepo refuses a plugin ID that is not official, or one published
// from a repository other than the one bound to it.
func CheckOfficialRepo(pluginID, repo string) error {
	allowed, ok := officialPluginRepos[pluginID]
	if !ok {
		return fmt.Errorf("plugin %q is not in the official plugin table", pluginID)
	}
	if repo != allowed {
		return fmt.Errorf("plugin %q must be published from %s, not %s", pluginID, allowed, repo)
	}
	return nil
}

// OfficialPluginID returns the plugin ID bound to repo, or an error when repo
// publishes no official plugin. The comparison is exact.
func OfficialPluginID(repo string) (string, error) {
	for pluginID, allowed := range officialPluginRepos {
		if repo == allowed {
			return pluginID, nil
		}
	}
	return "", fmt.Errorf("repository %s does not publish an official plugin", repo)
}
