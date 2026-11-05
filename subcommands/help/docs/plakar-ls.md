PLAKAR-LS(1) - General Commands Manual

# NAME

**plakar-ls** - List snapshots and their contents in a Plakar repository

# SYNOPSIS

**plakar&nbsp;ls**
\[**-uuid**]
\[**-recursive**]
\[**-tags**]
\[*snapshotID*:*path*]

# DESCRIPTION

The
**plakar ls**
command lists snapshots stored in a Plakar repository, and optionally
displays the contents of
*path*
in a specified snapshot.

Files that could not be read when the snapshot was made are missing
from the listing.
When listing the contents of a snapshot,
**plakar ls**
reports the errors recorded for them on standard error, interleaved
with the listing.
Errors recorded below the listed directory are only reported when
**-recursive**
is given.

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

**-tags**

> Show tags in snapshot listing.

# EXIT STATUS

The **plakar-ls** utility exits&#160;0 on success, and&#160;&gt;0 if an error occurs.

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

# SEE ALSO

plakar(1),
plakar-query(7)

Plakar - May 5, 2026
