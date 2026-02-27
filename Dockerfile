# Multi-stage Dockerfile for Plakar UI

# Build stage
FROM golang:1.23-alpine AS builder

# Install build dependencies
RUN apk add --no-cache git make

# Set working directory
WORKDIR /app

# Copy go mod files first for better caching
COPY ./go.mod ./go.sum ./

# Download dependencies
RUN go mod download

# Copy source code
COPY . .

# Build the application
RUN make plakar

# Final stage - minimal runtime image
FROM alpine:latest

# Install required runtime dependencies
RUN apk add --no-cache ca-certificates tzdata

# Create non-root user for security
RUN addgroup -g 1000 plakar && \
    adduser -D -u 1000 -G plakar plakar

# Set working directory
WORKDIR /app

# Copy the binary from builder stage
COPY --from=builder /app/plakar /usr/local/bin/plakar

# Create data directory and set permissions
RUN mkdir -p /data && chown plakar:plakar /data

# Switch to non-root user
USER plakar

# Expose the default port (will be random if not specified via -addr)
EXPOSE 8080

# Set environment variables
ENV PLAKAR_REPOSITORY=/data
ENV PLAKAR_PASSPHRASE=""

# Health check to ensure the service is running
HEALTHCHECK --interval=30s --timeout=10s --start-period=5s --retries=3 \
  CMD wget --no-verbose --tries=1 --spider http://localhost:8080/ || exit 1

# Default command to start the UI
# Users can override with custom arguments
CMD ["/usr/local/bin/plakar", "-no-agent", "ui", "-addr", "0.0.0.0:8080", "-no-spawn", "-no-auth"]
