#!/bin/bash

# Build script for code runner Docker images
# Usage: ./build-images.sh [language]
# If no language specified, builds all images

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
LANGUAGES_DIR="$SCRIPT_DIR/languages"
IMAGE_PREFIX="levelup/code-runner"

# Color codes for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

log_info() {
    echo -e "${GREEN}[INFO]${NC} $1"
}

log_warn() {
    echo -e "${YELLOW}[WARN]${NC} $1"
}

log_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

build_image() {
    local language=$1
    local dockerfile="$LANGUAGES_DIR/$language.Dockerfile"
    local image_name="$IMAGE_PREFIX:$language"
    
    if [ ! -f "$dockerfile" ]; then
        log_error "Dockerfile not found: $dockerfile"
        return 1
    fi
    
    log_info "Building image for $language..."
    if docker build -f "$dockerfile" -t "$image_name" "$SCRIPT_DIR"; then
        log_info "Successfully built $image_name"
    else
        log_error "Failed to build $image_name"
        return 1
    fi
}

# Available languages
LANGUAGES=("cpp" "python" "javascript" "java" "go")

if [ $# -eq 0 ]; then
    # Build all images
    log_info "Building all language images..."
    for lang in "${LANGUAGES[@]}"; do
        build_image "$lang"
    done
else
    # Build specific language
    language=$1
    if [[ " ${LANGUAGES[@]} " =~ " ${language} " ]]; then
        build_image "$language"
    else
        log_error "Unknown language: $language"
        log_info "Available languages: ${LANGUAGES[*]}"
        exit 1
    fi
fi

log_info "Build process completed!"