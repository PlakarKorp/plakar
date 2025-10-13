PLAKAR-SERVICE(1) - General Commands Manual

# NAME

**plakar-service** - Manage optional Plakar-connected services

# SYNOPSIS

**plakar&nbsp;service&nbsp;**list**&zwnj;**  
**plakar&nbsp;service&nbsp;**add**&nbsp;*name*&nbsp;\[*key*=*value&nbsp;...*]**  
**plakar&nbsp;service&nbsp;**rm**&nbsp;*name*&zwnj;**  
**plakar&nbsp;service&nbsp;**status**&nbsp;*name*&zwnj;**  
**plakar&nbsp;service&nbsp;**show**&nbsp;*name*&zwnj;**  
**plakar&nbsp;service&nbsp;**enable**&nbsp;*name*&zwnj;**  
**plakar&nbsp;service&nbsp;**disable**&nbsp;*name*&zwnj;**  
**plakar&nbsp;service&nbsp;**set**&nbsp;*name*&nbsp;\[*key*=*value&nbsp;...*]**  
**plakar&nbsp;service&nbsp;**unset**&nbsp;*name*&nbsp;\[*key&nbsp;...*]**

# DESCRIPTION

The
**plakar service**
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

**list**

> Display the list of available services.

**add** *name* \[*key*=*value ...*]

> Set the configuration for the service identified by
> *name*
> and enable it.
> The configuration is defined by the given set of
> *key*/*value*
> pairs.
> The existing configuration, if any, is discarded.

**rm** *name*

> Disable the service identified by
> *name*
> and discard its configuration.

**status** *name*

> Display the current status (enabled or disabled) of the named
> service.

**show** *name*

> Display the configuration for the specified service.

**enable** *name*

> Enable the specified service.

**disable** *name*

> Disable the specified service.

**set** *name* \[*key*=*value ...*]

> Set the configuration
> *key*
> to
> *value*
> for the service identified by
> *name*.
> Multiple
> *key*/*value*
> pairs can be specified.

**unset** *name* \[*key ...*]

> Unset the configuration
> *key*
> for the service identified by
> *name*.
> Multiple keys can be specified.

# EXAMPLES

Check the status of the alerting service:

	$ plakar service status alerting

Enable alerting:

	$ plakar service enable alerting

Disable alerting:

	$ plakar service disable alerting

# SEE ALSO

plakar-login(1),
plakar-ui(1)

Plakar - August 7, 2025 - PLAKAR-SERVICE(1)
