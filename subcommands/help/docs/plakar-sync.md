PLAKAR-SYNC(1) - General Commands Manual

# NAME

**plakar-sync** - Synchronize snapshots between Plakar repositories

# SYNOPSIS

**plakar&nbsp;sync**
\[**-name**&nbsp;*name*]
\[**-category**&nbsp;*category*]
\[**-environment**&nbsp;*environment*]
\[**-perimeter**&nbsp;*perimeter*]
\[**-job**&nbsp;*job*]
\[**-tag**&nbsp;*tag*]
\[**-latest**]
\[**-before**&nbsp;*date*]
\[**-since**&nbsp;*date*]
\[*snapshotID*]
**to**&nbsp;|&nbsp;**from**&nbsp;|&nbsp;**with**
*repository*

# DESCRIPTION

The
**plakar sync**
command synchronize snapshots between two Plakar repositories.
If a specific snapshot ID is provided, only snapshots with matching
IDs will be synchronized.

The options are as follows:

**-name** *string*

> Only apply command to snapshots that match
> *name*.

**-category** *string*

> Only apply command to snapshots that match
> *category*.

**-environment** *string*

> Only apply command to snapshots that match
> *environment*.

**-perimeter** *string*

> Only apply command to snapshots that match
> *perimeter*.

**-job** *string*

> Only apply command to snapshots that match
> *job*.

**-tag** *string*

> Only apply command to snapshots that match
> *tag*.

**-latest**

> Only apply command to latest snapshot matching filters.

**-before** *date*

> Only apply command to snapshots matching filters and older than the specified
> date.
> Accepted formats include relative durations
> (e.g. 2d for two days, 1w for one week)
> or specific dates in various formats
> (e.g. 2006-01-02 15:04:05).

**-since** *date*

> Only apply command to snapshots matching filters and created since the specified
> date, included.
> Accepted formats include relative durations
> (e.g. 2d for two days, 1w for one week)
> or specific dates in various formats
> (e.g. 2006-01-02 15:04:05).

The arguments are as follows:

**to** | **from** | **with**

> Specifies the direction of synchronization:

> **to**

> > Synchronize snapshots from the local repository to the specified peer
> > repository.

> **from**

> > Synchronize snapshots from the specified peer repository to the local
> > repository.

> **with**

> > Synchronize snapshots in both directions, ensuring both repositories
> > are fully synchronized.

*repository*

> Path to the peer repository to synchronize with.

# EXAMPLES

Synchronize the snapshot
'abcd'
with a peer repository:

	$ plakar sync abcd to @peer

Bi-directional synchronization with peer repository of recent snapshots:

	$ plakar sync -since 7d with @peer

# DIAGNOSTICS

The **plakar-sync** utility exits&#160;0 on success, and&#160;&gt;0 if an error occurs.

0

> Command completed successfully.

&gt;0

> General failure occurred, such as an invalid repository path, snapshot
> ID mismatch, or network error.

# SEE ALSO

plakar(1)

Plakar - July 3, 2025
