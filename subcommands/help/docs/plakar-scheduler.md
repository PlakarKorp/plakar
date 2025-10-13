PLAKAR-SCHEDULER(1) - General Commands Manual

# NAME

**plakar-scheduler** - Run the Plakar scheduler

# SYNOPSIS

**plakar&nbsp;scheduler**
\[**-foreground**]
\[**start**&nbsp;**-tasks**&nbsp;*configfile*]
\[**stop**]

# DESCRIPTION

The
**plakar scheduler**
runs in the background and manages task execution based on the defined schedule.

The options are as follows:

**-foreground**

> Run the scheduler in the foreground instead of as a background service.

**-tasks** *configfile*

> Specify the configuration file that contains the task definitions and schedules.

**start** **-tasks** *configfile*

> Starts the scheduler service and its tasks from
> *configfile*.

**stop**

> Stop the currently running scheduler service.

# DIAGNOSTICS

The **plakar-scheduler** utility exits&#160;0 on success, and&#160;&gt;0 if an error occurs.

0

> Command completed successfully.

&gt;0

> An error occurred, such as invalid parameters, inability to create the
> repository, or configuration issues.

# SEE ALSO

plakar(1)

Plakar - July 3, 2025 - PLAKAR-SCHEDULER(1)
