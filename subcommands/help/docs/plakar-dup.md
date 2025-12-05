PLAKAR-DUP(1) - General Commands Manual

# NAME

**plakar-dup** - Duplicates an existing snapshot with a different ID

# SYNOPSIS

**plakar&nbsp;dup**

# DESCRIPTION

The
**plakar dup**
command creates a duplicate of an existing snapshot
with a new snapshot ID.
The new snapshot is an exact copy of the original,
including all files and metadata.

# EXAMPLES

Create a duplicate of a snapshot with ID "abc123":

	$ plakar dup abc123

# DIAGNOSTICS

The **plakar-dup** utility exits&#160;0 on success, and&#160;&gt;0 if an error occurs.

0

> Command completed successfully.

&gt;0

> An error occurred, such as failure to retrieve existing snapshot or
> invalid snapshot ID.

# SEE ALSO

plakar(1)

Plakar - July 3, 2025
