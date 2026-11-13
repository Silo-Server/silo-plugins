# Silo Plugin Catalog

Catalog metadata and helpers for first-party Silo plugins.

Silo servers consume the generated `manifest.json` to discover approved plugin
releases, supported platforms, checksums, capabilities, and presentation
metadata. Source plugin repositories remain the authority for implementation
and release artifacts.

## Catalog updates

Plugin repositories should dispatch `plugin_release_published` after publishing
a release. Set `SILO_PLUGINS_DISPATCH_TOKEN` in the plugin repository so it can
call `repository_dispatch` on `Silo-Server/silo-plugins`.

`update-catalog` refuses a release unless all of these hold:

- The repository is `owner/name` and the tag is `v<major>.<minor>.<patch>`,
  optionally with a `-suffix`. Both are checked before either goes into a URL.
- The repository is bound to a plugin ID in
  [`catalog/official.go`](catalog/official.go), and the release's
  `manifest.json` names that ID. Adding a first-party plugin means adding it
  there, and to the matching table in Silo server.
- The version is newer than the plugin's listed version. Re-running for the
  listed version does nothing and succeeds. A deliberate rollback needs the
  `allow_rollback` input of a manual `workflow_dispatch` run; a
  `repository_dispatch` cannot set it.
- The release is published for exactly that tag and is neither a draft nor a
  prerelease. There is no override for drafts or prereleases.
- Every catalogued asset URL is
  `https://github.com/<repo>/releases/download/<tag>/<asset name>`.

`manifest.json` is read at a commit SHA, never at the tag name: the commit the
signature names for a signed release, or the commit the tag points at
(following an annotated tag) for an unsigned one.

A signed release also carries `plugin.sigstore.json`: a Sigstore bundle with
one SLSA provenance attestation covering every `plugin-<os>-<arch>` binary,
produced by the shared `plugin-release.yml` workflow in `Silo-Server/.github`.
The catalog entry then gains a package-level `bundle_url` beside
`checksums_url`. Before writing the entry, `update-catalog` checks each binary
with `gh attestation verify`, requiring the certificate identity
`https://github.com/Silo-Server/.github/.github/workflows/plugin-release.yml@refs/heads/main`,
the issuer `https://token.actions.githubusercontent.com`, the plugin's bound
repository, the source ref `refs/tags/v<version>`, a GitHub-hosted runner, a
Sigstore public-good transparency-log entry, and the same source commit for
every binary. Any failure refuses the release. Unsigned releases are still
accepted for a plugin until its catalog entry has a `bundle_url`; after that,
an unsigned release of that plugin is refused.

The workflow has no concurrency group. Runs race on the push, and each retry
rebuilds the entry on the current `main`, so a duplicate run is a no-op and an
older release that loses the race to a newer one is refused.

`silo-plugins` uses `CATALOG_PUSH_TOKEN` to push catalog updates. Reading a
plugin's release metadata and its tagged `manifest.json` needs no extra
credential: every catalogued plugin repository is public, so the workflow's own
`github.token` is enough. A plugin has to be public before it dispatches.

To exercise ingestion locally against an existing tagged release, pass the
repository in `owner/name` form and the exact release tag. Verifying a signed
release needs the `gh` CLI on `PATH`:

```sh
go run ./cmd/update-catalog \
  -repo Silo-Server/silo-plugin-metadata-tmdb \
  -tag v1.2.30
```

Use a tag newer than the listed version. The listed version does nothing and
an older one is refused; to rebuild an entry from an older or equal release
while testing, add `-allow-rollback` and pass `-manifest` with a copy of the
catalog so the real one is not rewritten.

For a private source repository, provide `GITHUB_TOKEN` through your normal
secret-injection workflow. The command rewrites `manifest.json`; review or
discard that diff after the check.

## Development

```sh
go test ./...
go vet ./...
go build ./...
```

## Contributing

Read [CONTRIBUTING.md](CONTRIBUTING.md) before opening a pull request. Plugin
implementation changes belong in the source plugin repository; catalog schema
and automation changes belong here.

## License

`silo-plugins` is licensed under `Apache-2.0`. See [LICENSE](LICENSE).
