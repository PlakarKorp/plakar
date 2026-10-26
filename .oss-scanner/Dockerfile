# Scanner image for https://github.com/PlakarKorp/plakar (main), built by Anthropic's OSS Scanner
# (https://github.com/anthropics/oss-scanner, enrolled as projects/plakar, whose project.yaml points here).
#
# The audit runs with no network, so this build fetches every Go module, builds the plakar binary and the
# tests of every package, runs the test suite once so the log shows the image works, and leaves two small
# repositories with one snapshot each under /srv for the scanner to use. threat_model.md, next to this
# file, says how to use them.
#
# Docker reads Dockerfile.dockerignore next to this file instead of the repository's .dockerignore, which
# is written for the release image and leaves out testing/ and unittests/, which `go test ./...` needs.

FROM golang:1.26-bookworm
ENV DEBIAN_FRONTEND=noninteractive
# curl and jq to talk to `plakar server` and the ui API by hand; fuse3 for `plakar mount` when /dev/fuse exists.
RUN apt-get update && apt-get install -y --no-install-recommends curl jq fuse3 \
    && rm -rf /var/lib/apt/lists/*

# scanner contract: the checkout lives inside the image, at /src
COPY . /src
WORKDIR /src

# Every module the binary and the tests depend on, including the kloset engine and the fs, http, ptar,
# stdio and tar integrations plakar links in. They stay readable in the module cache for the audit.
RUN go mod download

# The binary, as `make` builds it, then every package and its tests, with and without the race detector.
# A step that fails fails the build, and primary_contact gets the log.
RUN go build -v -o /usr/local/bin/plakar . \
    && go build ./... \
    && go test -run '^$' ./... \
    && go test -race -run '^$' ./...

# The test suite, as CI runs it (-p 4: several packages each spin up caches and do real encryption work).
# A failing test must not abort the build.
RUN (timeout 1500 go test -p 4 -timeout 2m ./... || echo "WARNING: go test reported failures")

# Two repositories with one snapshot of /src/docs each, so every plakar command has something to work on:
# an encrypted one, which is the default, and a plaintext one, whose packfiles and states can be edited
# with a hex editor. The passphrase and the consent to open plaintext repositories stay in the environment
# so `plakar at /srv/encrypted ...` and `plakar at /srv/plaintext ...` just work. The periodic check of
# the releases feed is turned off: there is no network.
ENV PLAKAR_PASSPHRASE=oss-scanner-throwaway-passphrase-for-the-demo-repository \
    PLAKAR_INSECURE_PLAINTEXT=1 \
    DISABLE_SECURITY_CHECK=1
RUN plakar at /srv/encrypted create \
    && plakar at /srv/encrypted backup /src/docs \
    && plakar at /srv/encrypted ls \
    && plakar at /srv/plaintext create -plaintext \
    && plakar at /srv/plaintext backup /src/docs \
    && plakar at /srv/plaintext ls

# Everything is fetched. From here on the go command must not try to reach a proxy or download a toolchain.
ENV GOPROXY=off \
    GOTOOLCHAIN=local
