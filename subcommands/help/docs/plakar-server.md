PLAKAR-SERVER(1) - General Commands Manual

# NAME

**plakar-server** - Start a Plakar server

# SYNOPSIS

**plakar&nbsp;server**
\[**-allow-delete**]
\[**-listen**&nbsp;\[*host*]:*port*]
\[**-cert**&nbsp;*path*]
\[**-key**&nbsp;*path*]
\[**-token**&nbsp;*string*]

# DESCRIPTION

The
**plakar server**
command starts a Plakar server instance at the provided
*address*,
allowing remote interaction with a Kloset store over a network.

The options are as follows:

**-allow-delete**

> Enable delete operations.
> By default, delete operations are disabled to prevent accidental data
> loss.

**-listen** \[*host*]:*port*

> The
> *host*
> and
> *port*
> where to listen to, separated by a colon.
> The host name is optional, and defaults to all available addresses.
> If
> **-listen**
> is not provided, the server defaults to listen on localhost at port 9876.

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

**-token** *string*

> Require clients to provide the given Authorization Bearer token in order
> to interact with
> **plakar server**.

# EXIT STATUS

The **plakar-server** utility exits&#160;0 on success, and&#160;&gt;0 if an error occurs.

# EXAMPLES

Start a plakar server on the local store:

	$ plakar server

Start a plakar server on a remote store:

	$ plakar at sftp://example.org server

Start a server on a specific address and port:

	$ plakar server -listen 127.0.0.1:12345

Start a https server on a specific address and port:

	$ plakar server -listen backup.example.com:12345 -cert fullchain.pem -key privkey.pem

# SEE ALSO

plakar(1)

# CAVEATS

When a host name is provided,
**plakar server**
uses only one of the IP addresses it resolves to,
preferably IPv4 .

Plakar - August 25, 2026 - PLAKAR-SERVER(1)
