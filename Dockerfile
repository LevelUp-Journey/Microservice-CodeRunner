# Multi-stage Dockerfile for gRPC CodeRunner Microservice with Docker-in-Docker support
# Stage 1: Build stage
FROM golang:1.25.1-alpine AS builder

# Install build dependencies
RUN apk add --no-cache \
  git \
  make \
  gcc \
  musl-dev \
  ca-certificates

WORKDIR /build

# Copy go mod files
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download

# Copy source code
COPY . .

# Build the application
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags="-w -s" -o /app/coderunner ./cmd/server

# Stage 2: Runtime stage with Docker-in-Docker support
FROM docker:27-dind

# Install runtime dependencies
RUN apk add --no-cache \
  ca-certificates \
  bash \
  curl \
  tzdata \
  su-exec

# Create non-root user for application (Docker daemon runs as root)
RUN addgroup -g 1000 appuser && \
  adduser -D -u 1000 -G appuser appuser && \
  adduser appuser docker

# Set working directory
WORKDIR /app

# Copy binary from builder
COPY --from=builder /app/coderunner /app/coderunner

# Copy Docker context files
COPY docker /app/docker

# Copy migrations directory
COPY migrations /app/migrations

# Create necessary directories with proper permissions
RUN mkdir -p /app/logs /app/temp && \
  chown -R appuser:appuser /app

# Set environment variables
ENV DOCKER_HOST=unix:///var/run/docker.sock \
  DOCKER_TLS_CERTDIR="" \
  PATH="/app:${PATH}"

# Health check
HEALTHCHECK --interval=30s --timeout=10s --start-period=40s --retries=3 \
  CMD curl -f http://localhost:8084/health || exit 1

# Expose gRPC and HTTP ports
EXPOSE 9084 8084

# Create entrypoint script
RUN echo '#!/bin/sh' > /app/entrypoint.sh && \
  echo 'set -e' >> /app/entrypoint.sh && \
  echo '' >> /app/entrypoint.sh && \
  echo '# Start Docker daemon in background if not already running' >> /app/entrypoint.sh && \
  echo 'if ! docker info >/dev/null 2>&1; then' >> /app/entrypoint.sh && \
  echo '    echo "🐳 Starting Docker daemon..."' >> /app/entrypoint.sh && \
  echo '    dockerd-entrypoint.sh &' >> /app/entrypoint.sh && \
  echo '    DOCKER_PID=$!' >> /app/entrypoint.sh && \
  echo '    ' >> /app/entrypoint.sh && \
  echo '    # Wait for Docker daemon to be ready' >> /app/entrypoint.sh && \
  echo '    echo "⏳ Waiting for Docker daemon to be ready..."' >> /app/entrypoint.sh && \
  echo '    for i in $(seq 1 30); do' >> /app/entrypoint.sh && \
  echo '        if docker info >/dev/null 2>&1; then' >> /app/entrypoint.sh && \
  echo '            echo "✅ Docker daemon is ready"' >> /app/entrypoint.sh && \
  echo '            break' >> /app/entrypoint.sh && \
  echo '        fi' >> /app/entrypoint.sh && \
  echo '        echo "Waiting for Docker daemon... ($i/30)"' >> /app/entrypoint.sh && \
  echo '        sleep 1' >> /app/entrypoint.sh && \
  echo '    done' >> /app/entrypoint.sh && \
  echo '    ' >> /app/entrypoint.sh && \
  echo '    if ! docker info >/dev/null 2>&1; then' >> /app/entrypoint.sh && \
  echo '        echo "❌ Docker daemon failed to start"' >> /app/entrypoint.sh && \
  echo '        exit 1' >> /app/entrypoint.sh && \
  echo '    fi' >> /app/entrypoint.sh && \
  echo 'fi' >> /app/entrypoint.sh && \
  echo '' >> /app/entrypoint.sh && \
  echo '# Pre-pull required Docker images' >> /app/entrypoint.sh && \
  echo 'echo "📦 Pre-pulling Docker images..."' >> /app/entrypoint.sh && \
  echo 'docker pull node:20-alpine || true' >> /app/entrypoint.sh && \
  echo 'docker pull python:3.12-alpine || true' >> /app/entrypoint.sh && \
  echo 'docker pull openjdk:21-slim || true' >> /app/entrypoint.sh && \
  echo 'docker pull gcc:13.2 || true' >> /app/entrypoint.sh && \
  echo 'docker pull mcr.microsoft.com/dotnet/sdk:8.0 || true' >> /app/entrypoint.sh && \
  echo 'docker pull golang:1.23-alpine || true' >> /app/entrypoint.sh && \
  echo 'docker pull rust:1.75-alpine || true' >> /app/entrypoint.sh && \
  echo '' >> /app/entrypoint.sh && \
  echo '# Build custom C++ image if Dockerfile exists' >> /app/entrypoint.sh && \
  echo 'if [ -f /app/docker/cpp/Dockerfile ]; then' >> /app/entrypoint.sh && \
  echo '    echo "🔨 Building C++ Docker image..."' >> /app/entrypoint.sh && \
  echo '    docker build -t coderunner-cpp:latest -f /app/docker/cpp/Dockerfile /app/docker/cpp/ || true' >> /app/entrypoint.sh && \
  echo 'fi' >> /app/entrypoint.sh && \
  echo '' >> /app/entrypoint.sh && \
  echo 'echo "🚀 Starting CodeRunner microservice..."' >> /app/entrypoint.sh && \
  echo '' >> /app/entrypoint.sh && \
  echo '# Execute the application' >> /app/entrypoint.sh && \
  echo 'exec "$@"' >> /app/entrypoint.sh && \
  chmod +x /app/entrypoint.sh

ENTRYPOINT ["/app/entrypoint.sh"]
CMD ["/app/coderunner"]
