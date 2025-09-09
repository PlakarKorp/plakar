PLAKAR-LS(1) - General Commands Manual

# NAME

**plakar-ls** - List snapshots and their contents in a Plakar repository

# SYNOPSIS

**plakar&nbsp;ls**
\[**-uuid**]
\[**-name**&nbsp;*name*]
\[**-category**&nbsp;*category*]
\[**-environment**&nbsp;*environment*]
\[**-perimeter**&nbsp;*perimeter*]
\[**-job**&nbsp;*job*]
\[**-tag**&nbsp;*tag*]
\[**-tags**]
\[**-latest**]
\[**-before**&nbsp;*date*]
\[**-since**&nbsp;*date*]
\[**-recursive**]
\[*snapshotID*:*path*]

# DESCRIPTION

The
**plakar ls**
command lists snapshots stored in a Plakar repository, and optionally
displays the contents of
*path*
in a specified snapshot.

The options are as follows:

**-name** *name*

> Only apply command to snapshots that match
> *name*.

**-category** *category*

> Only apply command to snapshots that match
> *category*.

**-environment** *environment*

> Only apply command to snapshots that match
> *environment*.

**-perimeter** *perimeter*

> Only apply command to snapshots that match
> *perimeter*.

**-job** *job*

> Only apply command to snapshots that match
> *job*.

**-tag** *tag*

> Filter snapshots by the specified tag, listing only those that contain
> the given tag.

**-tags**

> Display tags when listing snapshots.

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

plakar(1)

Plakar - July 3, 2025
