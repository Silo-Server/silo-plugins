# Contributing to the Silo Plugin Catalog

The [Silo contribution guide](https://github.com/Silo-Server/.github/blob/main/CONTRIBUTING.md)
covers project-wide coordination, focused changes, evidence, AI disclosure, and
pull request expectations. Those requirements apply here; this guide adds the
catalog-specific workflow.

## Before you start

Open an [issue](https://github.com/Silo-Server/silo-plugins/issues) before
changing catalog schema, provenance rules, release ingestion, or automation.
Plugin implementation changes belong in the individual plugin repository;
plugin contract changes belong in
[`silo-plugin-sdk`](https://github.com/Silo-Server/silo-plugin-sdk).

Release entries are normally generated from a plugin's published GitHub release
by the catalog workflows. Do not hand-edit checksums or release metadata to work
around a missing or incorrect upstream artifact.

## Development setup

Use the Go version declared in `go.mod`. The update command reads release assets
from GitHub, so use a real tagged release when testing ingestion and never place
tokens in command arguments, committed files, or logs.

## Validate your change

```sh
go test ./...
go vet ./...
go build ./...
gofmt -l .
```

`gofmt -l .` should print nothing. If it reports unrelated pre-existing drift,
none of the Go files touched by your change may appear in the output; do not add
to the output, and report what remains. For catalog output changes, inspect the
complete `manifest.json` diff and verify repository URLs, versions, platforms,
the checksum URL, capabilities, and presentation metadata against the source
release. The catalog stores a URL for `checksums.txt`, not the checksum values
themselves, so also confirm that the release asset exists and covers every
published binary.

## Open the pull request

Use a Conventional Commit title, explain the catalog or automation impact, and
paste the actual validation results. Read the
[AI-assisted contribution policy](https://github.com/Silo-Server/silo-server/blob/main/docs/ai-contributions.md)
and include its disclosure block.
