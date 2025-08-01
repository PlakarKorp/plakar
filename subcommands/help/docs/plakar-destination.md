PLAKAR-DESTINATION(1) - General Commands Manual

# NAME

**plakar-destination** - Manage Plakar restore destination configuration

# SYNOPSIS

**plakar&nbsp;destination**
\[subcommand&nbsp;...]

# DESCRIPTION

The
**plakar destination**
command manages the configuration of destinations where Plakar will restore.

The configuration consists in a set of named entries, each of them
describing a destination where a restore operation may happen.

A destination is defined by at least a location, specifying the exporter
to use, and some exporter-specific parameters.

The subcommands are as follows:

**add** *name* *location* \[option=value ...]

> Create a new destination entry identified by
> *name*
> with the specified
> *location*
> describing the exporter to use.
> Additional exporter options can be set by adding
> *option=value*
> parameters.

**check** *name*

> Check wether the exporter for the destination identified by
> *name*
> is properly configured.

**import** \[*names*]

> Ingest a destination configuration from stdin.
> If
> *names*
> are specified, only those sections will be imported.

**ls**

> Display the current destinations configuration.
> This is the default if no subcommand is specified.

**ping** *name*

> Try to open the destination identified by
> *name*
> to make sure it is reachable.

**rm** *name*

> Remove the destination identified by
> *name*
> from the configuration.

**set** *name* \[option=value ...]

> Set the
> *option*
> to
> *value*
> for the destination identified by
> *name*.
> Multiple option/value pairs can be specified.

**show** \[name ...]

> Display the current destinations configuration.
> This is the default if no subcommand is specified.

**unset** *name* \[option ...]

> Remove the
> *option*
> for the destination entry identified by
> *name*.

# DIAGNOSTICS

The **plakar-destination** utility exits&#160;0 on success, and&#160;&gt;0 if an error occurs.

# SEE ALSO

plakar(1)

Plakar - July 3, 2025
