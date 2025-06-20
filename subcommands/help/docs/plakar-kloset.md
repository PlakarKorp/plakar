PLAKAR-KLOSET(1) - General Commands Manual

# NAME

**plakar kloset** - Manage Plakar repository configurations

# SYNOPSIS

**plakar kloset**
**kloset**
\[subcommand&nbsp;...]

# DESCRIPTION

The
**plakar kloset**
command manages configuration of the Plakar repository configurations.

The subcommands are as follows:

**create** *name* *location* \[option=value ...]

> Create a new repository identified by
> *name*
> with the specified
> *location*.
> Specific additional configuration parameters might be set by adding
> *option=value*
> entries.
> Different repositories have different options available.
> \[key]

**set** *name* \[option=value ...]

> Set the
> *option*
> to
> *value*
> for the repository identified by
> *name*.
> Different repositories have different options available.
> Multiple option/value pairs might be specified.

**unset** *name* \[option ...]

> Remove the
> *option*
> for the repository identified by
> *name*.

**check** *name*

> Check wether the repository
> *name*
> is properly configured.

**default** *name*

> Make the repository
> *name*
> the default one.

# DIAGNOSTICS

The **plakar kloset** utility exits&#160;0 on success, and&#160;&gt;0 if an error occurs.

# SEE ALSO

plakar(1)

Plakar - February 27, 2025
