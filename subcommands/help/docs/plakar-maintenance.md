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

# DIAGNOSTICS

The **plakar-maintenance** utility exits&#160;0 on success, and&#160;&gt;0 if an error occurs.

0

> Command completed successfully.

&gt;0

> An error occurred during maintenance, such as failure to update indexes
> or remove data.

# SEE ALSO

plakar(1)

Plakar - July 3, 2025
