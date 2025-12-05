PLAKAR-CLONE(1) - General Commands Manual

# NAME

**plakar-clone** - Clone a Plakar repository to a new location

# SYNOPSIS

**plakar&nbsp;clone**
**to**
*path*

# DESCRIPTION

The
**plakar clone**
command creates a full clone of an existing Plakar repository,
including all snapshots, packfiles, and repository states, and saves
it at the specified
*path*.

# EXAMPLES

Clone a repository to a new location:

	plakar clone to /path/to/new/repository

# DIAGNOSTICS

The **plakar-clone** utility exits&#160;0 on success, and&#160;&gt;0 if an error occurs.

0

> Command completed successfully.

&gt;0

> An error occurred, such as failure to access the source repository or
> to create the target repository.

# SEE ALSO

plakar(1),
plakar-create(1)

Plakar - July 3, 2025
