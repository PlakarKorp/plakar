PLAKAR-MCP(1) - General Commands Manual

# NAME

**plakar-mcp** - Serve a Kloset store over the Model Context Protocol

# SYNOPSIS

**plakar&nbsp;mcp**
\[**-allow-backup**]
\[**-allow-delete**]
\[**-allow-restore**&nbsp;**-restore-root**&nbsp;*directory*]
\[**-allow-sync**]
\[**-max-output**&nbsp;*size*]

# DESCRIPTION

The
**plakar mcp**
command starts a Model Context Protocol (MCP) server exposing a Kloset store
to an MCP client, such as an AI assistant or an editor.

By default the server speaks JSON-RPC over standard input and output, the
transport MCP clients use when they spawn the server as a child process.
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

Reading goes through a single tool,
**plakar**,
which runs a read-only
**plakar**
command against the served Kloset store, in the server process, and returns
what it printed: its standard output, its standard error and its exit code.
A command answers exactly as it does on the command line, so there is no
second implementation of it to keep in step with the first.
The commands the tool runs are:

**ls**

> List the snapshots, or the entries of a directory in a snapshot.

**cat**

> Print the contents of files in a snapshot.

**info**

> Describe the Kloset store or a snapshot, including the entries a snapshot
> could not read.

**locate**

> Find the snapshots holding files whose name matches a pattern.

**diff**

> Compare a file or directory between two snapshots.

**digest**

> Compute the digest of files in a snapshot.

**check**

> Verify the integrity of snapshots.

**diag**

> Describe the internals of the Kloset store, through its
> **snapshot**,
> **locks**,
> **vfs**,
> **xattr**,
> **contenttype**,
> **search**,
> **chunks**,
> **object**,
> **blob**,
> **packfile**
> and
> **state**
> subcommands.

**help**

> Show the manual of a command, as Markdown.

Every other command is refused by name before its arguments are parsed, so no
command that writes to the Kloset store or to the local filesystem can be
reached through the tool, whatever its arguments.
Output past
**-max-output**
bytes is cut, the command is stopped, and the result says so.

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

**-max-output** *size*

> Maximum number of bytes of output returned by the
> **plakar**
> tool, and the largest file served as a resource.
> Longer output is cut and flagged as such in the result; a larger resource is
> refused.
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

Serve a specific Kloset store, allowing longer output:

	$ plakar at /var/backups mcp -max-output 4194304

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
plakar-diag(1),
plakar-diff(1),
plakar-digest(1),
plakar-info(1),
plakar-locate(1),
plakar-ls(1)

Plakar - November 13, 2026
