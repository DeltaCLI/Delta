#!/bin/bash
# Docker-based release builder for Delta CLI
# This runs the build in a container to avoid timeouts

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
BLUE='\033[0;34m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Function to print colored output
log_info() { echo -e "${BLUE}[INFO]${NC} $1"; }
log_success() { echo -e "${GREEN}[SUCCESS]${NC} $1"; }
log_error() { echo -e "${RED}[ERROR]${NC} $1"; }
log_warning() { echo -e "${YELLOW}[WARNING]${NC} $1"; }

# Check if version is provided
VERSION="${1:-v0.4.7-alpha}"
if [[ ! "$VERSION" =~ ^v[0-9]+\.[0-9]+\.[0-9]+ ]]; then
    log_error "Invalid version format. Expected format: vX.Y.Z[-suffix]"
    exit 1
fi

log_info "Building Delta CLI release ${VERSION} using Docker..."

# Build the Docker image
log_info "Building Docker release image..."
docker build -f Dockerfile.release \
    --build-arg VERSION="${VERSION}" \
    -t delta-release-builder:${VERSION} \
    .

# Create a container and extract the artifacts
log_info "Creating container to extract artifacts..."
CONTAINER_ID=$(docker create delta-release-builder:${VERSION})

# Create release directory
RELEASE_DIR="releases/${VERSION}_docker_$(date +%Y%m%d_%H%M%S)"
mkdir -p "${RELEASE_DIR}"

# Extract artifacts from container
log_info "Extracting release artifacts..."
docker cp ${CONTAINER_ID}:/release/. "${RELEASE_DIR}/"

# Clean up container
docker rm ${CONTAINER_ID}

# List the artifacts
log_success "Release artifacts created in ${RELEASE_DIR}:"
ls -la "${RELEASE_DIR}/"

# Optional: Create GitHub release
if [ "$2" == "--upload" ] && command -v gh &> /dev/null; then
    log_info "Creating GitHub release..."
    
    # Get release notes
    RELEASE_NOTES_FILE="RELEASE_NOTES/RELEASE_NOTES_${VERSION}.md"
    if [ -f "$RELEASE_NOTES_FILE" ]; then
        RELEASE_NOTES=$(cat "$RELEASE_NOTES_FILE")
    else
        RELEASE_NOTES="Release ${VERSION}"
    fi
    
    # Create release
    gh release create "${VERSION}" \
        --title "Delta CLI ${VERSION}" \
        --notes "${RELEASE_NOTES}" \
        "${RELEASE_DIR}"/*.tar.gz \
        "${RELEASE_DIR}"/*.zip \
        "${RELEASE_DIR}"/checksums.sha256
    
    log_success "GitHub release created successfully!"
else
    log_info "To upload to GitHub, run: $0 ${VERSION} --upload"
fi

log_success "Docker release build completed!"