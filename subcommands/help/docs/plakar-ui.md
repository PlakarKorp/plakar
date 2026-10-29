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

The UI serves a chat assistant when the environment names a model
provider.
The assistant answers questions about the repository by driving the
model through the read-only tools of
plakar-mcp(1).
With a hosted provider, snapshot metadata and file contents the
assistant reads are sent to that provider, but only for a message
someone sent it; with a local server nothing leaves the machine.
Anyone who can reach the UI can use the assistant, and so the
credentials behind it.

`ANTHROPIC_API_KEY`, `OPENAI_API_KEY`, `GEMINI_API_KEY`

> The key of a hosted provider.
> Setting one enables the assistant with that provider, the first of
> these when several are set.

`PLAKAR_ASSISTANT_PROVIDER`

> The provider to use:
> **anthropic**,
> **openai**,
> **gemini**,
> **lmstudio**,
> **ollama**
> or
> **openai-compatible**
> for any other server speaking the chat completions API.
> The local servers and
> **openai-compatible**
> have to be named here, as they need no key.

`PLAKAR_ASSISTANT_MODEL`

> The model to talk to.
> Required with every provider but
> **anthropic**,
> which has a default.

`PLAKAR_ASSISTANT_URL`

> The base URL of the provider's API, in place of its usual one.
> Required with
> **openai-compatible**.

`PLAKAR_ASSISTANT_API_KEY`

> The key to send, in place of the provider's own variable above.

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
