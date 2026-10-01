PLAKAR-RESTORE(1) - General Commands Manual

# NAME

**plakar-restore** - Restore files from a Plakar snapshot

# SYNOPSIS

**plakar&nbsp;restore**
\[**-category**&nbsp;*category*]
\[**-environment**&nbsp;*environment*]
\[**-job**&nbsp;*job*]
\[**-name**&nbsp;*name*]
\[**-perimeter**&nbsp;*perimeter*]
\[**-post-hook**&nbsp;*command*]
\[**-pre-hook**&nbsp;*command*]
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

**-pre-hook** *command*

> Run
> *command*
> in a shell before starting the restore.
> If the command exits with a non-zero status, restore is aborted.

**-post-hook** *command*

> Run
> *command*
> in a shell after a successful restore.
> If the command exits with a non-zero status, a warning is logged and
> the restore still succeeds.

**-to** *directory*

> Specify the base directory to which the files will be restored.
> If omitted, files are restored to the current working directory.

**-o** *option*=*value*

> Can be used to pass extra arguments to the destination connector.
> The given
> *option*
> takes precedence over the configuration file.

# EXIT STATUS

The **plakar-restore** utility exits&#160;0 on success, and&#160;&gt;0 if an error occurs.

# EXAMPLES

Restore all files from a specific snapshot to the current directory:

	$ plakar restore abc123

Restore to a specific directory:

	$ plakar restore -to /mnt/ abc123

Restore specific path to a specific directory:

	$ plakar restore -to /mnt/ abc123:/etc/apache2

Restore to a specific destination:

	$ plakar restore -to @s3target abc123

Restore specific path to a specific destination :

	$ plakar restore -to  @s3target abc123:/etc/apache2

# SEE ALSO

plakar(1),
plakar-backup(1)

Plakar - May 5, 2026
