PLAKAR-UI(1) - General Commands Manual

# NAME

**plakar-ui** - Serve the Plakar web user interface

# SYNOPSIS

**plakar&nbsp;ui**
\[**-addr**&nbsp;*address*]
\[**-cors**]
\[**-no-auth**]
\[**-no-refresh**]
\[**-no-spawn**]
\[**-cert**&nbsp;*path*]
\[**-key**&nbsp;*path*]

# DESCRIPTION

The
**plakar ui**
command serves the Plakar web user interface.
By default, it opens the default web browser.

The options are as follows:

**-addr** *address*

> Specify the address and port for the UI to listen on separated by a colon,
> (e.g. localhost:8080).
> If omitted,
> **plakar ui**
> listens on localhost on a random port.

**-cors**

> Set the
> 'Access-Control-Allow-Origin'
> HTTP headers to allow the UI to be accessed from any origin.

**-no-auth**

> Disable the authentication token that otherwise is needed to consume
> the exposed HTTP APIs.

**-no-refresh**

> Do not refresh the local state from the store on API calls.
> Useful when you want to share a common cache between multiple UIs.

**-no-spawn**

> Do not automatically open the web browser.

**-cert** *path*

> Path to a full certificate file in PEM format.
> If both
> **-cert**
> and
> **-key**
> are provided, the server will expect https connections.
> If one or both are missing, the server will fall back to http.

**-key** *path*

> Path to a certificate private key file in PEM format.

# ENVIRONMENT

`PLAKAR_UI_TOKEN`

> Authentication token to use instead of generating a random one at
> each start.
> Ignored when
> **-no-auth**
> is given.

# EXIT STATUS

The **plakar-ui** utility exits&#160;0 on success, and&#160;&gt;0 if an error occurs.

# EXAMPLES

Using a custom address and disable automatic browser execution:

	$ plakar ui -addr localhost:9090 -no-spawn

Create a https server with a custom certificate:

	$ plakar ui -cert fullchain.pem -key privkey.pem

# SEE ALSO

plakar(1)

Plakar - May 5, 2026 - PLAKAR-UI(1)
