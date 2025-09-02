PLAKAR-AGENT(1) - General Commands Manual

# NAME

**plakar-agent** - Run the Plakar agent

# SYNOPSIS

**plakar&nbsp;agent**

# DESCRIPTION

The
**plakar agent**
command starts the Plakar agent which will execute subsequent
plakar(1)
commands on their behalfs for faster processing.
**plakar agent**
continues is auto-spawned and terminates when idle for too long.

# DIAGNOSTICS

The **plakar-agent** utility exits&#160;0 on success, and&#160;&gt;0 if an error occurs.

0

> Command completed successfully.

&gt;0

> An error occurred, such as invalid parameters, inability to create the
> repository, or configuration issues.

# SEE ALSO

plakar(1)

Plakar - July 3, 2025 - PLAKAR-AGENT(1)
