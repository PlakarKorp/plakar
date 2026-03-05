# Stage 1: Build
FROM golang:1.24-alpine AS builder

WORKDIR /src

# Cache dependency downloads
COPY go.mod go.sum ./
RUN go mod download

# Copy full source (ui/v2/frontend/ and subcommands/help/docs/ are embedded via go:embed)
COPY . .

# Build static binary matching goreleaser configuration
RUN CGO_ENABLED=0 go build -trimpath -v -o /plakar .

# Stage 2: Runtime
FROM alpine:3.21

# CA certificates for HTTPS connections to remote repositories and plakar.io
RUN apk add --no-cache ca-certificates

# Create non-root user (/etc/passwd required by user.Current())
RUN addgroup -S plakar && adduser -S -G plakar -h /home/plakar plakar

COPY --from=builder /plakar /usr/local/bin/plakar

USER plakar
WORKDIR /home/plakar

ENTRYPOINT ["plakar"]
