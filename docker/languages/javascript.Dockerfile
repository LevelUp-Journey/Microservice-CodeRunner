# Node.js testing environment with Jest
FROM node:18-bullseye-slim

LABEL maintainer="LevelUp Journey"
LABEL description="Node.js execution environment with Jest framework"

# Install system dependencies
RUN apt-get update && apt-get install -y \
    python3 \
    make \
    g++ \
    && rm -rf /var/lib/apt/lists/*

# Set working directory
WORKDIR /workspace

# Create package.json with testing dependencies
RUN echo '{\
  "name": "code-runner-test",\
  "version": "1.0.0",\
  "scripts": {\
    "test": "jest --verbose --json --outputFile=/workspace/test-results.json"\
  },\
  "dependencies": {\
    "jest": "^29.7.0",\
    "@jest/reporters": "^29.7.0"\
  },\
  "jest": {\
    "testEnvironment": "node",\
    "testTimeout": 30000,\
    "maxWorkers": 1,\
    "detectOpenHandles": true,\
    "forceExit": true\
  }\
}' > package.json

# Install dependencies
RUN npm install --production

# Create directories for code execution
RUN mkdir -p /workspace/src /workspace/tests

# Default command
CMD ["/bin/bash"]