package catalog

import (
	"encoding/json"
	"fmt"
	"strings"
)

const (
	// BundleAssetName is the release asset holding the Sigstore bundle. It
	// carries one SLSA provenance attestation whose subjects are every
	// plugin-<os>-<arch> binary in the release.
	BundleAssetName = "plugin.sigstore.json"

	// SignerIdentity is the exact certificate SAN of the reusable release
	// workflow. Callers reference it at @main, so a tag or SHA reference, or a
	// fork commit, does not match.
	SignerIdentity = "https://github.com/Silo-Server/.github/.github/workflows/plugin-release.yml@refs/heads/main"

	// OIDCIssuer is GitHub Actions' token issuer.
	OIDCIssuer = "https://token.actions.githubusercontent.com"

	slsaProvenancePredicate = "https://slsa.dev/provenance/v1"

	// Fulcio's public-good intermediate. GitHub's own Sigstore instance, used
	// for private repositories, has a different issuer and no transparency log.
	publicGoodCertificateIssuerOrg = "O=sigstore.dev"
)

// VerifyArgs returns the `gh` arguments that verify one release binary against
// the release's bundle. They enforce the policy the Silo server applies to
// official plugins: the reusable workflow at main, GitHub's issuer, the
// plugin's bound repository, the tag the catalog version names, and a
// GitHub-hosted runner. Flags use the --flag=value form so no value can be
// read as another flag.
func VerifyArgs(binaryPath, bundlePath, repo, version string) []string {
	return []string{
		"attestation", "verify", binaryPath,
		"--bundle=" + bundlePath,
		"--repo=" + repo,
		"--cert-identity=" + SignerIdentity,
		"--cert-oidc-issuer=" + OIDCIssuer,
		"--source-ref=refs/tags/v" + version,
		"--predicate-type=" + slsaProvenancePredicate,
		"--deny-self-hosted-runners",
		"--format=json",
	}
}

// CheckVerifyOutput checks the JSON that `gh attestation verify --format=json`
// prints on success and returns the source commit the certificate names. gh
// also accepts GitHub's private Sigstore instance; the Silo server trusts only
// the public-good root with a transparency-log entry, so the catalog refuses
// anything else here rather than publish an entry every server would reject.
// Every verified attestation must name the same commit.
func CheckVerifyOutput(out []byte) (commit string, err error) {
	var results []struct {
		VerificationResult struct {
			VerifiedTimestamps []struct {
				Type string `json:"type"`
			} `json:"verifiedTimestamps"`
			Signature struct {
				Certificate struct {
					CertificateIssuer      string `json:"certificateIssuer"`
					SourceRepositoryDigest string `json:"sourceRepositoryDigest"`
				} `json:"certificate"`
			} `json:"signature"`
		} `json:"verificationResult"`
	}
	if err := json.Unmarshal(out, &results); err != nil {
		return "", fmt.Errorf("decode gh attestation verify output: %w", err)
	}
	if len(results) == 0 {
		return "", fmt.Errorf("gh attestation verify reported no verified attestation")
	}
	for _, result := range results {
		cert := result.VerificationResult.Signature.Certificate
		if !strings.Contains(cert.CertificateIssuer, publicGoodCertificateIssuerOrg) {
			return "", fmt.Errorf("attestation certificate is from %q, not the Sigstore public-good instance", cert.CertificateIssuer)
		}
		logged := false
		for _, ts := range result.VerificationResult.VerifiedTimestamps {
			if ts.Type == "Tlog" {
				logged = true
				break
			}
		}
		if !logged {
			return "", fmt.Errorf("attestation has no verified transparency-log entry")
		}
		if err := ValidateCommit(cert.SourceRepositoryDigest); err != nil {
			return "", fmt.Errorf("attestation source commit: %w", err)
		}
		if commit != "" && cert.SourceRepositoryDigest != commit {
			return "", fmt.Errorf("attestations name commits %s and %s", commit, cert.SourceRepositoryDigest)
		}
		commit = cert.SourceRepositoryDigest
	}
	return commit, nil
}

// CheckSignatureRatchet refuses an unsigned release of a plugin whose current
// catalog entry is signed. Unsigned releases stay accepted until a plugin's
// first signed release lands, which lets plugins move to signing one at a time.
func CheckSignatureRatchet(current *CatalogPackage, bundleURL string) error {
	if bundleURL == "" && current != nil && current.BundleURL != "" {
		return fmt.Errorf("plugin %q already has a signed catalog entry; this release has no %s", packagePluginID(*current), BundleAssetName)
	}
	return nil
}
