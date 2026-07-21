# Changelog

All notable changes to this project are documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/).

## [Unreleased]

Starting with the next release, `steampipe-config-generator` follows [Semantic
Versioning](#versioning). Everything below shipped since `v0.1.2` and will ship as `v1.0.0`.

### Breaking changes

- CLI flags now use a double dash (`--role`) instead of a single dash (`-role`). Update any
  scripts that invoke this tool.
- The Go library API (`pkg/aws`, `pkg/logger`) has been removed and replaced by a new public
  package, `generator`, with a different shape: a `Generator` interface, a `New(ctx, opts)`
  constructor, and pure `RenderConnections`/`RenderCredentials` functions. Anything importing
  the old `pkg/` packages directly will need to migrate.

### Added

- `--tagSplit` flag: split a multi-value AWS tag (e.g. `team=frontend:backend`) into individual
  values, so each becomes its own Steampipe aggregator group. Opt-in per tag key; existing
  single-value tags are unaffected. See the README for examples.
- `--version` flag and `version` subcommand, with version/commit/date injected at build time.
- `--help` (via Cobra), documenting every flag.
- Logs now include the path of each config file written (AWS credentials file and Steampipe
  connections file).

### Changed

- Migrated the CLI from the standard `flag` package to [Cobra](https://github.com/spf13/cobra).
- Restructured the codebase into an idiomatic library/CLI split: `generator/` (public library),
  `internal/aws/` and `internal/logger/` (implementation details), `cmd/` (Cobra wiring), with a
  minimal `main.go`.
- Replaced `logrus` with the standard library's `log/slog`.
- Replaced the hand-rolled concurrency semaphores for fetching account tags/OUs with
  `errgroup.SetLimit`.
- `AWS Organizations` account filtering now uses the `State` field instead of the deprecated
  `Status` field ([retired by AWS on 2026-09-09](https://aws.amazon.com/blogs/mt/updates-to-account-status-information-in-aws-organizations/)).
- Upgraded to Go 1.26.5 and refreshed all dependencies.

### Fixed

- A failed fetch of one account's tags or organizational unit used to be logged and silently
  dropped, leaving that account with incomplete data. It now fails the whole run with a clear
  error instead.
- The tool used to exit with status code `0` even when it failed. It now exits non-zero on any
  error.
- Templates were rendered with `html/template`, which HTML-escaped account names containing
  characters like `&`, corrupting the generated credentials/connections files. Switched to
  `text/template`.

## [0.1.2] and earlier

See the [GitHub releases](https://github.com/unicrons/steampipe-config-generator/releases) for
this and earlier versions.

[Unreleased]: https://github.com/unicrons/steampipe-config-generator/compare/v0.1.2...HEAD
