PLAKAR-LS(1) - General Commands Manual

# NAME

**plakar-ls** - List snapshots and their contents in a Plakar repository

# SYNOPSIS

**plakar&nbsp;ls**
\[**-uuid**]
\[**-recursive**]
\[*snapshotID*:*path*]

# DESCRIPTION

The
**plakar ls**
command lists snapshots stored in a Plakar repository, and optionally
displays the contents of
*path*
in a specified snapshot.

In addition to the flags described below,
**plakar ls**
supports the location flags documented in
plakar-query(7)
to precisely select snapshots.

The options are as follows:

**-uuid**

> Display the full UUID for each snapshot instead of the shorter
> snapshot ID.

**-recursive**

> List directory contents recursively when exploring snapshot contents.

# EXAMPLES

List all snapshots with their short IDs:

	$ plakar ls

List all snapshots with UUIDs instead of short IDs:

	$ plakar ls -uuid

List snapshots with a specific tag:

	$ plakar ls -tag daily-backup

List contents of a specific snapshot:

	$ plakar ls abc123

Recursively list contents of a specific snapshot:

	$ plakar ls -recursive abc123:/etc

# DIAGNOSTICS

The **plakar-ls** utility exits&#160;0 on success, and&#160;&gt;0 if an error occurs.

0

> Command completed successfully.

&gt;0

> An error occurred, such as failure to retrieve snapshot information or
> invalid snapshot ID.

# SEE ALSO

plakar(1),
plakar-query(7)

Plakar - September 10, 2025
