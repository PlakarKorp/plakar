PLAKAR-STORE(1) - General Commands Manual

# NAME

**plakar-store** - Manage Plakar store configurations

# SYNOPSIS

**plakar&nbsp;store**
\[subcommand&nbsp;...]

# DESCRIPTION

The
**plakar store**
command manages the Plakar store configurations.

The configuration consists in a set of named entries, each of them
describing a Plakar store holding backups.

A store is defined by at least a location, specifying the storage
implementation to use, and some storage-specific parameters.

The subcommands are as follows:

**add** *name* *location* \[option=value ...]

> Create a new store entry identified by
> *name*
> with the specified
> *location*.
> Specific additional configuration parameters can be set by adding
> *option=value*
> parameters.

**check** *name*

> Check wether the store identified by
> *name*
> is properly configured.

**import**

> Ingest a store configuration from stdin.

**ping** *name*

> Try to connect to the store identified by
> *name*
> to make sure it is reachable.

**rm** *name*

> Remove the store identified by
> *name*
> from the configuration.

**set** *name* \[option=value ...]

> Set the
> *option*
> to
> *value*
> for the store identified by
> *name*.
> Multiple option/value pairs can be specified.

**show** \[name ...]

> Display the current stores configuration.
> This is the default if no subcommand is specified.

**unset** *name* \[option ...]

> Remove the
> *option*
> for the store entry identified by
> *name*.

# DIAGNOSTICS

The **plakar-store** utility exits&#160;0 on success, and&#160;&gt;0 if an error occurs.

# SEE ALSO

plakar(1)

Plakar - July 3, 2025
