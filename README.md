# Silo Plugin Catalog

Catalog metadata and helpers for first-party Silo plugins.

## Catalog updates

Plugin repositories should dispatch `plugin_release_published` after publishing
a release. Set `SILO_PLUGINS_DISPATCH_TOKEN` in the plugin repository so it can
call `repository_dispatch` on `Silo-Server/silo-plugins`.

`silo-plugins` uses `CATALOG_PUSH_TOKEN` to push catalog updates. If plugin
repositories are private, also set `CATALOG_SOURCE_TOKEN` in `silo-plugins` so
the updater can read release metadata and the tagged `manifest.json`.

## License

`silo-plugins` is licensed under `Apache-2.0`. See [LICENSE](LICENSE).
