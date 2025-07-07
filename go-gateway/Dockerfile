# Multi-stage build for Bifrost Industrial Gateway
# Stage 1: Build stage
FROM golang:1.22-alpine AS builder

# Install build dependencies
RUN apk add --no-cache git ca-certificates tzdata

# Set working directory
WORKDIR /app

# Copy go mod files
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download

# Copy source code
COPY . .

# Build the binary with optimizations
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build \
    -ldflags='-w -s -extldflags "-static"' \
    -a -installsuffix cgo \
    -o bifrost-gateway \
    ./cmd/gateway

# Stage 2: Runtime stage
FROM scratch

# Import from builder
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY --from=builder /usr/share/zoneinfo /usr/share/zoneinfo

# Copy the binary
COPY --from=builder /app/bifrost-gateway /bifrost-gateway

# Copy configuration files
COPY --from=builder /app/gateway.yaml /gateway.yaml

# Create non-root user
USER 65534:65534

# Expose ports
EXPOSE 8080 9090 2112

# Health check
HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
    CMD ["/bifrost-gateway", "-health-check"]

# Run the binary
ENTRYPOINT ["/bifrost-gateway"]
CMD ["-config", "/gateway.yaml"]