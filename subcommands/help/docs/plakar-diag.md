PLAKAR-DIAG(1) - General Commands Manual

# NAME

**plakar-diag** - Display detailed information about Plakar internal structures

# SYNOPSIS

**plakar&nbsp;diag**
\[**chunks**&nbsp;|&nbsp;**contenttype**&nbsp;|&nbsp;**locks**&nbsp;|&nbsp;**object**&nbsp;|&nbsp;**packfile**&nbsp;|&nbsp;**snapshot**&nbsp;|&nbsp;**state**&nbsp;|&nbsp;**vfs**&nbsp;|&nbsp;**xattr**]

# DESCRIPTION

The
**plakar diag**
command provides detailed information about various internal data structures.
The type of information displayed depends on the specified argument.
Without any arguments, display information about the repository.

The sub-commands are as follows:

**chunks** *snapshotID*:*path*

> Display the list of chunks for a file within a snapshot, including
> the index, byte offset, length, MAC, and entropy of each chunk.

**contenttype** *snapshotID*:*path*

**locks**

> Display the list of locks currently held on the repository.

**object** *objectID*

> Display information about a specific object, including its mac,
> type, tags, and associated data chunks.

**packfile** *packfileID*

> Show details of packfiles, including entries and macs, which
> store object data within the repository.

**snapshot** *snapshotID*

> Show detailed information about a specific snapshot, including its
> metadata, directory and file count, and size.

**state**

> List or describe the states in the repository.

**vfs** *snapshotID*:*path*

> Show filesystem (VFS) details for a specific path within a snapshot,
> listing directory or file attributes, including permissions,
> ownership, and custom metadata.

**xattr** *snapshotID*:*path*

# EXIT STATUS

The **plakar-diag** utility exits&#160;0 on success, and&#160;&gt;0 if an error occurs.

# EXAMPLES

Show repository information:

	$ plakar diag

Show detailed information for a snapshot:

	$ plakar diag snapshot abc123

List all states in the repository:

	$ plakar diag state

Display a specific object within a snapshot:

	$ plakar diag object 1234567890abcdef

Display filesystem details for a path within a snapshot:

	$ plakar diag vfs abc123:/etc/passwd

# SEE ALSO

plakar(1),
plakar-backup(1)

Plakar - May 5, 2026 - PLAKAR-DIAG(1)
