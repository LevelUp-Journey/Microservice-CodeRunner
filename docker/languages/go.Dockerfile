# Go testing environment with native testing framework
FROM golang:1.21-bullseye

LABEL maintainer="LevelUp Journey"
LABEL description="Go execution environment with native testing framework"

# Install system dependencies
RUN apt-get update && apt-get install -y \
    git \
    && rm -rf /var/lib/apt/lists/*

# Set working directory
WORKDIR /workspace

# Initialize Go module
RUN go mod init code-runner-test

# Create directories for code execution
RUN mkdir -p /workspace/src /workspace/tests

# Set Go environment variables
ENV GO111MODULE=on
ENV GOPROXY=direct
ENV GOSUMDB=off

# Set resource limits
ENV GOGC=100
ENV GOMEMLIMIT=512MiB

# Default command
CMD ["/bin/bash"]