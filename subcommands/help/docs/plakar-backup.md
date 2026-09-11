PLAKAR-BACKUP(1) - General Commands Manual

# NAME

**plakar-backup** - Create a new snapshot in a Kloset store

# SYNOPSIS

**plakar&nbsp;backup**
\[**-cache**&nbsp;*path*]
\[**-category**&nbsp;*category*]
\[**-check**]
\[**-dry-run**]
\[**-environment**&nbsp;*environment*]
\[**-force-timestamp**&nbsp;*timestamp*]
\[**-ignore**&nbsp;*pattern*]
\[**-ignore-file**&nbsp;*file*]
\[**-job**&nbsp;*job*]
\[**-name**&nbsp;*name*]
\[**-no-progress**]
\[**-no-xattr**]
\[**-o**&nbsp;*option*=*value*]
\[**-packfiles**&nbsp;*path*]
\[**-perimeter**&nbsp;*perimeter*]
\[**-tag**&nbsp;*tag*]
\[*place&nbsp;...*]

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

The alias can also be in the form of
"@*name*\[:path-override]"
to override the alias path on the command line.
If
*path-override*
starts with
'/'
the whole path is replaced with the override, otherwise it is
appended to the existing path.

Multiple
*places*
can be given, as long as they all refer to different paths
on the same remote, e.g. different files or different prefixes
on the same bucket.
Not all importer connectors support this feature, refer to their
documentation for more information.

The options are as follows:

**-cache** *path*

> Specify a path to store the vfs cache.
> Use the special value
> 'no'
> to disable caching.
> Use the special value
> 'vfs'
> to use the in-memory vfs cache (the default).

**-category** *category*

> Set the snapshot category.

**-check**

> Perform a full check on the backup after success.

**-dry-run**

> Do not write a snapshot; instead, perform a dry run by outputting the list of
> files and directories that would be included in the backup.
> Respects all exclude patterns and other options, but makes no changes to the
> Kloset store.

**-environment** *environment*

> Set the snapshot environment.

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
> This option can be repeated.

**-job** *job*

> Name the snapshot job.

**-name** *name*

> Name the snapshot.

**-no-progress**

> Do not compute or display progress.
> By default,
> **plakar backup**
> does two passes on the source of the backup: one to compute the
> number of items, and a second for processing the items themselves.
> This flag disables the pass to compute the number of items.
> It is set implicitly for some importer connectors that don't support
> the two-passes.

**-no-xattr**

> Skip extended attributes (xattrs) when creating the backup.

**-o** *option*=*value*

> Can be used to pass extra arguments to the source connector.
> The given
> *option*
> takes precedence over the configuration file.

**-packfiles** *path*

> Path where to put the temporary packfiles instead of building them in
> the default temporary directory.
> If the special value
> 'memory'
> is specified then the packfiles are built in memory.

**-perimeter** *perimeter*

> Set the snapshot perimeter.

**-tag** *tag*

> Comma-separated list of tags to apply to the snapshot.

# ENVIRONMENT

`PLAKAR_TAGS`

> Comma-separated list of tags to apply to the snapshot during backup.
> Overridden by the
> **-tag**
> command-line flag.

# EXIT STATUS

The **plakar-backup** utility exits&#160;0 on success, and&#160;&gt;0 if an error occurs.

# EXAMPLES

Create a snapshot of the current directory with two tags:

	$ plakar backup -tag daily-backup,production

Define an alias for an s3 bucket and backup multiple path prefixes

	$ plakar source add bucket s3://example.com \
	        access_key=... secret_access_key=...
	$ plakar backup @bucket:/assets @bucket:/uploads @bucket:/logs

Ignore files using patterns in one or more files or from the command line:

	$ plakar backup -ignore-file ~/.plkignore -ignore "*.tmp" /var/www

Pass an option to the importer, in this case to don't traverse mount
points:

	$ plakar backup -o dont_traverse_fs=true /

# SEE ALSO

plakar(1),
plakar-source(1)

Plakar - July 23, 2026 - PLAKAR-BACKUP(1)
