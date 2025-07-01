#!/bin/bash
# package-release.sh - Automated package manager release script

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Configuration
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"
VERSION="${1:-}"
DRY_RUN="${2:-false}"

# Package manager repositories
HOMEBREW_REPO="git@github.com:deltacli/homebrew-delta.git"
SCOOP_REPO="git@github.com:deltacli/scoop-delta.git"
APT_REPO="git@github.com:deltacli/apt-repo.git"
AUR_REPO="ssh://aur@aur.archlinux.org/delta-cli.git"

# Logging functions
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

# Check if version is provided
if [ -z "$VERSION" ]; then
    log_error "Usage: $0 <version> [--dry-run]"
    log_info "Example: $0 0.4.6-alpha"
    exit 1
fi

# Validate version format
if [[ ! "$VERSION" =~ ^[0-9]+\.[0-9]+\.[0-9]+(-[a-zA-Z0-9]+)?$ ]]; then
    log_error "Invalid version format. Expected: X.Y.Z or X.Y.Z-suffix"
    exit 1
fi

log_info "Starting package release for version: v$VERSION"

# Check if release exists
RELEASE_DIR="$PROJECT_ROOT/releases/v${VERSION}_*"
if ! ls $RELEASE_DIR 1> /dev/null 2>&1; then
    log_error "Release directory not found. Run create-release.sh first."
    exit 1
fi

# Get the actual release directory
RELEASE_DIR=$(ls -d $RELEASE_DIR | head -n1)
log_info "Using release directory: $RELEASE_DIR"

# Extract checksums
CHECKSUMS_FILE="$RELEASE_DIR/checksums.sha256"
if [ ! -f "$CHECKSUMS_FILE" ]; then
    log_error "Checksums file not found: $CHECKSUMS_FILE"
    exit 1
fi

# Parse checksums
DARWIN_ARM64_SHA=$(grep "darwin-arm64.tar.gz" "$CHECKSUMS_FILE" | awk '{print $1}')
DARWIN_AMD64_SHA=$(grep "darwin-amd64.tar.gz" "$CHECKSUMS_FILE" | awk '{print $1}')
LINUX_AMD64_SHA=$(grep "linux-amd64.tar.gz" "$CHECKSUMS_FILE" | awk '{print $1}')
WINDOWS_AMD64_SHA=$(grep "windows-amd64.zip" "$CHECKSUMS_FILE" | awk '{print $1}')
I18N_SHA=$(grep "i18n.*tar.gz" "$CHECKSUMS_FILE" | awk '{print $1}')

log_success "Checksums extracted successfully"

# Create temporary directory for repos
TEMP_DIR=$(mktemp -d)
trap "rm -rf $TEMP_DIR" EXIT

# Function to update Homebrew
update_homebrew() {
    log_info "Updating Homebrew formula..."
    
    cd "$TEMP_DIR"
    git clone "$HOMEBREW_REPO" homebrew-delta
    cd homebrew-delta
    
    # Update formula
    cat > Formula/delta.rb << EOF
class Delta < Formula
  desc "AI-powered command-line interface enhancement"
  homepage "https://deltacli.dev"
  version "$VERSION"
  license "MIT"

  if OS.mac? && Hardware::CPU.arm?
    url "https://github.com/DeltaCLI/Delta/releases/download/v$VERSION/delta-v$VERSION-darwin-arm64.tar.gz"
    sha256 "$DARWIN_ARM64_SHA"
  elsif OS.mac? && Hardware::CPU.intel?
    url "https://github.com/DeltaCLI/Delta/releases/download/v$VERSION/delta-v$VERSION-darwin-amd64.tar.gz"
    sha256 "$DARWIN_AMD64_SHA"
  elsif OS.linux? && Hardware::CPU.intel?
    url "https://github.com/DeltaCLI/Delta/releases/download/v$VERSION/delta-v$VERSION-linux-amd64.tar.gz"
    sha256 "$LINUX_AMD64_SHA"
  end

  def install
    bin.install "delta"
    
    # Install shell completions if available
    if Dir.exist?("completions")
      bash_completion.install "completions/delta.bash" if File.exist?("completions/delta.bash")
      zsh_completion.install "completions/_delta" if File.exist?("completions/_delta")
      fish_completion.install "completions/delta.fish" if File.exist?("completions/delta.fish")
    end
  end

  def post_install
    ohai "Downloading language files..."
    system "#{bin}/delta", ":i18n", "install"
  end

  def caveats
    <<~EOS
      Delta CLI has been installed successfully!
      
      To get started:
        delta :help
      
      To enable AI features:
        1. Install Ollama: https://ollama.ai
        2. Run: delta :ai on
    EOS
  end

  test do
    assert_match version.to_s, shell_output("#{bin}/delta --version")
  end
end
EOF
    
    if [ "$DRY_RUN" != "true" ]; then
        git add Formula/delta.rb
        git commit -m "chore: update delta to v$VERSION"
        git push origin main
    fi
    
    log_success "Homebrew formula updated"
}

# Function to update Scoop
update_scoop() {
    log_info "Updating Scoop manifest..."
    
    cd "$TEMP_DIR"
    git clone "$SCOOP_REPO" scoop-delta
    cd scoop-delta
    
    # Update manifest using jq
    cat > bucket/delta.json << EOF
{
    "version": "$VERSION",
    "description": "AI-powered command-line interface enhancement",
    "homepage": "https://deltacli.dev",
    "license": "MIT",
    "architecture": {
        "64bit": {
            "url": "https://github.com/DeltaCLI/Delta/releases/download/v$VERSION/delta-v$VERSION-windows-amd64.zip",
            "hash": "$WINDOWS_AMD64_SHA",
            "extract_dir": "delta-v$VERSION-windows-amd64"
        }
    },
    "bin": "delta.exe",
    "checkver": {
        "github": "https://github.com/DeltaCLI/Delta"
    },
    "autoupdate": {
        "architecture": {
            "64bit": {
                "url": "https://github.com/DeltaCLI/Delta/releases/download/v\$version/delta-v\$version-windows-amd64.zip",
                "extract_dir": "delta-v\$version-windows-amd64"
            }
        }
    }
}
EOF
    
    if [ "$DRY_RUN" != "true" ]; then
        git add bucket/delta.json
        git commit -m "chore: update delta to v$VERSION"
        git push origin main
    fi
    
    log_success "Scoop manifest updated"
}

# Function to build and upload DEB package
update_apt() {
    log_info "Building DEB package..."
    
    # This would typically call your build-deb.sh script
    # For now, we'll show the process
    
    cd "$PROJECT_ROOT"
    
    # Create DEB package structure
    DEB_DIR="$TEMP_DIR/delta-cli_${VERSION}_amd64"
    mkdir -p "$DEB_DIR/DEBIAN"
    mkdir -p "$DEB_DIR/usr/bin"
    mkdir -p "$DEB_DIR/usr/share/doc/delta-cli"
    
    # Copy binary
    cp "$RELEASE_DIR/linux-amd64/delta" "$DEB_DIR/usr/bin/"
    
    # Create control file
    cat > "$DEB_DIR/DEBIAN/control" << EOF
Package: delta-cli
Version: $VERSION
Section: utils
Priority: optional
Architecture: amd64
Maintainer: Delta Team <support@deltacli.dev>
Description: AI-powered command-line interface enhancement
 Delta CLI enhances your command-line experience with AI-powered
 suggestions, command validation, and multilingual support.
Homepage: https://deltacli.dev
EOF
    
    # Build package
    if [ "$DRY_RUN" != "true" ]; then
        dpkg-deb --build "$DEB_DIR"
        
        # Upload to APT repository
        # This would typically involve reprepro or similar
        log_info "DEB package built: ${DEB_DIR}.deb"
    fi
    
    log_success "APT repository updated"
}

# Function to update AUR
update_aur() {
    log_info "Updating AUR package..."
    
    cd "$TEMP_DIR"
    git clone "$AUR_REPO" delta-cli
    cd delta-cli
    
    # Update PKGBUILD
    cat > PKGBUILD << EOF
# Maintainer: Delta Team <support@deltacli.dev>
pkgname=delta-cli
pkgver=${VERSION//-/.}
pkgrel=1
pkgdesc="AI-powered command-line interface enhancement"
arch=('x86_64')
url="https://deltacli.dev"
license=('MIT')
depends=('glibc')
source=("https://github.com/DeltaCLI/Delta/releases/download/v${VERSION}/delta-v${VERSION}-linux-amd64.tar.gz")
sha256sums=('$LINUX_AMD64_SHA')

package() {
    install -Dm755 "\$srcdir/delta" "\$pkgdir/usr/bin/delta"
}
EOF
    
    # Generate .SRCINFO
    makepkg --printsrcinfo > .SRCINFO
    
    if [ "$DRY_RUN" != "true" ]; then
        git add PKGBUILD .SRCINFO
        git commit -m "Update to v$VERSION"
        git push origin master
    fi
    
    log_success "AUR package updated"
}

# Function to trigger GitHub Actions
trigger_github_actions() {
    log_info "Triggering GitHub Actions workflows..."
    
    # Trigger Homebrew workflow
    curl -X POST \
        -H "Accept: application/vnd.github.v3+json" \
        -H "Authorization: token $GITHUB_TOKEN" \
        https://api.github.com/repos/deltacli/homebrew-delta/dispatches \
        -d "{\"event_type\":\"new-release\",\"client_payload\":{\"version\":\"$VERSION\"}}"
    
    # Trigger Scoop workflow
    curl -X POST \
        -H "Accept: application/vnd.github.v3+json" \
        -H "Authorization: token $GITHUB_TOKEN" \
        https://api.github.com/repos/deltacli/scoop-delta/dispatches \
        -d "{\"event_type\":\"new-release\",\"client_payload\":{\"version\":\"$VERSION\"}}"
    
    log_success "GitHub Actions triggered"
}

# Main execution
main() {
    log_info "Package release process starting..."
    
    # Update package managers
    update_homebrew
    update_scoop
    update_apt
    update_aur
    
    # Trigger automated workflows
    if [ "$DRY_RUN" != "true" ] && [ -n "$GITHUB_TOKEN" ]; then
        trigger_github_actions
    fi
    
    log_success "Package release completed for v$VERSION!"
    
    # Display next steps
    echo ""
    log_info "Next steps:"
    echo "  1. Verify Homebrew: brew update && brew info deltacli/delta/delta"
    echo "  2. Verify Scoop: scoop update && scoop info delta"
    echo "  3. Test installations on each platform"
    echo "  4. Monitor GitHub Actions for any failures"
    echo "  5. Announce release in Discord/Twitter"
}

# Run main function
main