PLAKAR-STORE(1) - General Commands Manual

# NAME

**plakar-store** - Manage Plakar store configurations

# SYNOPSIS

**plakar&nbsp;store**
*subcommand&nbsp;...*

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

**import** \[**-config** *location*] \[**-overwrite**] \[**-rclone**] \[*sections ...*]

> Import a configuration from either stdin (default),
> a file, or a URL.

> If
> *location*
> is specified, the input will be read from that file or URL.

> If
> **-overwrite**
> is specified, existing sections will be overwritten by new ones.

> If
> **-rclone**
> is specified, the input will be treated as an rclone configuration.

> If
> *sections*
> are specified, only those sections will be imported.
> A section can be renamed on import by appending a colon and the new name.

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

**show** \[**-secrets**] \[name ...]

> Display the current stores configuration.
> If
> **-secrets**
> is specified, sensitive information such as passwords or tokens will be shown.

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
