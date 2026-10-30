# Limitations:
#   This container runs plakar without a shared host cache.
#   Each invocation starts with a cold cache (no VFS cache, no cached
#   daemon persistence).  For improved incremental performance, mount a
#   persistent volume at the plakar cache directory, e.g.:
#       docker run -v plakar-cache:/home/plakar/.cache/plakar ...
#
# Stage 1: Shared runtime base
FROM alpine:3.23@sha256:ff7f0c727dd796aa36787bfcd9a64bf35a6ce72a73451b3359fde849b6703d63 AS runtime-base

# CA certificates for HTTPS connections to remote repositories and plakar.io
RUN apk add --no-cache ca-certificates

# Create non-root user (/etc/passwd required by user.Current())
RUN addgroup -S plakar && adduser -S -G plakar -h /home/plakar plakar

USER plakar
WORKDIR /home/plakar

ENTRYPOINT ["plakar"]

# Stage 2: GoReleaser runtime image (binary provided via GoReleaser build context)
FROM runtime-base AS goreleaser
ARG TARGETPLATFORM
COPY ${TARGETPLATFORM}/plakar /usr/local/bin/plakar

# Stage 3: Build from source for manual Docker builds
FROM golang:1.27-alpine@sha256:d200d2dd76b03a99b52651fd1325aff7dcd35422c35c01bac56571b0507f3deb AS builder

WORKDIR /src

# Cache dependency downloads
COPY go.mod go.sum ./
RUN go mod download

# Copy full source (ui/web/build/ and subcommands/help/docs/ are embedded via go:embed)
COPY . .

# Build static binary matching goreleaser configuration
RUN CGO_ENABLED=0 go build -trimpath -v -o /plakar .

# Stage 4: Runtime image for manual Docker builds
FROM runtime-base

COPY --from=builder /plakar /usr/local/bin/plakar
