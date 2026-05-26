# Changelog — probe-proxmox

All notable changes to **fluid-pub/probe-proxmox** are documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

Tag naming: `0.y.z` (no `v` prefix). Align `cmd/version.go` with the tag before release.

## [Unreleased]

## [0.1.0] - 2026-05-26

### Added

- Initial public release aligned with **probe-core** (HTTP `/probes`, `MergedConfigProvider`, `runtime_config` reload).
- Entities: `qemu`, `access_users`, optional `cluster_resources` (Proxmox VE API).
- CI/CD via `fluid-pub/actions`, distroless image with embedded `config/schema.yml`.
- Local `gofmt` pre-commit hook matching CI.
