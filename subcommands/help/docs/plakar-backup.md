PLAKAR-BACKUP(1) - General Commands Manual

# NAME

**plakar-backup** - Create a new snapshot in a Kloset store

# SYNOPSIS

**plakar&nbsp;backup**
\[**--concurrency**&nbsp;*number*]
\[**-x&nbsp;|&nbsp;--exclude**&nbsp;*pattern*]
\[**--exclude-file**&nbsp;*file*]
\[**-c&nbsp;|&nbsp;--check**]
\[**-o&nbsp;|&nbsp;--option**&nbsp;*option*]
\[**-q&nbsp;|&nbsp;--quiet**]
\[**-s&nbsp;|&nbsp;--silent**]
\[**-t&nbsp;|&nbsp;--tag**&nbsp;*tag*]
\[**-n&nbsp;|&nbsp;--scan**]
\[*place*]

# DESCRIPTION

The
**plakar backup**
command creates a new snapshot of
*place*,
or the current directory.
Snapshots can be filtered to exclude specific files or directories
based on patterns provided through options.

*place*
can be either a path, an URI, or a label with the form
"@*name*"
to reference a source connector configured with
plakar-source(1).

The options are as follows:

**--concurrency** *number*

> Set the maximum number of parallel tasks for faster processing.
> Defaults to
> `8 * CPU count + 1`.

**-x, --exclude** *pattern*

> Specify individual glob exclusion patterns to ignore files or
> directories in the backup.
> This option can be repeated.

**--exclude-file** *file*

> Specify a file containing glob exclusion patterns, one per line, to
> ignore files or directories in the backup.

**-c, --check**

> Perform a full check on the backup after success.

**-o, --option** *option*

> Can be used to pass extra arguments to the source connector.
> The given
> *option*
> takes precedence over the configuration file.

**-q, --quiet**

> Suppress output to standard input, only logging errors and warnings.

**-s, --silent**

> Suppress all output.

**-t, --tag** *tag*

> Comma-separated list of tags to apply to the snapshot.

**-n, --scan**

> Do not write a snapshot; instead, perform a dry run by outputting the list of
> files and directories that would be included in the backup.
> Respects all exclude patterns and other options, but makes no changes to the
> Kloset store.

# EXAMPLES

Create a snapshot of the current directory with two tags:

	$ plakar backup --tag daily-backup,production

Backup a specific directory with exclusion patterns from a file:

	$ plakar backup --exclude-file ~/my-excludes-file /var/www

Backup a directory with specific file exclusions:

	$ plakar backup --exclude "*.tmp" -exclude "*.log" /var/www

# DIAGNOSTICS

The **plakar-backup** utility exits&#160;0 on success, and&#160;&gt;0 if an error occurs.

0

> Command completed successfully; a snapshot was created, but some items may have
> been skipped (for example, files without sufficient permissions).
> Run
> plakar-info(1)
> on the new snapshot to view any errors.

&gt;0

> An error occurred, such as failure to access the Kloset store or issues
> with exclusion patterns.

# SEE ALSO

plakar(1),
plakar-source(1)

Plakar - July 3, 2025
