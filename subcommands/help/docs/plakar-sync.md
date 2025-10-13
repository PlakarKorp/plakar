PLAKAR-SYNC(1) - General Commands Manual

# NAME

**plakar-sync** - Synchronize snapshots between Plakar repositories

# SYNOPSIS

**plakar&nbsp;sync**
\[**-packfiles**&nbsp;*path*]
\[*snapshotID*]
**to**&nbsp;|&nbsp;**from**&nbsp;|&nbsp;**with**
*repository*

# DESCRIPTION

The
**plakar sync**
command synchronize snapshots between two Plakar repositories.
If a specific snapshot ID is provided, only snapshots with matching
IDs will be synchronized.

**plakar sync**
supports the location flags documented in
plakar-query(7)
to precisely select snapshots.

The options are as follows:

**-packfiles** *path*

> Path where to put the temporary packfiles instead of building them in memory.
> If the special value
> 'memory'
> is specified then the packfiles are build in memory (the default value)

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

Synchronize all snapshots of @peer to @repo:

	$ plakar at @repo sync from @peer

# DIAGNOSTICS

The **plakar-sync** utility exits&#160;0 on success, and&#160;&gt;0 if an error occurs.

0

> Command completed successfully.

&gt;0

> General failure occurred, such as an invalid repository path, snapshot
> ID mismatch, or network error.

# SEE ALSO

plakar(1),
plakar-query(7)

Plakar - September 10, 2025 - PLAKAR-SYNC(1)
