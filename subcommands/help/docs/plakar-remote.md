PLAKAR-REMOTE(1) - General Commands Manual

# NAME

**plakar remote** - Manage Plakar remote repository configurations

# SYNOPSIS

**plakar remote**
**remote**
\[subcommand&nbsp;...]

# DESCRIPTION

The
**plakar remote**
command manages configuration of the remote Plakar repository configurations.

The subcommands are as follows:

**create** *name* *location* \[option=value ...]

> Create a new remote identified by
> *name*
> with the specified
> *location*.
> Specific additional configuration parameters might be set by adding
> *option=value*
> entries.
> Different remotes have different options available.
> \[key]

**set** *name* \[option=value ...]

> Set the
> *option*
> to
> *value*
> for the remote identified by
> *name*.
> Different remotes have different options available.
> Multiple option/value pairs might be specified.

**unset** *name* \[option ...]

> Remove the
> *option*
> for the remote identified by
> *name*.

**check** *name*

> Check wether the remote
> *name*
> is properly configured.

**ping** *name*

> Attempt to contact the remote
> *name*
> to ensure it is reachable.

# DIAGNOSTICS

The **plakar remote** utility exits&#160;0 on success, and&#160;&gt;0 if an error occurs.

# SEE ALSO

plakar(1)

Plakar - February 27, 2025
