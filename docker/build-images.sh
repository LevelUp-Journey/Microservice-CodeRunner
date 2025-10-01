#!/bin/bash

# Script para construir la imagen Docker de C++

echo "🔨 Building C++ Docker image..."

cd "$(dirname "$0")"

docker build -t coderunner-cpp:latest -f cpp/Dockerfile cpp/

if [ $? -eq 0 ]; then
    echo "✅ C++ Docker image built successfully!"
    echo ""
    echo "📋 Image details:"
    docker images coderunner-cpp:latest
else
    echo "❌ Failed to build C++ Docker image"
    exit 1
fi
