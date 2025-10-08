<div align="center">

<img src="./docs/assets/Plakar_Logo_Simple_Pirmary.png" alt="Plakar Backup & Restore Solution" width="200"/>

# plakar - Effortless backup & more

[![Join our Discord community](https://img.shields.io/badge/Discord-Join%20Us-purple?logo=discord&logoColor=white&style=for-the-badge)](https://discord.gg/A2yvjS6r2C)
[![Subscribe on YouTube](https://img.shields.io/badge/YouTube-Subscribe-red?logo=youtube&logoColor=white&style=for-the-badge)](https://www.youtube.com/@PlakarKorp)
[![Join our Subreddit](https://img.shields.io/badge/Reddit-Join%20r%2Fplakar-orange?logo=reddit&logoColor=white&style=for-the-badge)](https://www.reddit.com/r/plakar/)

[Deutsch](https://www.readme-i18n.com/PlakarKorp/plakar?lang=de) |
[Español](https://www.readme-i18n.com/PlakarKorp/plakar?lang=es) |
[français](https://www.readme-i18n.com/PlakarKorp/plakar?lang=fr) |
[日本語](https://www.readme-i18n.com/PlakarKorp/plakar?lang=ja) |
[한국어](https://www.readme-i18n.com/PlakarKorp/plakar?lang=ko) |
[Português](https://www.readme-i18n.com/PlakarKorp/plakar?lang=pt) |
[Русский](https://www.readme-i18n.com/PlakarKorp/plakar?lang=ru) |
[中文](https://www.readme-i18n.com/PlakarKorp/plakar?lang=zh)

</div>

## 🔄 Latest Release

### **V1.0.4 - Major Release: Plugins, Windows, Packages, Performance** *(September 16, 2025)*

- **Pre-packaged binaries** for easy installs: `.deb`, `.rpm`, `.apk`, plus static tarballs.  
  Package repositories coming right after to install via `apt`, `yum`, or `apk`.
- **Initial Windows support**: Plakar now runs natively on Windows, including CLI and UI.  
  Current limitation: one concurrent operation per agent, as multi-agent support is coming next.
- **Integrations as plugins** with `plakar pkg add <integration>`  
  Example: `plakar pkg add s3`, `plakar pkg add sftp`, `plakar pkg add gcp`, `imap`, `ftp`, ...
- **Smarter agent**: auto-spawn and auto-teardown after idle for frictionless concurrency.
- **Cache improvements**: fewer disk hits, lower footprint, better accuracy on very large corpora.
- **Performance boosts** across backup, check, restore: faster indexing, traversal, data access, and dedupe pipelines.  
  From x2 to x10 depending on workloads.
- **Policy-based lifecycle** via `plakar prune`  
  Examples:  
  `plakar prune -days 2 -per-day 3 -weeks 4 -per-week 5 -months 3 -per-month 2`  
  `plakar prune -tags finance -per-day 5`
- **UI refinements**: cleaner layouts, clearer hierarchy, better progress and error messages.  
  Try the demo: https://demo.plakar.io

[📝 Release article](https://plakar.io/posts/2025-09-16/release-v1.0.4-a-new-milestone-for-plakar/)

## 🧭 Introduction

plakar provides an intuitive, powerful, and scalable backup solution.

Plakar goes beyond file-level backups. It captures application data with its full context.

Data and context are stored using [Kloset](https://www.plakar.io/posts/2025-04-29/kloset-the-immutable-data-store/), an open-source, immutable data store that enables the implementation of advanced data protection scenarios.

Plakar's main strengths:
- **Effortless**: Easy to use, clean default. Check out our [quick start guide](https://www.plakar.io/docs/v1.0.4/quickstart/).
- **Secure**: Provide audited end-to-end encryption for data and metadata. See our latest [crypto audit report](https://www.plakar.io/posts/2025-02-28/audit-of-plakar-cryptography/).
- **Reliable**: Backups are stored in Kloset, an open-source immutable data store. Learn more about [Kloset](https://www.plakar.io/posts/2025-04-29/kloset-the-immutable-data-store/).
- **Vertically scalable**: Backup and restore very large datasets with limited RAM usage.
- **Horizontally scalable**: Support high concurrency and multiple backups type in a single Kloset.
- **Browsable**: Browse, sort, search, and compare backups using the Plakar UI.
- **Fast**: backup, check, sync and restore are  operations are optimized for large-scale data.
- **Efficient**: more restore points, less storage, thanks to Kloset's unmatched [deduplication](https://www.plakar.io/posts/2025-07-11/introducing-go-cdc-chunkers-chunk-and-deduplicate-everything/) and compression.
- **Open Source and actively maintained**: open source forever and now maintained by [Plakar Korp](https://www.plakar.io)

Simplicity and efficiency are plakar's main priorities.

Our mission is to set a new standard for effortless secure data protection. 

## 🖥️ Plakar UI

Plakar includes a built-in web-based user interface to **monitor, browse, and restore** your backups with ease.

### 🚀 Launch the UI

You can start the interface from any machine with access to your backups:

```
$ plakar ui
```

### 📂 Snapshot Overview

Quickly list all available snapshots and explore them:

![Snapshot browser](https://www.plakar.io/readme/snapshot-list.png)

### 🔍 Granular Browsing

Navigate the contents of each snapshot to inspect, compare, or selectively restore files:

![Snapshot browser](https://www.plakar.io/readme/snapshot-browser.png)




## 📦 Installing the CLI

### From binaries

Visit https://www.plakar.io/download/

### From source

`plakar` requires Go 1.23.3 or higher,
it may work on older versions but hasn't been tested.

```
go install github.com/PlakarKorp/plakar@latest
```

## 🚀 Quickstart

plakar quickstart: https://www.plakar.io/docs/v1.0.4/quickstart/

A taste of plakar (please follow the quickstart to begin):
```
$ plakar at /var/backups create                             # Create a repository
$ plakar at /var/backups backup /private/etc                # Backup /private/etc
$ plakar at /var/backups ls                                 # List all repository backup
$ plakar at /var/backups restore -to /tmp/restore 9abc3294  # Restore a backup to /tmp/restore
$ plakar at /var/backups ui                                 # Start the UI
$ plakar at /var/backups sync to @s3                        # Synchronise a backup repository to S3

```

## 🧠 Notable Capabilities

- **Instant recovery**: Instantly mount large backups on any devices without full restoration.
- **Distributed backup**: Kloset can be easily distributed to implement 3,2,1 rule or advanced strategies (push, pull, sync) across heterogeneous environments.
- **Granular restore**: Restore a complete snapshot or only a subset of your data.
- **Cross-storage restore**: Back up from one storage type (e.g., S3-compatible object store) and restore to another (e.g., file system)..
- **Production safe-guarding**: Automatically adjusts backup speed to avoid impacting production workloads.
- **Lock-free maintenance**: Perform garbage collection without interrupting backup or restore operations.
- **Integrations**: back up and restore from and to any source (file systems, object stores, SaaS applications...) with the right integration.

## 🗄️ Plakar archive format : ptar

[ptar](https://www.plakar.io/posts/2025-06-27/it-doesnt-make-sense-to-wrap-modern-data-in-a-1979-format-introducing-.ptar/) is Plakar’s lightweight, high-performance archive format for secure and efficient backup snapshots.

[Kapsul](https://www.plakar.io/posts/2025-07-07/kapsul-a-tool-to-create-and-manage-deduplicated-compressed-and-encrypted-ptar-vaults/) is a companion tool that lets you run most plakar sub-commands directly on a .ptar archive without extracting it.
It mounts the archive in memory as a read-only Plakar repository, enabling transparent and efficient inspection, restoration, and diffing of snapshots.

For installation, usage examples, and full documentation, see the [Kapsul repository](https://github.com/PlakarKorp/kapsul).

## 📚 Documentation

For the latest information,
you can read the documentation available at https://www.plakar.io/docs/v1.0.4/

## 💬 Community

- 🗨️ Join our very active [Discord](https://discord.gg/uqdP9Wfzx3)
- 📣 Follow our subreddit [r/plakar](https://www.reddit.com/r/plakar/)
- ▶️ Subscribe to our YouTube channel [@PlakarKorp](https://www.youtube.com/@PlakarKorp)
