PLAKAR-MCP(1) - General Commands Manual

# NAME

**plakar-mcp** - Serve a Kloset store over the Model Context Protocol

# SYNOPSIS

**plakar&nbsp;mcp**
\[**-allow-backup**]
\[**-allow-delete**]
\[**-allow-restore**&nbsp;**-restore-root**&nbsp;*directory*]
\[**-allow-sync**]
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
delete a snapshot or write to the local filesystem; the write tools described
under
**-allow-backup**,
**-allow-delete**,
**-allow-restore**
and
**-allow-sync**
are not merely hidden but absent, and calling one is answered with an unknown
tool error.

The server also exposes the files stored in snapshots as MCP resources, under
URIs of the form
**plakar://SNAPSHOT/PATH**,
so a client can attach a backed-up file directly.

The read-only tools are:

**list\_snapshots**

> List the snapshots stored in the Kloset store, with their ID, creation time,
> source directory, size and tags.
> The listing can be filtered by tag and paged with an offset and a limit.

**list\_tags**

> List the tags carried by the snapshots, with how many snapshots carry each.

**repository\_health**

> Summarize the state of the backups: how fresh the latest snapshot is, a
> per-source breakdown to spot a source whose backups stopped, sizes and held
> locks.

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
> returned, and a file larger than
> **-max-file-size**
> can be paged through with a byte offset.

**file\_history**

> Report every version of a file across all snapshots, newest first, flagging
> the snapshots where the content changed.

**compare\_snapshots**

> List the entries added, removed or modified between two snapshots.

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

**-allow-restore**

> Register the
> **restore\_files**
> tool, which writes snapshot contents to the local filesystem.
> Restores are confined to the directory given with
> **-restore-root**,
> which must exist; a destination that resolves outside it is refused.
> Disabled by default.

**-restore-root** *directory*

> The one directory
> **restore\_files**
> may write under.
> Required by, and only meaningful with,
> **-allow-restore**.

**-allow-sync**

> Register the
> **sync\_snapshots**
> tool, which pushes snapshots to a peer repository named in the
> **plakar**
> configuration.
> Only peers from the configuration are accepted, only the push direction is
> offered, and an encrypted peer must have its passphrase configured: the server
> never prompts.
> Disabled by default.

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

**restore\_files**

> Restore a file or directory from a snapshot under the restore root.
> Requires
> **-allow-restore**.

**sync\_snapshots**

> Push snapshots to a peer repository from the configuration.
> Requires
> **-allow-sync**.

# EXAMPLES

Serve the default Kloset store, as an MCP client would:

	$ plakar mcp

Serve a specific Kloset store, allowing larger files to be read:

	$ plakar at /var/backups mcp -max-file-size 4194304

Allow a client to create snapshots, but not to remove any:

	$ plakar mcp -allow-backup

Allow restores, confined to a scratch directory:

	$ plakar mcp -allow-restore -restore-root /var/tmp/restores

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

Plakar - September 11, 2026
