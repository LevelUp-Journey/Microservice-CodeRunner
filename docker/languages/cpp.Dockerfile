# C++ testing environment with doctest
FROM gcc:12-bullseye

LABEL maintainer="LevelUp Journey"
LABEL description="C++ execution environment with doctest framework"

# Install essential tools and libraries
RUN apt-get update && apt-get install -y \
    cmake \
    ninja-build \
    valgrind \
    gdb \
    curl \
    wget \
    && rm -rf /var/lib/apt/lists/*

# Install doctest (header-only library)
RUN mkdir -p /usr/local/include && \
    wget -O /usr/local/include/doctest.h \
    https://raw.githubusercontent.com/doctest/doctest/master/doctest/doctest.h

# Set working directory
WORKDIR /workspace

# Create directories for code execution
RUN mkdir -p /workspace/src /workspace/build /workspace/tests

# Set memory and time limits via ulimit
RUN echo 'ulimit -v 524288' >> /etc/bash.bashrc  # 512MB virtual memory
RUN echo 'ulimit -t 30' >> /etc/bash.bashrc      # 30 seconds CPU time

# Default command
CMD ["/bin/bash"]