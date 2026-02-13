PLAKAR-PRUNE(1) - General Commands Manual

# NAME

**plakar-prune** - Prune snapshots according to a policy

# SYNOPSIS

**plakar&nbsp;prune**
\[**-apply**]
\[**-policy**&nbsp;*name*]
\[*snapshotID&nbsp;...*]

# DESCRIPTION

The
**plakar prune**
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

**plakar prune**
supports the location flags documented in
plakar-query(7)
to precisely select snapshots.

The arguments are as follows:

**-apply**

> Delete matching snapshot.
> The default is to just show the snapshot that would be removed but not
> actually execute the operation.

**-policy** *name*

> Use the given policy.
> See
> plakar-policy(1)
> for how policies are managed.

# EXAMPLES

Remove a specific snapshot by ID:

	$ plakar prune abc123

Remove snapshots older than 30 days:

	$ plakar prune -days 30

Remove snapshots with a specific tag:

	$ plakar prune -tag daily-backup

Remove snapshots older than 1 year with a specific tag:

	$ plakar prune -years 1 -tag daily-backup

# DIAGNOSTICS

The **plakar-prune** utility exits&#160;0 on success, and&#160;&gt;0 if an error occurs.

0

> Command completed successfully.

&gt;0

> An error occurred, such as invalid date format or failure to delete a
> snapshot.

# SEE ALSO

plakar(1),
plakar-backup(1),
plakar-policy(1),
plakar-query(7)

Plakar - September 10, 2025
