# Python testing environment with pytest
FROM python:3.11-slim-bullseye

LABEL maintainer="LevelUp Journey"
LABEL description="Python execution environment with pytest framework"

# Install system dependencies
RUN apt-get update && apt-get install -y \
    gcc \
    g++ \
    && rm -rf /var/lib/apt/lists/*

# Install Python testing and utility packages
RUN pip install --no-cache-dir \
    pytest==7.4.2 \
    pytest-timeout==2.1.0 \
    pytest-json-report==1.5.0 \
    memory-profiler==0.61.0 \
    psutil==5.9.5

# Set working directory
WORKDIR /workspace

# Create directories for code execution
RUN mkdir -p /workspace/src /workspace/tests

# Set Python path
ENV PYTHONPATH=/workspace/src:/workspace

# Resource limits via Python
ENV PYTHONMALLOC=malloc
ENV MALLOC_ARENA_MAX=1

# Default command
CMD ["/bin/bash"]