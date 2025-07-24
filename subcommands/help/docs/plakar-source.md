PLAKAR-SOURCE(1) - General Commands Manual

# NAME

**plakar-source** - Manage Plakar backup source configuration

# SYNOPSIS

**plakar&nbsp;source**
\[subcommand&nbsp;...]

# DESCRIPTION

The
**plakar source**
command manages the configuration of data sources for Plakar to backup.

The configuration consists in a set of named entries, each of them
describing a source for a backup operation.

A source is defined by at least a location, specifying the importer
to use, and some importer-specific parameters.

The subcommands are as follows:

**add** *name* *location* \[option=value ...]

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

> Ingest a source configuration from stdin.

**ls**

> Display the current sources configuration.
> This is the default if no subcommand is specified.

**ping** *name*

> Try to open the data source identified by
> *name*
> to make sure it is reachable.

**rm** *name*

> Remove the source identified by
> *name*
> from the configuration.

**set** *name* \[option=value ...]

> Set the
> *option*
> to
> *value*
> for the source identified by
> *name*.
> Multiple option/value pairs can be specified.

**show** \[name ...]

> Display the current sources configuration.
> This is the default if no subcommand is specified.

**unset** *name* \[option ...]

> Remove the
> *option*
> for the source entry identified by
> *name*.

# DIAGNOSTICS

The **plakar-source** utility exits&#160;0 on success, and&#160;&gt;0 if an error occurs.

# SEE ALSO

plakar(1)

Plakar - July 3, 2025
