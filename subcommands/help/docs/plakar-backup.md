PLAKAR-BACKUP(1) - General Commands Manual

# NAME

**plakar-backup** - Create a new snapshot in a Kloset store

# SYNOPSIS

**plakar&nbsp;backup**
\[**-concurrency**&nbsp;*number*]
\[**-force-timestamp**&nbsp;*timestamp*]
\[**-ignore**&nbsp;*pattern*]
\[**-ignore-file**&nbsp;*file*]
\[**-check**]
\[**-no-xattr**]
\[**-o**&nbsp;*option*=*value*]
\[**-packfiles**&nbsp;*path*]
\[**-quiet**]
\[**-silent**]
\[**-tag**&nbsp;*tag*]
\[**-scan**]
\[*place*]

# DESCRIPTION

The
**plakar backup**
command creates a new snapshot of
*place*,
or the current directory.
Snapshots can be filtered to ignore specific files or directories
based on patterns provided through options.

*place*
can be either a path, an URI, or a label with the form
"@*name*"
to reference a source connector configured with
plakar-source(1).

The options are as follows:

**-concurrency** *number*

> Set the maximum number of parallel tasks for faster processing.
> Defaults to
> `8 * CPU count + 1`.

**-force-timestamp** *timestamp*

> Specify a fixed timestamp (in ISO 8601 or relative human format) to use
> for the snapshot.
> Could be used to reimport an existing backup with the same timestamp.

**-ignore** *pattern*

> Specify individual gitignore exclusion patterns to ignore files or
> directories in the backup.
> This option can be repeated.

**-ignore-file** *file*

> Specify a file containing gitignore exclusion patterns, one per line, to
> ignore files or directories in the backup.

**-check**

> Perform a full check on the backup after success.

**-no-xattr**

> Skip extended attributes (xattrs) when creating the backup.

**-o** *option*=*value*

> Can be used to pass extra arguments to the source connector.
> The given
> *option*
> takes precedence over the configuration file.

**-quiet**

> Suppress output to standard input, only logging errors and warnings.

**-packfiles** *path*

> Path where to put the temporary packfiles instead of building them in memory.
> If the special value
> 'memory'
> is specified then the packfiles are build in memory (the default value)

**-silent**

> Suppress all output.

**-tag** *tag*

> Comma-separated list of tags to apply to the snapshot.

**-scan**

> Do not write a snapshot; instead, perform a dry run by outputting the list of
> files and directories that would be included in the backup.
> Respects all exclude patterns and other options, but makes no changes to the
> Kloset store.

# EXAMPLES

Create a snapshot of the current directory with two tags:

	$ plakar backup -tag daily-backup,production

Ignore files using patterns in a given file:

	$ plakar backup -ignore-file ~/my-ignore-file /var/www

or by using patterns specified inline:

	$ plakar backup -ignore "*.tmp" -ignore "*.log" /var/www

Pass an option to the importer, in this case to don't traverse mount
points:

	$ plakar backup -o dont_traverse_fs=true /

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
