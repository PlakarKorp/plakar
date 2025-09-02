PLAKAR-SERVICES(1) - General Commands Manual

# NAME

**plakar-services** - Manage optional Plakar-connected services

# SYNOPSIS

**plakar&nbsp;services**
\[**status**&nbsp;|&nbsp;**enable**&nbsp;|&nbsp;**disable**]
*service\_name*

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

Services are managed by the backend and discovered at runtime.
For example, when the
"alerting"
service is enable, it will:

1.	Send email notifications when operations fail.

2.	Expose the latest alerting reports in the Plakar UI
	(see plakar-ui(1)).

By default, all services are disabled.

# SUBCOMMANDS

*status* *service\_name*

> Display the current configuration status (enabled or disabled) of the named
> service.

*enable* *service\_name*

> Enable the specified service.

*disable* *service\_name*

> Disable the specified service.

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

Plakar - August 7, 2025 - PLAKAR-SERVICES(1)
