PLAKAR-MCP(1) - General Commands Manual

# NAME

**plakar-mcp** - Serve a Kloset store over the Model Context Protocol

# SYNOPSIS

**plakar&nbsp;mcp**
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

**-max-output** *size*

> Maximum number of bytes of output returned by the
> **plakar**
> tool, and the largest file served as a resource.
> Longer output is cut and flagged as such in the result; a larger resource is
> refused.
> Defaults to 1048576.

# EXAMPLES

Serve the default Kloset store, as an MCP client would:

	$ plakar mcp

Serve a specific Kloset store, allowing longer output:

	$ plakar at /var/backups mcp -max-output 4194304

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
