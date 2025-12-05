PLAKAR-DIGEST(1) - General Commands Manual

# NAME

**plakar-digest** - Compute digests for files in a Plakar snapshot

# SYNOPSIS

**plakar&nbsp;digest**
\[**-hashing**&nbsp;*algorithm*]
*snapshotID*\[:*path*]
\[...]

# DESCRIPTION

The
**plakar digest**
command computes and displays digests for specified
*path*
in a the given
*snapshotID*.
Multiple
*snapshotID*
and
*path*
may be given.
By default, the command computes the digest by reading the file
contents.

The options are as follows:

**-hashing** *algorithm*

> Use
> *algorithm*
> to compute the digest.
> Defaults to SHA256.

# EXAMPLES

Compute the digest of a file within a snapshot:

	$ plakar digest abc123:/etc/passwd

Use BLAKE3 as the digest algorithm:

	$ plakar digest -hashing BLAKE3 abc123:/etc/netstart

# DIAGNOSTICS

The **plakar-digest** utility exits&#160;0 on success, and&#160;&gt;0 if an error occurs.

0

> Command completed successfully.

&gt;0

> An error occurred, such as failure to retrieve a file digest or
> invalid snapshot ID.

# SEE ALSO

plakar(1)

Plakar - July 3, 2025
