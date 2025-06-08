#!/bin/bash
set -e

# Delta CLI Release Creation Script
# This script creates release binaries, checksums, and uploads to GitHub

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"
cd "$PROJECT_ROOT"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Helper functions
log_info() {
    echo -e "${BLUE}[INFO]${NC} $1"
}

log_success() {
    echo -e "${GREEN}[SUCCESS]${NC} $1"
}

log_warning() {
    echo -e "${YELLOW}[WARNING]${NC} $1"
}

log_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

# Check if version tag is provided
if [ $# -eq 0 ]; then
    log_error "Usage: $0 <version-tag>"
    log_info "Example: $0 v0.1.0-alpha"
    exit 1
fi

VERSION_TAG="$1"
TIMESTAMP=$(date +"%Y%m%d_%H%M%S")
RELEASE_BASE_DIR="releases"
RELEASE_DIR="$RELEASE_BASE_DIR/${VERSION_TAG}_${TIMESTAMP}"
BUILD_DIR="build"

log_info "Creating release for version: $VERSION_TAG"

# Validate version tag format
if [[ ! "$VERSION_TAG" =~ ^v[0-9]+\.[0-9]+\.[0-9]+(-[a-zA-Z0-9]+)?$ ]]; then
    log_error "Invalid version tag format. Expected: vX.Y.Z or vX.Y.Z-suffix"
    exit 1
fi

# Check if git tag exists
if ! git tag -l | grep -q "^$VERSION_TAG$"; then
    log_error "Git tag '$VERSION_TAG' does not exist"
    log_info "Create the tag first with: git tag -a $VERSION_TAG -m 'Release $VERSION_TAG'"
    exit 1
fi

# Check if repository is clean (no uncommitted changes)
if ! git diff --quiet 2>/dev/null; then
    log_error "Repository has uncommitted changes. Commit or stash changes before creating release."
    log_info "Use 'git status' to see uncommitted changes"
    exit 1
fi

# Check if there are untracked files that might affect the build
if [ -n "$(git status --porcelain 2>/dev/null)" ]; then
    log_warning "Repository has untracked files. This may not affect the build, but consider committing important files."
    log_info "Untracked files:"
    git status --porcelain
    read -p "Continue with release? (y/N): " -n 1 -r
    echo
    if [[ ! $REPLY =~ ^[Yy]$ ]]; then
        log_info "Release cancelled"
        exit 1
    fi
fi

# Clean and create release directory
log_info "Setting up release directory..."
mkdir -p "$RELEASE_BASE_DIR"
rm -rf "$RELEASE_DIR"
mkdir -p "$RELEASE_DIR"

# Build binaries for different platforms
log_info "Building binaries..."

# Clean build first
make clean

# Build all supported platforms with explicit version (let IS_DIRTY auto-detect)
log_info "Building for all platforms with version $VERSION_TAG..."
make build-all VERSION="$VERSION_TAG"

# Check that all binaries were built successfully and organize by platform folders
PLATFORMS="linux/amd64 darwin/amd64 darwin/arm64 windows/amd64"
for platform in $PLATFORMS; do
    os=$(echo $platform | cut -d'/' -f1)
    arch=$(echo $platform | cut -d'/' -f2)
    
    if [ "$os" = "windows" ]; then
        binary_name="delta.exe"
    else
        binary_name="delta"
    fi
    
    if [ ! -f "$BUILD_DIR/$platform/$binary_name" ]; then
        log_error "Failed to build $platform binary"
        exit 1
    fi
    
    # Create platform-specific directory in release
    platform_dir="$RELEASE_DIR/$os-$arch"
    mkdir -p "$platform_dir"
    
    # Copy binary to platform directory with simple name
    cp "$BUILD_DIR/$platform/$binary_name" "$platform_dir/$binary_name"
    
    log_success "$platform binary created in $os-$arch/"
done

# Copy important documentation files
log_info "Including documentation files..."
cp README.md "$RELEASE_DIR/" 2>/dev/null || log_warning "README.md not found"
cp LICENSE.md "$RELEASE_DIR/" 2>/dev/null || log_warning "LICENSE.md not found"
cp RELEASE_NOTES/RELEASE_NOTES_${VERSION_TAG}.md "$RELEASE_DIR/" 2>/dev/null || log_warning "Release notes not found"

# Copy user guide if it exists
if [ -f "UserGuide.md" ]; then
    cp UserGuide.md "$RELEASE_DIR/"
    log_success "Included UserGuide.md"
fi

log_success "Documentation files included"

# Create compressed archives
log_info "Creating compressed archives..."

cd "$RELEASE_DIR"

# Create archives for each platform
PLATFORMS="linux/amd64 darwin/amd64 darwin/arm64 windows/amd64"
for platform in $PLATFORMS; do
    os=$(echo $platform | cut -d'/' -f1)
    arch=$(echo $platform | cut -d'/' -f2)
    platform_dir="$os-$arch"
    archive_base="delta-${VERSION_TAG}-$os-$arch"
    
    if [ "$os" = "windows" ]; then
        binary_name="delta.exe"
    else
        binary_name="delta"
    fi
    
    # Create a temporary directory for the archive contents
    temp_archive_dir="temp_$platform_dir"
    mkdir -p "$temp_archive_dir"
    
    # Copy binary from platform directory
    cp "$platform_dir/$binary_name" "$temp_archive_dir/"
    
    # Copy documentation files
    cp *.md "$temp_archive_dir/" 2>/dev/null || true
    
    # Create tar.gz archive
    tar -czf "${archive_base}.tar.gz" -C "$temp_archive_dir" .
    log_success "Created ${archive_base}.tar.gz"
    
    # Create zip archive
    cd "$temp_archive_dir" && zip -q "../${archive_base}.zip" * && cd ..
    log_success "Created ${archive_base}.zip"
    
    # Clean up temporary directory
    rm -rf "$temp_archive_dir"
done

# Generate checksums
log_info "Generating checksums..."

# Create SHA256 checksums for all files
sha256sum *.tar.gz > checksums.sha256 2>/dev/null || touch checksums.sha256
sha256sum *.zip >> checksums.sha256 2>/dev/null || true

# Add checksums for individual binaries in platform directories
for platform_dir in linux-amd64 darwin-amd64 darwin-arm64 windows-amd64; do
    if [ -d "$platform_dir" ]; then
        if [ "$platform_dir" = "windows-amd64" ]; then
            binary_name="delta.exe"
        else
            binary_name="delta"
        fi
        if [ -f "$platform_dir/$binary_name" ]; then
            sha256sum "$platform_dir/$binary_name" >> checksums.sha256
        fi
    fi
done

log_success "Generated SHA256 checksums"

# Display checksums
log_info "SHA256 Checksums:"
cat checksums.sha256

# Create release info file
cat > release-info.txt << EOF
Delta CLI Release Information
============================

Version: $VERSION_TAG
Build Date: $(date -u '+%Y-%m-%d %H:%M:%S UTC')
Git Commit: $(git rev-parse HEAD)
Git Tag: $VERSION_TAG

Supported Platforms:
- Linux AMD64 (x86_64)
- macOS Intel (x86_64) 
- macOS Apple Silicon (ARM64)
- Windows AMD64 (x86_64)

Files in this release:
Archives (binary + documentation):
- delta-${VERSION_TAG}-linux-amd64.tar.gz / .zip
- delta-${VERSION_TAG}-darwin-amd64.tar.gz / .zip (macOS Intel)
- delta-${VERSION_TAG}-darwin-arm64.tar.gz / .zip (macOS Apple Silicon)
- delta-${VERSION_TAG}-windows-amd64.tar.gz / .zip

Raw binaries:
- linux-amd64/delta (Linux AMD64 binary)
- darwin-amd64/delta (macOS Intel binary)  
- darwin-arm64/delta (macOS Apple Silicon binary)
- windows-amd64/delta.exe (Windows AMD64 binary)

Documentation:
- README.md (Project documentation and installation instructions)
- LICENSE.md (MIT License)
- RELEASE_NOTES_${VERSION_TAG}.md (Release notes for this version)
- checksums.sha256 (SHA256 checksums for all files)
- release-info.txt (This file)

Installation:
1. Download the appropriate archive for your platform
2. Extract: tar -xzf delta-${VERSION_TAG}-<platform>.tar.gz
3. Read the README.md for detailed installation instructions
4. Make executable (Unix/macOS): chmod +x delta
5. Move to PATH: 
   - Unix/macOS: sudo mv delta /usr/local/bin/delta
   - Windows: Move delta.exe to a directory in your PATH
6. Verify installation: delta --version

Verification:
To verify the integrity of downloaded files, check against the provided checksums:
- sha256sum -c checksums.sha256

Copyright (c) 2025 Source Parts Inc.
License: MIT
EOF

log_success "Created release-info.txt"

cd "$PROJECT_ROOT"

# Create release report in base releases directory
RELEASE_REPORT="$RELEASE_BASE_DIR/release-report-${VERSION_TAG}-${TIMESTAMP}.txt"
cat > "$RELEASE_REPORT" << EOF
Delta CLI Release Report
========================

Release: $VERSION_TAG
Timestamp: $TIMESTAMP
Build Date: $(date -u '+%Y-%m-%d %H:%M:%S UTC')
Git Commit: $(git rev-parse HEAD)
Release Directory: $RELEASE_DIR

Build Summary:
- Platforms: Linux AMD64, macOS Intel, macOS Apple Silicon, Windows AMD64
- Linux Binary Size: $(stat -c%s "$RELEASE_DIR/linux-amd64/delta" 2>/dev/null | numfmt --to=iec || echo "N/A")
- macOS Intel Binary Size: $(stat -c%s "$RELEASE_DIR/darwin-amd64/delta" 2>/dev/null | numfmt --to=iec || echo "N/A")
- macOS ARM64 Binary Size: $(stat -c%s "$RELEASE_DIR/darwin-arm64/delta" 2>/dev/null | numfmt --to=iec || echo "N/A")
- Windows Binary Size: $(stat -c%s "$RELEASE_DIR/windows-amd64/delta.exe" 2>/dev/null | numfmt --to=iec || echo "N/A")

SHA256 Checksums:
$(cat "$RELEASE_DIR/checksums.sha256")

Release Files:
$(ls -la "$RELEASE_DIR")

Total Release Size: $(du -sh "$RELEASE_DIR" | cut -f1)

Build Status: SUCCESS
EOF

# List all release files
log_info "Release files created:"
ls -la "$RELEASE_DIR"

# Calculate total size
TOTAL_SIZE=$(du -sh "$RELEASE_DIR" | cut -f1)
log_info "Total release size: $TOTAL_SIZE"

log_success "Release report created: $RELEASE_REPORT"

# Optional: Upload to GitHub if gh CLI is available and --upload flag is passed
if [ "$2" = "--upload" ]; then
    if command -v gh &> /dev/null; then
        log_info "Uploading release to GitHub..."
        
        # Check if release exists
        if gh release view "$VERSION_TAG" &> /dev/null; then
            log_warning "Release $VERSION_TAG already exists on GitHub"
            read -p "Do you want to delete and recreate it? (y/N): " -n 1 -r
            echo
            if [[ $REPLY =~ ^[Yy]$ ]]; then
                gh release delete "$VERSION_TAG" --yes
                log_info "Deleted existing release"
            else
                log_info "Skipping upload"
                exit 0
            fi
        fi
        
        # Create release
        log_info "Creating GitHub release..."
        gh release create "$VERSION_TAG" \
            --title "$VERSION_TAG: Multilingual Delta - Internationalization Alpha Release" \
            --notes-file "RELEASE_NOTES/RELEASE_NOTES_${VERSION_TAG}.md" \
            --prerelease \
            "$RELEASE_DIR"/*.tar.gz \
            "$RELEASE_DIR"/*.zip \
            "$RELEASE_DIR"/checksums.sha256 \
            "$RELEASE_DIR"/release-info.txt
        
        log_success "Release uploaded to GitHub!"
        gh release view "$VERSION_TAG" --web
    else
        log_warning "GitHub CLI (gh) not found. Skipping upload."
        log_info "To upload manually, run:"
        log_info "gh release create $VERSION_TAG --title '$VERSION_TAG: Release' --notes-file RELEASE_NOTES/RELEASE_NOTES_${VERSION_TAG}.md --prerelease $RELEASE_DIR/*"
    fi
fi

log_success "Release creation completed successfully!"
log_info "Release files are in: $RELEASE_DIR/"
log_info "To upload to GitHub: $0 $VERSION_TAG --upload"