# Build stage
FROM golang:1.24-alpine AS builder

# Install build dependencies
RUN apk add --no-cache git gcc musl-dev curl bash

# Install go-task
RUN sh -c "$(curl --location https://taskfile.dev/install.sh)" -- -d -b /usr/local/bin

# Install bun (required for WebUI build)
RUN curl -fsSL https://bun.sh/install | bash
ENV PATH="/root/.bun/bin:${PATH}"

# Set working directory
WORKDIR /build

# Copy go mod files first for layer caching
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Build all binaries (server, CLI, WebUI)
RUN task build

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
RUN mkdir -p /opt/ldf/bin /opt/ldf/config /data /workspace && \
    chown -R ldf:ldf /opt/ldf /data /workspace

# Copy binaries from builder
COPY --from=builder /build/build/bin/ldfd /opt/ldf/bin/
COPY --from=builder /build/build/bin/ldfctl /opt/ldf/bin/

# Copy WebUI dist from builder
COPY --from=builder /build/src/webui/dist /opt/ldf/webui/

# Copy default configuration
COPY --from=builder /build/docs/samples/ldfd.yml /opt/ldf/config/ldfd.yml

# Set environment variables
ENV PATH="/opt/ldf/bin:${PATH}"
ENV LDFD_CONFIG="/opt/ldf/config/ldfd.yml"

# Switch to non-root user
USER ldf

# Set working directory
WORKDIR /workspace

# Expose API port
EXPOSE 8443

# Default command
CMD ["ldfd"]
