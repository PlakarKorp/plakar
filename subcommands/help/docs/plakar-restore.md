PLAKAR-RESTORE(1) - General Commands Manual

# NAME

**plakar-restore** - Restore files from a Plakar snapshot

# SYNOPSIS

**plakar&nbsp;restore**
\[**-before**&nbsp;*date*]
\[**-category**&nbsp;*category*]
\[**-environment**&nbsp;*environment*]
\[**-job**&nbsp;*job*]
\[**-latest**]
\[**-name**&nbsp;*name*]
\[**-perimeter**&nbsp;*perimeter*]
\[**-quiet**]
\[**-since**&nbsp;*date*]
\[**-skip-permissions**]
\[**-tag**&nbsp;*tag*]
\[**-to**&nbsp;*directory*]
\[**-o**&nbsp;*option*=*value*]
\[*snapshotID*:*path&nbsp;...*]

# DESCRIPTION

The
**plakar restore**
command is used to restore files and directories at
*path*
from a specified Plakar snapshot to the local file system.
If
*path*
is omitted, then all the files in the specified
*snapshotID*
are restored.
If no
*snapshotID*
is provided, the command attempts to restore the current working
directory from the last matching snapshot.

The options are as follows:

**-name** *string*

> Only apply command to snapshots that match
> *name*.

**-category** *string*

> Only apply command to snapshots that match
> *category*.

**-environment** *string*

> Only apply command to snapshots that match
> *environment*.

**-perimeter** *string*

> Only apply command to snapshots that match
> *perimeter*.

**-job** *string*

> Only apply command to snapshots that match
> *job*.

**-tag** *string*

> Only apply command to snapshots that match
> *tag*.

**-skip-permissions**

> Skip restoring file permissions and ownership during restore,
> defaulting to 0750 for directories and 0640 for files.
> It Fl to Ar directory
> Specify the base directory to which the files will be restored.
> If omitted, files are restored to the current working directory.

**-o** *option*=*value*

> Can be used to pass extra arguments to the destination connector.
> The given
> *option*
> takes precedence over the configuration file.

**-quiet**

> Suppress output to standard input, only logging errors and warnings.

# EXAMPLES

Restore all files from a specific snapshot to the current directory:

	$ plakar restore abc123

Restore to a specific directory:

	$ plakar restore -to /mnt/ abc123

Restore latest snapshot to a specific directory:

	$ plakar restore -latest -to /mnt/ abc123

Restore specific path to a specific directory:

	$ plakar restore -to /mnt/ abc123:/etc/apache2

Restore to a specific destination:

	$ plakar restore -to @s3target abc123

Restore specific path to a specific destination :

	$ plakar restore -to  @s3target abc123:/etc/apache2

# DIAGNOSTICS

The **plakar-restore** utility exits&#160;0 on success, and&#160;&gt;0 if an error occurs.

0

> Command completed successfully.

&gt;0

> An error occurred, such as a failure to locate the snapshot or a
> destination directory issue.

# SEE ALSO

plakar(1),
plakar-backup(1)

Plakar - July 3, 2025
