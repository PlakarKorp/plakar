PLAKAR-AGENT(1) - General Commands Manual

# NAME

**plakar-agent** - Run the Plakar agent

# SYNOPSIS

**plakar&nbsp;agent**
\[**start**&nbsp;**-teardown**&nbsp;*delay*]
\[**stop**]

# DESCRIPTION

The
**plakar agent start**
command starts the Plakar agent which will execute subsequent
plakar(1)
commands on their behalfs for faster processing.
**plakar agent**
continues is auto-spawned and terminates when idle for too long.

The options are as follows:

**-teardown** *delay*

> Specify the delay after which the idle agent terminate.
> The
> *delay*
> parameter must be given as a sequence of decimal value,
> each followed by a time unit
> (e.g. "1m30s").
> Delaults to 5 seconds.

**stop**

> Force the currently running agent to stop.
> This is useful when upgrading from an older
> plakar(1)
> version were the agent was always running.

# DIAGNOSTICS

The **plakar-agent** utility exits&#160;0 on success, and&#160;&gt;0 if an error occurs.

0

> Command completed successfully.

&gt;0

> An error occurred, such as invalid parameters, inability to create the
> repository, or configuration issues.

# SEE ALSO

plakar(1)

Plakar - July 3, 2025
