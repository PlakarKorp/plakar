PLAKAR-MAINTENANCE(1) - General Commands Manual

# NAME

**plakar-maintenance** - Remove unused data from a Plakar repository

# SYNOPSIS

**plakar&nbsp;maintenance**

# DESCRIPTION

The
**plakar maintenance**
command removes unused blobs, objects, and chunks from a Plakar
repository to reduce storage space.
It identifies unreferenced data and reorganizes packfiles to ensure
only active snapshots and their dependencies are retained.
The maintenance process updates snapshot indexes to reflect these
changes.

# ENVIRONMENT

`PLAKAR_GRACEPERIOD`

> How long a packfile that is no longer referenced is kept before it is
> removed.
> The same delay protects a packfile that is orphaned but recent, so that
> a backup still in progress is left alone.
> The value is a duration, where an hour is
> "1h"
> and the seven day default is written
> "168h":
> days are not a unit, so the
> "7d"
> printed by the colouring pass is not a value this variable accepts.
> A value that cannot be parsed is replaced by the default without
> warning.

`PLAKAR_NODELETION`

> Set to a true boolean value to keep packfiles in the store.
> The repository state is still updated, so the data is no longer
> referenced by any snapshot, but nothing is removed from the store
> itself.

`PLAKAR_LOCKLESS`

> Set to a true boolean value to neither take nor check the exclusive
> repository lock.
> A concurrent backup or maintenance run against the same repository is
> then no longer prevented.

# EXIT STATUS

The **plakar-maintenance** utility exits&#160;0 on success, and&#160;&gt;0 if an error occurs.

# SEE ALSO

plakar(1)

Plakar - May 5, 2026 - PLAKAR-MAINTENANCE(1)
