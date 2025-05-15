# plakar - Effortless backup

## Introduction

plakar provides an intuitive, powerful, and scalable backup solution.

Plakar goes beyond file-level backups. It captures application data with its full context.

Data and context are stored using [Kloset](https://www.plakar.io/posts/2025-04-29/kloset-the-immutable-data-store/), an open-source, immutable data store that enables the implementation of advanced data protection scenarios.

Plakar's main strengths:
- **Effortless**: Easy to use, clean default. Check out our [quick start guide](https://docs.plakar.io/en/quickstart/).
- **Secure**: Provide audited end-to-end encryption for data and metadata. See our latest [crypto audit report](http://localhost:1313/docs/audits/).
- **Reliable**: Backups are stored in Kloset, an open-source immutable data store. Learn more about [Kloset](http://localhost:1313/posts/2025-04-29/kloset-the-immutable-data-store/).
- **Vertically scalable**: Backup and restore very large datasets with limited RAM usage.
- **Horizontally scalable**: Support high concurrency and multiple backups type in a single Kloset.
- **Browsable**: Browse, sort, search, and compare backups using the Plakar UI.
- **Fast**: backup, check, sync and restore are  operations are optimized for large-scale data.
- **Efficient**: more restore points, less storage, thanks to Kloset's unmatched deduplication and compression.
- **Open Source and actively maintained**: open source forever and now maintained by [Plakar Korp](https://www.plakar.io)

Simplicity and efficiency are plakar's main priorities.

Our mission is to set a new standard for effortless secure data protection. 

## Plakar UI

Plakar includes a built-in web-based user interface to **monitor, browse, and restore** your backups with ease.

### 🖥️ Launch the UI

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


## Requirement

`plakar` requires Go 1.23.3 or higher,
it may work on older versions but hasn't been tested.


## Installing the CLI

```
go install github.com/PlakarKorp/plakar/cmd/plakar@latest
```

## Quickstart

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

## Notable Capabilities

- **Instant recovery**: Instantly mount large backups on any devices without full restoration.
- **Distributed backup**: Kloset can be easily distributed to implement 3,2,1 rule or advanced strategies (push, pull, sync) across heterogeneous environments.
- **Granular restore**: Restore a complete snapshot or only a subset of your data.
- **Cross-storage restore**: Back up from one storage type (e.g., S3-compatible object store) and restore to another (e.g., file system)..
- **Production safe-guarding**: Automatically adjusts backup speed to avoid impacting production workloads.
- **Lock-free maintenance**: Perform garbage collection without interrupting backup or restore operations.
- **Integrations**: back up and restore from and to any source (file systems, object stores, SaaS applications...) with the right integration.


## Documentation

For the latest information,
you can read the documentation available at https://docs.plakar.io

## Community

You can join our very active [Discord](https://discord.gg/uuegtnF2Q5) to discuss the project !

## Warning

plakar is currently in beta and **NOT** production ready yet but it is most definitely stable enough to be tested by others.

Feel free to give it a try, give feedback on what you like/dislike and report bugs.
