# Build stage
FROM golang:1.21-alpine AS builder

# Install build dependencies
RUN apk add --no-cache git make

# Set working directory
WORKDIR /build

# Copy go mod files
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download

# Copy source code
COPY . .

# Build all binaries
RUN make build

# Runtime stage
FROM alpine:latest

# Install runtime dependencies
RUN apk add --no-cache \
    ca-certificates \
    git \
    curl \
    wget \
    xz \
    bzip2 \
    cpio \
    gcc \
    g++ \
    make \
    linux-headers \
    perl \
    python3 \
    bash

# Create non-root user
RUN addgroup -g 1000 ldf && \
    adduser -D -u 1000 -G ldf ldf

# Create necessary directories
RUN mkdir -p /opt/ldf/bin /data /workspace && \
    chown -R ldf:ldf /opt/ldf /data /workspace

# Copy binaries from builder
COPY --from=builder /build/bin/* /opt/ldf/bin/

# Copy default configuration
COPY --from=builder /build/configs/default.yaml /opt/ldf/config.yaml

# Set environment variables
ENV PATH="/opt/ldf/bin:${PATH}"
ENV LDF_CONFIG="/opt/ldf/config.yaml"

# Switch to non-root user
USER ldf

# Set working directory
WORKDIR /workspace

# Expose API port
EXPOSE 8080

# Default command (can be overridden)
CMD ["ldf", "--help"]
