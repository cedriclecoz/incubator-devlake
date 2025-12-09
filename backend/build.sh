#!/bin/bash

# Build script for DevLake with version information
# Usage: ./build.sh [tag_name]
# Example: ./build.sh github5

set -e

# Default tag if not provided
TAG=${1:-github-latest}

# Get git information
GIT_COMMIT=$(git rev-parse --short HEAD 2>/dev/null || echo "unknown")
GIT_BRANCH=$(git rev-parse --abbrev-ref HEAD 2>/dev/null || echo "unknown")
BUILD_DATE=$(date -Iseconds)

# Version string
VERSION="1.0.3-beta8-${TAG}-${BUILD_DATE}"

echo "================================================"
echo "Building DevLake Docker Image"
echo "================================================"
echo "Tag:          devlake:${TAG}"
echo "Version:      ${VERSION}"
echo "Git Commit:   ${GIT_COMMIT}"
echo "Git Branch:   ${GIT_BRANCH}"
echo "Build Date:   ${BUILD_DATE}"
echo "================================================"

# Build the Docker image
docker build \
  -t "devlake:${TAG}" \
  --build-arg VERSION="${VERSION}" \
  --build-arg TAG=devlake --build-arg SHA=${VERSION} \
  --label "git-commit=${GIT_COMMIT}" \
  --label "git-branch=${GIT_BRANCH}" \
  --label "build-date=${BUILD_DATE}" \
  --label "version=${VERSION}" \
  -f Dockerfile \
  .

echo ""
echo "================================================"
echo "Build Complete!"
echo "================================================"
echo "Image: devlake:${TAG}"
echo "Version: ${VERSION}"
echo ""
echo "To use this image, update ~/internal/docker-compose.yml:"
echo "  sed -i 's/devlake:github[0-9]*/devlake:${TAG}/g' ~/internal/docker-compose.yml"
echo ""
echo "Then restart docker-compose:"
echo "  cd ~/internal && docker compose restart"
echo "================================================"
