PLAKAR-MCP(1) - General Commands Manual

# NAME

**plakar-mcp** - Serve a Kloset store over the Model Context Protocol

# SYNOPSIS

**plakar&nbsp;mcp**
\[**-allow-backup**]
\[**-allow-delete**]
\[**-max-file-size**&nbsp;*size*]

# DESCRIPTION

The
**plakar mcp**
command starts a Model Context Protocol (MCP) server exposing a Kloset store
to an MCP client, such as an AI assistant or an editor.

The server speaks JSON-RPC over standard input and output, the transport MCP
clients use when they spawn the server as a child process.
It is not meant to be run interactively: standard output carries the protocol
stream, so everything normally printed to standard output, including progress
reports, is redirected to standard error for the lifetime of the server.

By default the exposed tools are read-only, and no tool can create, modify or
delete a snapshot; the write tools described under
**-allow-backup**
and
**-allow-delete**
are not merely hidden but absent, and calling one is answered with an unknown
tool error.

The read-only tools are:

**list\_snapshots**

> List the snapshots stored in the Kloset store, with their ID, creation time,
> source directory, size and tags.

**snapshot\_details**

> Report the full metadata of a snapshot: when and where it was taken, by whom,
> and a summary of what it contains.

**repository\_info**

> Report configuration, storage and logical size information about the Kloset
> store.

**repository\_locks**

> List the locks currently held on the Kloset store, to tell whether another
> **plakar**
> process is operating on it.

**list\_files**

> List the files and directories contained in a snapshot, optionally recursively.

**search\_files**

> Search for files across one or every snapshot, by name pattern and/or MIME
> type.

**stat\_entry**

> Report the detailed metadata of a single entry: ownership, mode, content type,
> entropy, extended attributes and classifications.

**read\_file**

> Read the contents of a regular file stored in a snapshot.
> Files that are not valid UTF-8 text are reported as binary rather than
> returned.

**diff\_snapshots**

> Compare the same file in two snapshots and return a unified diff.
> Unlike
> plakar-diff(1),
> this only ever compares two snapshots against each other, never a snapshot
> against the local filesystem.

**digest\_file**

> Compute the cryptographic digest of a file stored in a snapshot.

**check\_snapshot**

> Verify the integrity of a snapshot.
> Metadata is checked by default; a deep check re-reads and rehashes every chunk,
> which is accurate but slow and I/O heavy on large snapshots.

**snapshot\_errors**

> List the entries that could not be read when a snapshot was created, to tell an
> incomplete backup from a complete one.

The options are as follows:

**-allow-backup**

> Register the
> **create\_backup**
> tool, which backs up a directory into the repository and creates a snapshot.
> Note that the source is read from the local filesystem, so a client may read any
> path this process can read and commit it to the repository.
> Disabled by default.

**-allow-delete**

> Register the
> **remove\_snapshots**
> and
> **prune\_snapshots**
> tools, which delete snapshots permanently and cannot be undone.
> Both report what they would remove and only act when passed
> *apply*.
> Disabled by default, and independent of
> **-allow-backup**
> so that backups can be enabled without arming deletion.

**-max-file-size** *size*

> Maximum number of bytes returned by the
> **read\_file**
> tool.
> Larger files are truncated, and flagged as such in the result.
> Defaults to 1048576.

The write tools are:

**create\_backup**

> Back up a directory into the repository.
> Requires
> **-allow-backup**.

**remove\_snapshots**

> Permanently remove the named snapshots.
> Requires
> **-allow-delete**.

**prune\_snapshots**

> Permanently remove the snapshots the retention policy selects.
> Requires
> **-allow-delete**.

# EXAMPLES

Serve the default Kloset store, as an MCP client would:

	$ plakar mcp

Serve a specific Kloset store, allowing larger files to be read:

	$ plakar at /var/backups mcp -max-file-size 4194304

Allow a client to create snapshots, but not to remove any:

	$ plakar mcp -allow-backup

# DIAGNOSTICS

The **plakar-mcp** utility exits&#160;0 on success, and&#160;&gt;0 if an error occurs.

# SEE ALSO

plakar(1),
plakar-cat(1),
plakar-check(1),
plakar-diff(1),
plakar-digest(1),
plakar-info(1),
plakar-locate(1),
plakar-ls(1)

Plakar - August 13, 2026 - PLAKAR-MCP(1)
