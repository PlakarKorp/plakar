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

**add** *name* *location* \[*option*=*value ...*]

> Create a new store entry identified by
> *name*
> with the specified
> *location*.
> Specific additional configuration parameters can be set by adding
> *option*=*value*
> parameters.

**check** *name*

> Check wether the store identified by
> *name*
> is properly configured.

**import**
\[**-config** *location*]
\[**-overwrite**]
\[**-rclone**]
\[*sections ...*]

> Import store configurations from various sources including files,
> piped input, or rclone configurations.

> By default, reads from stdin, allowing for piped input from other commands.

> The
> **-config**
> option specifies a file or URL to read the configuration from.

> The
> **-overwrite**
> option allows overwriting existing store configurations with
> the same names.

> The
> **-rclone**
> option treats the input as an rclone configuration, useful for
> importing rclone remotes as Plakar stores.

> Specific sections can be imported by listing their names.

> Sections can be renamed during import by appending
> **:**&zwnj;*newname*.

> For detailed examples and usage patterns, see the
> [https://docs.plakar.io/en/guides/importing-configurations/](https://docs.plakar.io/en/guides/importing-configurations/)
> Importing Configurations
> guide.

**ping** *name*

> Try to connect to the store identified by
> *name*
> to make sure it is reachable.

**rm** *name*

> Remove the store identified by
> *name*
> from the configuration.

**set** *name* \[*option*=*value ...*]

> Set the
> *option*
> to
> *value*
> for the store identified by
> *name*.
> Multiple option/value pairs can be specified.

**show** \[**-secrets**] \[*name ...*]

> Display the current stores configuration.
> If
> **-secrets**
> is specified, sensitive information such as passwords or tokens will be shown.

**unset** *name* \[*option ...*]

> Remove the
> *option*
> for the store entry identified by
> *name*.

# DIAGNOSTICS

The **plakar-store** utility exits&#160;0 on success, and&#160;&gt;0 if an error occurs.

# SEE ALSO

plakar(1)

Plakar - September 11, 2025
