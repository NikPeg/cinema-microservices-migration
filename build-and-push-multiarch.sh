#!/bin/bash

# Script to build and push multi-architecture Docker images
# This script builds images for both amd64 and arm64 architectures

set -e

# Configuration
REGISTRY="ghcr.io"
NAMESPACE="nikpeg/cinema-microservices-migration"
TAG="latest"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

echo -e "${GREEN}Starting multi-architecture Docker build and push...${NC}"

# Check if Docker buildx is available
if ! docker buildx version > /dev/null 2>&1; then
    echo -e "${RED}Docker buildx is not available. Please install Docker Desktop or Docker CE with buildx plugin.${NC}"
    exit 1
fi

# Create and use a new buildx builder instance
echo -e "${YELLOW}Setting up Docker buildx builder...${NC}"
docker buildx create --name multiarch-builder --use 2>/dev/null || docker buildx use multiarch-builder

# Ensure the builder is bootstrapped
docker buildx inspect --bootstrap

# Login to GitHub Container Registry
echo -e "${YELLOW}Please login to GitHub Container Registry...${NC}"
echo "You need a GitHub Personal Access Token with 'write:packages' permission"
echo "Username: your-github-username"
echo "Password: your-github-personal-access-token"
docker login ${REGISTRY}

# Build and push each service
services=(
    "monolith:src/monolith"
    "movies-service:src/microservices/movies"
    "events-service:src/microservices/events"
    "proxy-service:src/microservices/proxy"
)

for service_path in "${services[@]}"; do
    IFS=':' read -r service path <<< "$service_path"

    echo -e "${GREEN}Building ${service}...${NC}"

    IMAGE="${REGISTRY}/${NAMESPACE}/${service}:${TAG}"

    docker buildx build \
        --platform linux/amd64,linux/arm64 \
        --tag ${IMAGE} \
        --push \
        ${path}

    if [ $? -eq 0 ]; then
        echo -e "${GREEN}✓ Successfully built and pushed ${service}${NC}"
    else
        echo -e "${RED}✗ Failed to build ${service}${NC}"
        exit 1
    fi
done

echo -e "${GREEN}All images have been successfully built and pushed!${NC}"
echo -e "${YELLOW}Images are available at:${NC}"
for service_path in "${services[@]}"; do
    IFS=':' read -r service path <<< "$service_path"
    echo "  - ${REGISTRY}/${NAMESPACE}/${service}:${TAG}"
done

# Clean up the builder (optional)
# docker buildx rm multiarch-builder

echo -e "${GREEN}Done!${NC}"
