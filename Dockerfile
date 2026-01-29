# Multi-stage build for Moon - Dynamic Headless Engine
# Stage 1: Builder
FROM golang:1.24-alpine AS builder

# Install CA certificates for go mod download
RUN apk add --no-cache ca-certificates

# Set working directory
WORKDIR /build

# Copy go mod files first for better caching
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Build the binary
# Using CGO_ENABLED=0 for a fully static binary
RUN CGO_ENABLED=0 GOOS=linux go build -a -ldflags="-w -s" -o moon ./cmd/moon

# Stage 2: Runtime - using scratch for minimal image
FROM scratch

# Copy CA certificates from builder (needed for HTTPS if any)
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/

# Copy binary from builder
COPY --from=builder /build/moon /usr/local/bin/moon

# Expose default port
EXPOSE 6006

# Run moon in foreground (daemon mode)
CMD ["/usr/local/bin/moon"]
