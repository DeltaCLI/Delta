#!/bin/bash
# Quick release script that builds locally and creates GitHub release
# This avoids long Docker builds and timeouts

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

log_info "Quick release for Delta CLI ${VERSION}"

# Check if gh CLI is available
if ! command -v gh &> /dev/null; then
    log_error "GitHub CLI (gh) is required but not installed"
    log_info "Install with: brew install gh (macOS) or see https://cli.github.com/"
    exit 1
fi

# Check if logged in to GitHub
if ! gh auth status &> /dev/null; then
    log_error "Not logged in to GitHub. Run: gh auth login"
    exit 1
fi

# Create release directory
RELEASE_DIR="releases/${VERSION}_$(date +%Y%m%d_%H%M%S)"
mkdir -p "${RELEASE_DIR}"

log_info "Building current platform binary..."
make build VERSION="${VERSION}"

# Copy the binary
if [[ -f "build/linux/amd64/delta" ]]; then
    cp "build/linux/amd64/delta" "${RELEASE_DIR}/delta-linux-amd64"
elif [[ -f "build/darwin/amd64/delta" ]]; then
    cp "build/darwin/amd64/delta" "${RELEASE_DIR}/delta-darwin-amd64"
elif [[ -f "build/darwin/arm64/delta" ]]; then
    cp "build/darwin/arm64/delta" "${RELEASE_DIR}/delta-darwin-arm64"
elif [[ -f "build/windows/amd64/delta.exe" ]]; then
    cp "build/windows/amd64/delta.exe" "${RELEASE_DIR}/delta-windows-amd64.exe"
else
    log_error "No binary found in build directory"
    exit 1
fi

# Create a simple archive for current platform
cd "${RELEASE_DIR}"
if [[ "$OSTYPE" == "msys" || "$OSTYPE" == "win32" ]]; then
    zip "delta-${VERSION}-windows-amd64.zip" delta-windows-amd64.exe
else
    tar -czf "delta-${VERSION}-${OSTYPE}-${HOSTTYPE}.tar.gz" delta-*
fi
cd - > /dev/null

# Generate checksums
cd "${RELEASE_DIR}"
sha256sum * > checksums.sha256 || shasum -a 256 * > checksums.sha256
cd - > /dev/null

log_success "Build completed in ${RELEASE_DIR}"

# Get release notes
RELEASE_NOTES_FILE="RELEASE_NOTES/RELEASE_NOTES_${VERSION}.md"
if [[ -f "$RELEASE_NOTES_FILE" ]]; then
    RELEASE_NOTES=$(cat "$RELEASE_NOTES_FILE")
else
    RELEASE_NOTES="Release ${VERSION}"
fi

# Create GitHub release
log_info "Creating GitHub release..."

# Check if release already exists
if gh release view "${VERSION}" &> /dev/null; then
    log_warning "Release ${VERSION} already exists. Uploading additional assets..."
    # Upload assets to existing release
    gh release upload "${VERSION}" \
        "${RELEASE_DIR}"/*.tar.gz \
        "${RELEASE_DIR}"/*.zip \
        "${RELEASE_DIR}"/checksums.sha256 \
        --clobber
else
    # Create new release
    gh release create "${VERSION}" \
        --title "Delta CLI ${VERSION}" \
        --notes "${RELEASE_NOTES}" \
        "${RELEASE_DIR}"/*.tar.gz \
        "${RELEASE_DIR}"/*.zip \
        "${RELEASE_DIR}"/checksums.sha256
fi

log_success "GitHub release created/updated successfully!"
log_info "View at: https://github.com/DeltaCLI/delta/releases/tag/${VERSION}"