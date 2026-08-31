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

`silo-plugins` uses `CATALOG_PUSH_TOKEN` to push catalog updates. If plugin
repositories are private, also set `CATALOG_SOURCE_TOKEN` in `silo-plugins` so
the updater can read release metadata and the tagged `manifest.json`.

To exercise ingestion locally against an existing tagged release, pass the
repository in `owner/name` form and the exact release tag:

```sh
go run ./cmd/update-catalog \
  -repo Silo-Server/silo-plugin-metadata-tmdb \
  -tag v1.2.23
```

For a private source repository, provide `GITHUB_TOKEN` through your normal
secret-injection workflow. The command rewrites `manifest.json`; review or
discard that diff after the check.

## Development

```sh
go test ./...
go vet ./...
go build ./cmd/update-catalog
```

## Contributing

Read [CONTRIBUTING.md](CONTRIBUTING.md) before opening a pull request. Plugin
implementation changes belong in the source plugin repository; catalog schema
and automation changes belong here.

## License

`silo-plugins` is licensed under `Apache-2.0`. See [LICENSE](LICENSE).
