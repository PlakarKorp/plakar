PLAKAR-LOGIN(1) - General Commands Manual

# NAME

**plakar-login** - Authenticate to Plakar services

# SYNOPSIS

**plakar&nbsp;login**
\[**-email**&nbsp;*email*]
\[**-github**]
\[**-no-spawn**]
\[**-status**]

# DESCRIPTION

The
**plakar login**
command initiates an authentication flow with the Plakar platform.
Login is optional for most
**plakar**
commands but required to enable certain services, such as alerting.
See also
plakar-service(1).

Only one authentication method may be specified per invocation: the
**-email**
and
**-github**
options are mutually exclusive.
If neither is provided,
**-github**
is assumed.

The options are as follows:

**-email** *email*

> Send a login link to the specified email address.
> Clicking the link in the received email will authenticate
> **plakar**.

**-github**

> Use GitHub OAuth to authenticate.
> A browser will be spawned to initiate the OAuth flow unless
> **-no-spawn**
> is specified.

**-no-spawn**

> Do not automatically open a browser window for authentication flows.

**-status**

> Check wether the user is currently logged in.
> This option cannot be used with any other options.

# EXAMPLES

Start a login via email:

	$ plakar login -email user@example.com

Authenticate via GitHub (default, opens browser):

	$ plakar login

# SEE ALSO

plakar(1),
plakar-logout(1),
plakar-service(1)

Plakar - July 8, 2025 - PLAKAR-LOGIN(1)
