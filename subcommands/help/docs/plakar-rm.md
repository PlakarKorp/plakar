PLAKAR-RM(1) - General Commands Manual

# NAME

**plakar-rm** - Remove snapshots from a Plakar repository

# SYNOPSIS

**plakar&nbsp;rm**
\[**-apply**]
\[*snapshotID&nbsp;...*]

# DESCRIPTION

The
**plakar rm**
command deletes snapshots from a Plakar repository.
Snapshots can be filtered for deletion by age, by tag, or by
specifying the snapshot IDs to remove.
If no
*snapshotID*
are provided, either
**-older**
or
**-tag**
must be specified to filter the snapshots to delete.

In addition to the flags described below,
**plakar ls**
supports the location flags documented in
plakar-query(7)
to precisely select snapshots.

The arguments are as follows:

**-apply**

> Delete the matching snapshots.
> By default,
> **plakar rm**
> only prints the snapshots that would be deleted.

# EXIT STATUS

The **plakar-rm** utility exits&#160;0 on success, and&#160;&gt;0 if an error occurs.

# EXAMPLES

Remove a specific snapshot by ID:

	$ plakar rm abc123

Remove snapshots older than 30 days:

	$ plakar rm -before 30d

Remove snapshots with a specific tag:

	$ plakar rm -tag daily-backup

Remove snapshots older than 1 year with a specific tag:

	$ plakar rm -before 1y -tag daily-backup

# SEE ALSO

plakar(1),
plakar-backup(1)

Plakar - May 5, 2026
