PLAKAR-DESTINATION(1) - General Commands Manual

# NAME

**plakar-destination** - Manage Plakar restore destination configuration

# SYNOPSIS

**plakar&nbsp;destination**
*subcommand&nbsp;...*

# DESCRIPTION

The
**plakar destination**
command manages the configuration of destinations where Plakar will restore.

The configuration consists in a set of named entries, each of them
describing a destination where a restore operation may happen.

A destination is defined by at least a location, specifying the exporter
to use, and some exporter-specific parameters.

The subcommands are as follows:

**add** *name* *location* \[*option*=*value ...*]

> Create a new destination entry identified by
> *name*
> with the specified
> *location*
> describing the exporter to use.
> Additional exporter options can be set by adding
> *option*=*value*
> parameters.

**check** *name*

> Check wether the exporter for the destination identified by
> *name*
> is properly configured.

**import**
\[**-config** *location*]
\[**-overwrite**]
\[**-rclone**]
\[*sections ...*]

> Import destination configurations from various sources including files,
> piped input, or rclone configurations.

> By default, reads from stdin, allowing for piped input from other commands
> like
> **plakar source show**.

> The
> **-config**
> option specifies a file or URL to read the configuration from.

> The
> **-overwrite**
> option allows overwriting existing destination configurations with
> the same names.

> The
> **-rclone**
> option treats the input as an rclone configuration, useful for
> importing rclone remotes as Plakar destinations.

> Specific sections can be imported by listing their names.

> Sections can be renamed during import by appending
> **:**&zwnj;*newname*.

> For detailed examples and usage patterns, see the
> [https://plakar.io/docs/v1.1.0/guides/importing-configurations/](https://plakar.io/docs/v1.1.0/guides/importing-configurations/)
> Importing Configurations
> guide.

**ping** *name*

> Try to open the destination identified by
> *name*
> to make sure it is reachable.

**rm** *name*

> Remove the destination identified by
> *name*
> from the configuration.

**set** *name* \[*option*=*value ...*]

> Set the
> *option*
> to
> *value*
> for the destination identified by
> *name*.
> Multiple option/value pairs can be specified.

**show** \[**-secrets**] \[*name ...*]

> Display the current destinations configuration.
> If
> **-secrets**
> is specified, sensitive information such as passwords or tokens will be shown.

**unset** *name* \[*option ...*]

> Remove the
> *option*
> for the destination entry identified by
> *name*.

# EXIT STATUS

The **plakar-destination** utility exits&#160;0 on success, and&#160;&gt;0 if an error occurs.

# SEE ALSO

plakar(1)

Plakar - September 11, 2025
