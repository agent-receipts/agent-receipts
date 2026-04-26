# Changelog

All notable changes to `@agnt-rcpt/sdk-ts` are documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

This file starts at 0.5.0; earlier releases are recorded only in git history.
A repo-wide effort to auto-generate changelogs from Conventional Commits is
tracked in [#253](https://github.com/agent-receipts/ar/issues/253).

## [0.5.0] - 2026-04-27

### Added

- `parameters_preview?: Record<string, string>` field on the `Action` interface.
  An operator-controlled, additive map of field name → stringified value that
  sits alongside the existing `parameters_hash`. Only fields explicitly listed
  in the taxonomy `preview_fields` config should ever be included; the hash
  still covers the full parameter set.
