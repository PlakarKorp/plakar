# Changelog

## v1.1.4 — 2026-06-30

### Added

- **Dirpack prefetcher for backup walks.** Warms directory metadata ahead of time instead of loading on demand, speeding up backups on remote/high-latency backends.

### Fixed

- `prune -group-by` was being dropped when merged with a retention policy, causing per-day/per-minute caps to apply globally instead of per group. Now carries through correctly.

## v1.1.3 — 2026-06-25

This is the largest release we have ever published. v1.1.3 is the final, stable outcome of the v1.1.0-beta cycle that started in January, hardened through three release candidates and several thousand additional snapshots of real-world testing.

> We jumped straight to v1.1.3. We cut a few quick follow-ups on the way out the door; v1.1.3 is the one you actually want. Several of the extra version bumps also reflect us finding our footing with a new integrations monorepo workflow.

Since v1.0.6 (roughly six months of work):

| Repository   | Commits | Pull requests |
| ------------ | ------- | ------------- |
| plakar       | 421     | 250           |
| kloset       | 599     | 274           |
| integrations | 1,090   | 206           |

### New features

- **New terminal UI.** A proper rendering interface with an `stdio` renderer (old behaviour, unchanged) and a new `tui` renderer that provides a clean, live view during backup and restore.
- **Multi-directory backups.** `plakar backup /etc /home` now works on a single source. Multi-source snapshots (spanning multiple independent data origins) are in progress for the next release.
- **Rewritten FUSE mount.** Far more reliable, including over high-latency connections and on macOS. `plakar mount` can now target specific snapshots or individual directories, and can expose a mount over HTTP. Added `-allow-others` flag to pass `fuse.AllowOther()` at mount time.
- **New package manager.** Simpler, cleaner, and able to update integrations — the missing capability from v1.0.0.
- **Simpler integration interfaces.** The importer, exporter, and storage interfaces have been redesigned. An importer can now feed an exporter directly, enabling a future transfer path between origins and destinations without going through a Kloset.

### Reliability

- **Agent removed; `cached` introduced.** The `agent` process (present since v1.0.0, auto-managed since v1.0.4) is gone. It is replaced by `cached`, a lightweight background process responsible only for shared cache maintenance and locking. Commands now run directly in the CLI, shrinking the failure blast radius and unlocking features that were previously awkward.
- `repair` now takes an exclusive lock.
- The builder no longer fails on a stale lock.
- Maintenance waits for its lock to drain between runs.
- `cached` runs correctly in the background on Windows and no longer trips over syslog setup.
- `prune -group-by` is implemented and documented; Kloset's `locate` also gained `-group-by` along with dataset and data-class filters.

### Performance

Measured with Korpus (1,000,000 items) on a 14-core Mac Mini with 64 GiB RAM and NVMe storage, compared to v1.0.6:

| Operation | v1.0.6      | v1.1.3     | Change |
| --------- | ----------- | ---------- | ------ |
| Backup    | ~3 minutes  | ~2 minutes | −33%   |
| Sync      | ~5 minutes  | ~4 minutes | −20%   |
| Restore   | ~60 minutes | ~3 minutes | −95%   |
| Check     | ~1 minute   | ~1 minute  | —      |

The restore improvement comes from a better algorithm, smarter parallelism, improved prefetch utilization, and removal of unnecessary system calls.

### Memory and disk

Peak RAM, same test environment:

| Operation | v1.0.6   | v1.1.3   | Change |
| --------- | -------- | -------- | ------ |
| Backup    | ~3.0 GiB | ~1.3 GiB | −43%   |
| Sync      | ~3.6 GiB | ~1.7 GiB | −52%   |
| Restore   | ~2.3 GiB | ~800 MiB | −66%   |
| Check     | ~1.3 GiB | ~800 MiB | −40%   |

Sources: fixed a gRPC-level memory leak affecting long-running backups over SFTP and S3; reworked the caching subsystem; default to spilling temporary data to disk rather than holding it in RAM.

Default on-disk cache footprint (1,000,000 items): reduced from ~4 GiB to ~1.8 GiB (−55%) by removing the on-disk VFS cache in favour of store queries. The old behaviour is available via flag for environments where bandwidth matters more than disk.

### Beta-to-release changes

- Significantly increased test coverage across plakar and Kloset, reaching 100% on several packages.
- Consolidated integrations into a single monorepo.
- `plakar mount` handles direct access to filesystem paths without browsing from the root.
