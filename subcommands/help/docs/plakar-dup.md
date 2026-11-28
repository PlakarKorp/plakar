PLAKAR-DUP(1) - General Commands Manual

# NAME

**plakar-dup** - Duplicates an existing snapshot with a different ID

# SYNOPSIS

**plakar&nbsp;dup**
*snapshots&nbsp;...*

# DESCRIPTION

The
**plakar dup**
command creates a duplicate of an existing snapshot
with a new snapshot ID.
The new snapshot is an exact copy of the original,
including all files and metadata.

# EXIT STATUS

The **plakar-dup** utility exits&#160;0 on success, and&#160;&gt;0 if an error occurs.

# EXAMPLES

Create a duplicate of a snapshot with ID "abc123":

	$ plakar dup abc123

# SEE ALSO

plakar(1)

Plakar - May 5, 2026
