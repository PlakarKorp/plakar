PLAKAR-PRUNE(1) - General Commands Manual

# NAME

**plakar-prune** - Prune snapshots according to a policy

# SYNOPSIS

**plakar&nbsp;prune**
\[**-apply**]
\[**-policy**&nbsp;*name*]

# DESCRIPTION

The
**plakar prune**
command deletes snapshots from a Plakar repository according to a
retention policy.
Filters such as
**-tag**
select the snapshots the policy applies to, and retention flags such as
**-days**
decide which of them to keep.
Selected snapshots that no retention rule keeps are deleted.
Without a retention rule, nothing is deleted.
To delete specific snapshots, use
plakar-rm(1).

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

# EXIT STATUS

The **plakar-prune** utility exits&#160;0 on success, and&#160;&gt;0 if an error occurs.

# EXAMPLES

Remove snapshots older than 30 days:

	$ plakar prune -days 30

Remove snapshots older than 1 year with a specific tag:

	$ plakar prune -years 1 -tag daily-backup

# SEE ALSO

plakar(1),
plakar-backup(1),
plakar-policy(1),
plakar-rm(1),
plakar-query(7)

Plakar - May 5, 2026
