PLAKAR-POLICY(1) - General Commands Manual

# NAME

**plakar-policy** - Manage Plakar retention policies

# SYNOPSIS

**plakar&nbsp;policy**
*subcommand&nbsp;...*

# DESCRIPTION

The
**plakar policy**
command manages the retention policies for
plakar-prune(1).

The configuration consists in a set of named entries, each of them
describing a retention policy.

The subcommands are as follows:

**add** *name* \[*option*=*value ...*]

> Create a new source entry identified by
> *name*.
> Additional parameters can be set by adding
> *option*=*value*
> parameters.

**rm** *name*

> Remove the policy identified by
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

**show**
\[**-ini**]
\[**-json**]
\[**-yaml**]
\[*name ...*]

> Display the current sources configuration.
> **-ini**,
> **-json**
> and
> **-yaml**
> control the output format, which is YAML by default.

**unset** *name* \[option ...]

> Remove the
> *option*
> for the policy identified by
> *name*.

The available options as described in
plakar-query(7):
each option corresponds the similarly named flag.

# EXIT STATUS

The **plakar-policy** utility exits&#160;0 on success, and&#160;&gt;0 if an error occurs.

# EXAMPLES

Create a policy
'weekly'
that keeps one backup per week and discards backups older than three
months:

	$ plakar policy add weekly
	$ plakar policy set weekly since='3 months'
	$ plakar policy set weekly per-week=1

Prune snapshots accordingly to the
'weekly'
policy:

	$ plakar prune -policy weekly

# SEE ALSO

plakar(1),
plakar-prune(1)

Plakar - September 11, 2025 - PLAKAR-POLICY(1)
