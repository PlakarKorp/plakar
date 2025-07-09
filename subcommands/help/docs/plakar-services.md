PLAKAR-SERVICES(1) - General Commands Manual

# NAME

**plakar-services** - Manage optional Plakar-connected services

# SYNOPSIS

**plakar&nbsp;services&nbsp;status&nbsp;*service\_name*&zwnj;**  
**plakar&nbsp;services&nbsp;enable&nbsp;*service\_name*&zwnj;**  
**plakar&nbsp;services&nbsp;disable&nbsp;*service\_name*&zwnj;**

# DESCRIPTION

The
**plakar services**
command allows you to enable, disable, and inspect additional services that
integrate with the
**plakar**
platform via
plakar-login(1)
authentication.
These services connect to the plakar.io infrastructure, and should only be
enabled if you agree to transmit non-sensitive operational data to plakar.io.

All subcommands require prior authentication via
plakar-login(1).

At present, only the
"alerting"
service is available.
When enabled, alerting will:

1.	Send email notifications when operations fail.

2.	Expose the latest alerting reports in the Plakar UI
	(see plakar-ui(1)).

By default, all services are disabled.

# SUBCOMMANDS

*status* *service\_name*

> Display the current configuration status (enabled or disabled) of the named
> service.
> Currently, only the "alerting" service is supported.

*enable* *service\_name*

> Enable the specified service.
> Currently, only the "alerting" service is supported.

*disable* *service\_name*

> Disable the specified service.
> Currently, only the "alerting" service is supported.

# EXAMPLES

Check the status of the alerting service:

	$ plakar services status alerting

Enable alerting:

	$ plakar services enable alerting

Disable alerting:

	$ plakar services disable alerting

# SEE ALSO

plakar-login(1),
plakar-ui(1)

Plakar - July 8, 2025
