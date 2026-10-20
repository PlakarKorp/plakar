# Threat model: plakar

## What this project does and where untrusted input enters

plakar is a backup tool. The `plakar` command reads data from a source through an importer (the local
filesystem by default, or an integration such as S3, a database or a SaaS), chunks, deduplicates, compresses
and encrypts it, and writes it as a snapshot into a repository called a kloset, held by a storage connector
(a local directory by default, or a bucket, a `plakar server`, a `.ptar` archive). Later it lists, browses,
checks, diffs, mounts and restores those snapshots. The storage engine is the kloset library
(https://github.com/PlakarKorp/kloset), enrolled in the scanner on its own; this repository holds the command
line, the web ui and its JSON API, the `plakar server` store server, the local cache daemon, the plugin
system, and the client side of the plakar.io services.

The design assumes the repository is stored somewhere the user does not trust: a public cloud, a shared
host, a server someone else runs. Confidentiality and integrity come from the user's passphrase, never from
the storage. Treat all of the following as attacker-controlled:

- **Everything read back from a repository**: the repository configuration (`CONFIG`), states, packfiles,
  snapshot headers, the virtual filesystem indexes (B-trees), chunk data, locks. Whoever controls the storage
  controls every byte. The parsers live mostly in kloset; the paths that reach them from here are the
  subcommands (`ls`, `cat`, `check`, `restore`, `diff`, `sync`, `maintenance`, `rm`, `info`, `locate`,
  `mount`), the ui API, and `plakar server`.
- **The data being backed up**: names, symlink targets, extended attributes, sizes, directory shapes.
  A hostile source tree (symlink loops, links pointing outside the root, names with `..` or control
  characters, millions of entries in one directory) must not make a backup write outside the repository,
  record something that later restores somewhere else, or crash the run.
- **The contents of a snapshot at restore time**, which is the previous two combined: `plakar restore -to DIR`
  writes files, directories, symlinks, modes and xattrs named by the snapshot into DIR. This is the
  archive-extraction problem: nothing in a snapshot may make a restore write outside DIR, follow a symlink
  it just created out of DIR, or change the mode or ownership of something it did not create.
- **Clients of `plakar server`**: `plakar at REPO server -listen HOST:PORT` serves a store over HTTP
  (`GET /`, `GET /resources/{resource}`, `GET|PUT|DELETE /resources/{resource}/{mac}`, see
  `server/httpd/httpd.go`). The resource name, the MAC, the `Range` header and the body are client
  controlled. `-token T` requires `Authorization: Bearer T`; without it any client may read and write.
  Deletes are refused unless `-allow-delete` is given. `-cert`/`-key` enable TLS, otherwise it is plain HTTP.
- **Servers that `plakar` talks to**: a repository reached over `http://` or `https://` (the http storage
  integration) is a `plakar server` that may lie; the plugin registry and the plakar.io API behind
  `plakar login`, `plakar service` and `plakar pkg add NAME` (default `PLAKAR_API_URL`, over TLS) return
  JSON, packages and tokens that are parsed here.
- **Browsers and processes that reach the ui**: `plakar at REPO ui` listens on localhost on a random port
  (`-addr` to choose), opens a browser, and requires a bearer token it generates (`PLAKAR_UI_TOKEN` to set
  it, `-no-auth` to drop it). Readers and downloads use URLs the API signs (`api/api_snapshot.go`). The
  API returns, and the ui renders, names, metadata and file contents taken from the snapshot, so a hostile
  snapshot is a hostile web page. The built ui bundle (`ui/web/build`) is committed only on release tags;
  on main the binary serves a placeholder page, and the frontend sources are under `ui/web/src`.
- **Plugin packages**: `plakar pkg add FILE` installs an integration from a `.ptar` package, verifying a
  signify signature against the keys trusted in `signify/` unless `-allow-unsigned`. The manifest and the
  files it extracts under the data directory are untrusted until the signature has been checked. Installed
  integrations run as separate processes that plakar spawns and talks to over gRPC
  (`plugins/`, `github.com/PlakarKorp/integration-grpc`), some as containers (`plugins/docker.go`).
- **Other local users**: the cache daemon (`plakar cached`) listens on a unix socket in the cache directory
  (`cached.sock`) with a lock file next to it; caches, cookies with service tokens, and the configuration
  live under the user's cache, config and data directories.

The command line, configuration files, environment variables and the `-pre-hook` and `-post-hook` commands
are the user's own and are trusted.

## Components that matter most / least

Most important, in this order:

1. Restore and export: `subcommands/restore`, the export path in kloset (`snapshot/export.go`,
   `snapshot/vfs`) and the fs exporter in the `github.com/PlakarKorp/integrations/fs` module, all readable
   in the module cache.
2. What is reachable from a hostile repository: `subcommands/*`, `api/`, `server/httpd`, and the kloset
   code they call.
3. `server/httpd` and the http storage integration (`github.com/PlakarKorp/integrations/http`) as a
   network-facing pair, including the token check and `-allow-delete`.
4. `api/` and `ui/web`: authentication, the URL signer, CORS (`-cors`), what the API echoes from a
   snapshot, and the frontend under `ui/web/src`.
5. `signify/`, `plugins/`, `pkg.go` and `subcommands/pkg`: signature verification, package extraction,
   how plugin processes are started.
6. `login/`, `services/`, `cookies/`, `reporting/`: what is sent to plakar.io, what is accepted back, where
   tokens are stored.
7. `cached/` and `appcontext/`: the local daemon, its socket and lock, file permissions.

Out of scope: `docs/`, man pages, `.github/`, `testing/` and `unittests/`, test files, and `ui/web/src`
tooling (not the application code). Third-party dependencies have their own processes; report a flaw in
one when plakar's use of it makes it reachable from untrusted input, and show the path. A flaw whose root
cause is in kloset belongs to kloset's enrolment; it is still useful to hear which plakar command reaches
it. The integrations in `github.com/PlakarKorp/integrations` are their own repository and are not enrolled,
but a path from a plakar command through the fs, http, ptar, tar or stdio integration that ships with the
binary is in scope here.

## How to exercise it

The image has only loopback networking. Every module is in the Go module cache (`go env GOMODCACHE`),
`GOPROXY=off` and `GOTOOLCHAIN=local` are set, the binary is `/usr/local/bin/plakar`, and the tests of
every package are built. `plakar help` and `plakar help COMMAND` show the man pages.

- **Repositories**: `/srv/encrypted` (the default configuration; `PLAKAR_PASSPHRASE` is set in the
  environment, so `plakar at /srv/encrypted ls` just works) and `/srv/plaintext` (created with `-plaintext`,
  neither encrypted nor compressed, so packfiles and states can be edited directly; plakar refuses to
  open such a repository unless `PLAKAR_INSECURE_PLAINTEXT` is set, and the image sets it). Each holds one
  snapshot of `/src/docs`. On disk a repository is `CONFIG`, `packfiles/`, `states/` and `locks/`, written by
  the fs storage integration. Make new ones with `plakar at DIR create`, with `-plaintext` for another
  plaintext one.
- **Commands**: `backup PATH`, `ls`, `cat SNAP:PATH`, `check [SNAP]`, `restore -to DIR SNAP`, `diff`,
  `info`, `locate`, `rm`, `maintenance`, `sync`, `archive`, `ptar`. The kloset formats are documented in
  the module cache (`packfile/README.md`, `repository/state`, `snapshot/header`, `docs/ptar.md`).
- **Tampering**: edit bytes under `/srv/plaintext/packfiles` or `/srv/plaintext/states`, then run `check`,
  `ls`, `cat` and `restore` and see what is detected and what happens when it is not. For the encrypted
  repository the same, with MACs failing first.
- **The store server**: `plakar at /srv/encrypted server -listen 127.0.0.1:9876` (add `-token T`,
  `-allow-delete`, `-cert`/`-key`), then `plakar at http://127.0.0.1:9876 ls` as a client, or `curl` the
  resource endpoints directly.
- **The ui and its API**: `plakar at /srv/encrypted ui -addr 127.0.0.1:9877 -no-spawn` prints the URL
  with the token; `curl -H "Authorization: Bearer $TOKEN" http://127.0.0.1:9877/api/repository/info`.
  The routes are listed in `api/api.go`. `-no-auth` drops the token, `-cors` opens CORS.
- **The cache daemon**: `plakar cached` in the background, then any command; the socket is
  `$XDG_CACHE_HOME/plakar/cached.sock` or `~/.cache/plakar/cached.sock`.
- **Plugins**: `plakar pkg create`, `plakar pkg build` and `plakar pkg add FILE` with `signify/testdata`
  for keys; `plakar pkg add NAME` needs the network and does not work in this image.
- **Mount**: `plakar at /srv/encrypted mount DIR` needs `/dev/fuse`, which the container may not have.
- **Tests**: `go test -p 4 ./...`. `testing/` has repository fixtures and mocks for writing new ones; the
  `api` and `server/httpd` tests show how to drive the HTTP surfaces in-process.

## How you rate severity

A finding counts when input from one of the sources above makes plakar do something the user did not ask
for. Say which source the attacker controls (storage, source tree, network peer, local user) and which
options the reproducer needs; a finding against the defaults weighs more than one that needs `-no-auth`,
`-allow-unsigned`, `-allow-delete`, `-cors`, `-plaintext` or a server without `-token`.

- **Critical**: a restore or a plugin install that writes, chmods or deletes outside its target directory.
  Restored data that differs from what was backed up without `check` or `restore` reporting an error
  (silent corruption from a tampered repository). Recovery of content, metadata, or key material from a
  repository without the passphrase, or the passphrase or a service token written to the repository, a log,
  a report or a remote service. Code execution from repository contents, from a source tree, from a package
  that passed signature verification, or from a `plakar server` client. A client deleting or overwriting
  store data past `-token` or without `-allow-delete`.
- **High**: reading backup data past the `plakar server` or ui token. A hostile repository, server, source
  tree or package that makes `backup`, `check`, `restore` or `sync` panic, hang or exhaust memory: a backup
  tool that dies on malformed input has lost the user's run, and a restore that cannot finish is the failure
  we exist to prevent. Cross-site scripting or request forgery in the ui that reaches the API with the
  user's token. Signature verification that accepts a package it should refuse. A backup that silently
  skips or mis-records data because of how the source tree is shaped.
- **Medium**: the same classes when they need a non-default option named above, or control of the local
  machine as another user (socket, lock file, cache and config permissions). A panic confined to one
  command on malformed command-line input or a malformed configuration file. Disclosure of file names or
  metadata that the ui or a report should not reveal.
- **Low**: a crash in a diagnostic command (`diag`, `info`, `locate`) on a hostile repository that `check`
  and `restore` handle correctly. Resource use that is merely proportional to a large input.

Bugs that are not vulnerabilities: output differences between `-json` and the text format, performance,
a documented behaviour you would have designed differently, deviations from tar or ptar conventions with
no security consequence.

## How to report

- A reproducer that runs in this image: the commands, and for tampering the exact bytes changed (a short
  Python or Go script that patches a file under `/srv/plaintext` is ideal), or the `curl` against
  `plakar server` or the ui. Say what was observed and what should have happened.
- A minimal patch against `main` with a regression test next to the existing ones. Go 1.26, `gofmt`,
  errors wrapped with `%w`, no `panic` outside `main` and `init`, every goroutine with an owner. Keep
  refactoring out of it.
- One root cause once, listing each command or endpoint it is reachable from. If the root cause is in
  kloset or in an integration module, say so and name the file in the module cache.

## Anything to leave alone

- The repository configuration (algorithms, KDF parameters, salt) and the shape of the store (number and
  size of packfiles and states, timestamps) are visible to whoever holds the storage, by design. Only the
  content and the metadata inside the encrypted objects are secret.
- `plakar server` without `-token`, without TLS, or with `-allow-delete`, and `plakar ui` with `-no-auth`
  or `-cors`, are documented modes for trusted networks. A finding there must show what an attacker gains
  beyond what that configuration already grants.
- The ui listens on localhost. A process on the same machine that already has the token is the user.
- Deleting snapshots with `rm`, `prune` or `maintenance` is the user's decision; a command that removes
  more than what was selected is a finding, the existence of the command is not.
- Weak passphrases are the user's choice; `create` already refuses the weakest.
- `plakar login` and the plakar.io services are opt-in and send operational data; the service side is
  not in scope, the client side (what is sent, what is trusted in the reply) is.
- Do not report the absence of rate limiting, HSTS or similar hardening on the localhost ui.
