PLAKAR-INFO(1) - General Commands Manual

# NAME

**plakar-info** - Display detailed information about internal structures

# SYNOPSIS

**plakar&nbsp;info**
\[**-errors**]
\[*snapshot*]

# DESCRIPTION

The
**plakar info**
command provides detailed information about a Plakar repository
and snapshots.
The type of information displayed depends on the specified argument.
Without any arguments, display information about the repository.

The options are as follows:

**-errors**

> Show errors within the specified snapshot.

# EXAMPLES

Show repository information:

	$ plakar info

Show detailed information for a snapshot:

	$ plakar info abc123

Show errors within a snapshot:

	$ plakar info -errors abc123

# DIAGNOSTICS

The **plakar-info** utility exits&#160;0 on success, and&#160;&gt;0 if an error occurs.

0

> Command completed successfully.

&gt;0

> An error occurred, such as an invalid snapshot or object ID, or a
> failure to retrieve the requested data.

# SEE ALSO

plakar(1),
plakar-backup(1)

Plakar - July 3, 2025 - PLAKAR-INFO(1)
