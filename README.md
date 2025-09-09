<div align="center">

<img src="./docs/assets/Plakar_Logo_Simple_Pirmary.png" alt="Plakar Backup & Restore Solution" width="200"/>

# plakar - Effortless backup & more

[![Join our Discord community](https://img.shields.io/badge/Discord-Join%20Us-purple?logo=discord&logoColor=white&style=for-the-badge)](https://discord.gg/A2yvjS6r2C)


[Deutsch](https://www.readme-i18n.com/PlakarKorp/plakar?lang=de) |
[Español](https://www.readme-i18n.com/PlakarKorp/plakar?lang=es) |
[français](https://www.readme-i18n.com/PlakarKorp/plakar?lang=fr) |
[日本語](https://www.readme-i18n.com/PlakarKorp/plakar?lang=ja) |
[한국어](https://www.readme-i18n.com/PlakarKorp/plakar?lang=ko) |
[Português](https://www.readme-i18n.com/PlakarKorp/plakar?lang=pt) |
[Русский](https://www.readme-i18n.com/PlakarKorp/plakar?lang=ru) |
[中文](https://www.readme-i18n.com/PlakarKorp/plakar?lang=zh)

</div>




## ⚙️ Requirement

`plakar` requires Go 1.24.0 or higher,
it may work on older versions but hasn't been tested.

On systems that package older versions,
such as Debian or Ubuntu,
it is preferable to install the latest version from the [official website](https://go.dev/doc/install).


## 📦 Installing the CLI

```
go install github.com/PlakarKorp/plakar@latest
```


## 🔄 Latest Releases

### **V1.0.2 – Minor Release: S3 Performance Boost** *(June 2, 2025)*

- Achieved a **60× performance improvement** for backups over S3.  
  A backup that previously took ~14 minutes now completes in ~13 seconds.

[📝 Tech blog post](https://www.plakar.io/posts/2025-06-03/plakar-v1.0.2-was-released-mostly-s3-improvements/)

### **V1.0.1 – Major Release: Plakar is Production-Ready** *(May 15, 2025)*

- **Plakar is now stable and production-ready**, marking a major milestone in our open-source journey.
- Introduced **long-term support for our immutable storage engine**, [**Kloset**](https://www.plakar.io/posts/2025-04-29/kloset-the-immutable-data-store/).

[📝 Tech blog post](https://www.plakar.io/posts/2025-05-01/introducing-plakar-v1.0-to-redefine-open-source-data-protection-with-3m-funding/)

## 🧭 Introduction

plakar provides an intuitive, powerful, and scalable backup solution.

Plakar goes beyond file-level backups. It captures application data with its full context.

Data and context are stored using [Kloset](https://www.plakar.io/posts/2025-04-29/kloset-the-immutable-data-store/), an open-source, immutable data store that enables the implementation of advanced data protection scenarios.

Plakar's main strengths:
- **Effortless**: Easy to use, clean default. Check out our [quick start guide](https://docs.plakar.io/en/quickstart/).
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


## 🚀 Quickstart

plakar quickstart: https://docs.plakar.io/en/quickstart/

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
you can read the documentation available at https://docs.plakar.io

## 💬 Community

You can join our very active [Discord](https://discord.gg/uqdP9Wfzx3) to discuss the project !
