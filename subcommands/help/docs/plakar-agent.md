PLAKAR-AGENT(1) - General Commands Manual

# NAME

**plakar-agent** - Run the Plakar agent

# SYNOPSIS

**plakar&nbsp;agent**
\[**start**
\[**-foreground**]
\[**-log**&nbsp;*logfile*]
\[**-teardown**&nbsp;*delay*]]  
**plakar&nbsp;agent**
**stop**

# DESCRIPTION

The
**plakar agent start**
command, which is the default, starts the Plakar agent which will
execute subsequent
plakar(1)
commands on their behalfs for faster processing.

**plakar agent**
is executed automatically by most
plakar(1)
commands and terminates by itself when idle for too long, so usually
there's no need to manually start it.

The options for
**plakar agent**
**start**
are as follows:

**-foreground**

> Do not daemonize, run in the foreground and log to standard error.

**-log** *logfile*

> Write log output to the given
> *logfile*
> which is created if it does not exist.
> The default is to log to syslog.

**-teardown** *delay*

> Specify the delay after which the idle agent terminate.
> The
> *delay*
> parameter must be given as a sequence of decimal value,
> each followed by a time unit
> (e.g. "1m30s").
> Defaults to 5 seconds.

**plakar agent**
**stop**
forces the currently running agent to stop.
This is useful when upgrading from an older
plakar(1)
version were the agent was always running.

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
