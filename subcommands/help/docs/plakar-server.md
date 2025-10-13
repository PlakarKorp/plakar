PLAKAR-SERVER(1) - General Commands Manual

# NAME

**plakar-server** - Start a Plakar server

# SYNOPSIS

**plakar&nbsp;server**
\[**-allow-delete**]
\[**-listen**&nbsp;\[*host*]:*port*]

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

# EXAMPLES

Start a plakar server on the local store:

	$ plakar server

Start a plakar server on a remote store:

	$ plakar at sftp://example.org server

Start a server on a specific address and port:

	$ plakar server -listen 127.0.0.1:12345

# DIAGNOSTICS

The **plakar-server** utility exits&#160;0 on success, and&#160;&gt;0 if an error occurs.

# SEE ALSO

plakar(1)

# CAVEATS

When a host name is provided,
**plakar server**
uses only one of the IP addresses it resolves to,
preferably IPv4 .

Plakar - July 3, 2025 - PLAKAR-SERVER(1)
