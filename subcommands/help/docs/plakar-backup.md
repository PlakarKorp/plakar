PLAKAR-BACKUP(1) - General Commands Manual

# NAME

**plakar-backup** - Create a new snapshot in a Plakar repository

# SYNOPSIS

**plakar&nbsp;backup**
\[**-concurrency**&nbsp;*number*]
\[**-exclude**&nbsp;*pattern*]
\[**-exclude-file**&nbsp;*file*]
\[**-check**]
\[**-o**&nbsp;*option*]
\[**-quiet**]
\[**-tag**&nbsp;*tag*]
\[**-scan**]
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
can be either a path, an URI, or a label on the form
"@*name*"
to reference a source configured with
plakar-source(1).

The options are as follows:

**-concurrency** *number*

> Set the maximum number of parallel tasks for faster processing.
> Defaults to
> `8 * CPU count + 1`.

**-exclude** *pattern*

> Specify individual glob exclusion patterns to ignore files or
> directories in the backup.
> This option can be repeated.

**-exclude-file** *file*

> Specify a file containing glob exclusion patterns, one per line, to
> ignore files or directories in the backup.

**-check**

> Perform a full check on the backup after success.

**-o** *option*

> Can be used to pass extra arguments to the importer.
> The given
> *option*
> takes precence over the configuration file.

**-quiet**

> Suppress output to standard input, only logging errors and warnings.

**-tag** *tag*

> Specify a tag to assign to the snapshot for easier identification.

**-scan**

> Don't actually create a snapshot, just output the list of files.

# EXAMPLES

Create a snapshot of the current directory with a tag:

	$ plakar backup -tag daily-backup

Backup a specific directory with exclusion patterns from a file:

	$ plakar backup -exclude-file ~/my-excludes-file /var/www

Backup a directory with specific file exclusions:

	$ plakar backup -exclude "*.tmp" -exclude "*.log" /var/www

# DIAGNOSTICS

The **plakar-backup** utility exits&#160;0 on success, and&#160;&gt;0 if an error occurs.

0

> Command completed successfully, snapshot created.

&gt;0

> An error occurred, such as failure to access the repository or issues
> with exclusion patterns.

# SEE ALSO

plakar(1),
plakar-source(1)

Plakar - July 3, 2025
