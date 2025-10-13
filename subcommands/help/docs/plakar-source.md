PLAKAR-SOURCE(1) - General Commands Manual

# NAME

**plakar-source** - Manage Plakar backup source configuration

# SYNOPSIS

**plakar&nbsp;source**
*subcommand&nbsp;...*

# DESCRIPTION

The
**plakar source**
command manages the configuration of data sources for Plakar to backup.

The configuration consists in a set of named entries, each of them
describing a source for a backup operation.

A source is defined by at least a location, specifying the importer
to use, and some importer-specific parameters.

The subcommands are as follows:

**add** *name* *location* \[*option*=*value ...*]

> Create a new source entry identified by
> *name*
> with the specified
> *location*
> describing the importer to use.
> Additional importer options can be set by adding
> *option=value*
> parameters.

**check** *name*

> Check wether the importer for the source identified by
> *name*
> is properly configured.

**import**
\[**-config** *location*]
\[**-overwrite**]
\[**-rclone**]
\[*sections ...*]

> Import source configurations from various sources including files,
> piped input, or rclone configurations.

> By default, reads from stdin, allowing for piped input from other commands.

> The
> **-config**
> option specifies a file or URL to read the configuration from.

> The
> **-overwrite**
> option allows overwriting existing source configurations with
> the same names.

> The
> **-rclone**
> option treats the input as an rclone configuration, useful for
> importing rclone remotes as Plakar sources.

> Specific sections can be imported by listing their names.

> Sections can be renamed during import by appending
> **:**&zwnj;*newname*.

> For detailed examples and usage patterns, see the
> [https://docs.plakar.io/en/guides/importing-configurations/](https://docs.plakar.io/en/guides/importing-configurations/)
> Importing Configurations
> guide.

**ping** *name*

> Try to open the data source identified by
> *name*
> to make sure it is reachable.

**rm** *name*

> Remove the source identified by
> *name*
> from the configuration.

**set** *name* \[*option*=*value ...*]

> Set the
> *option*
> to
> *value*
> for the source identified by
> *name*.
> Multiple option/value pairs can be specified.

**show** \[**-secrets**] \[*name ...*]

> Display the current sources configuration.
> If
> **-secrets**
> is specified, sensitive information such as passwords or tokens will be shown.

**unset** *name* \[*option ...*]

> Remove the
> *option*
> for the source entry identified by
> *name*.

# EXIT STATUS

The **plakar-source** utility exits&#160;0 on success, and&#160;&gt;0 if an error occurs.

# SEE ALSO

plakar(1)

Plakar - September 11, 2025 - PLAKAR-SOURCE(1)
