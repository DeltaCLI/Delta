#!/bin/bash
# Download and verify EnvVarUpdate.nsh for NSIS installer

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

echo -e "${GREEN}EnvVarUpdate.nsh Downloader${NC}"

# Create installer directory if it doesn't exist
mkdir -p installer

# Try different sources for EnvVarUpdate.nsh
URLS=(
    "https://raw.githubusercontent.com/elebertus/nsis-envvarupdate/master/EnvVarUpdate.nsh"
    "https://raw.githubusercontent.com/GsNSIS/EnvVarUpdate/master/EnvVarUpdate.nsh"
    "https://nsis.sourceforge.io/mediawiki/images/a/ad/EnvVarUpdate.7z"
)

ENVVAR_FILE="installer/EnvVarUpdate.nsh"
DOWNLOADED=false

for url in "${URLS[@]}"; do
    echo -e "${YELLOW}Trying: $url${NC}"
    
    if [[ $url == *.7z ]]; then
        # Handle 7z archive
        if command -v 7z &> /dev/null; then
            if curl -fsSL "$url" -o "installer/EnvVarUpdate.7z" 2>/dev/null; then
                7z x -o"installer/" "installer/EnvVarUpdate.7z" EnvVarUpdate.nsh 2>/dev/null
                rm -f "installer/EnvVarUpdate.7z"
                if [ -f "$ENVVAR_FILE" ]; then
                    DOWNLOADED=true
                    break
                fi
            fi
        fi
    else
        # Direct download
        if curl -fsSL "$url" -o "$ENVVAR_FILE" 2>/dev/null; then
            # Verify it's actually the NSIS script
            if head -n 1 "$ENVVAR_FILE" | grep -q "EnvVarUpdate" || head -n 5 "$ENVVAR_FILE" | grep -q "!ifndef"; then
                DOWNLOADED=true
                break
            else
                rm -f "$ENVVAR_FILE"
            fi
        fi
    fi
done

if [ "$DOWNLOADED" = true ]; then
    echo -e "${GREEN}✓ Successfully downloaded EnvVarUpdate.nsh${NC}"
    
    # Calculate SHA256
    if command -v sha256sum &> /dev/null; then
        SHA256=$(sha256sum "$ENVVAR_FILE" | cut -d' ' -f1)
    elif command -v shasum &> /dev/null; then
        SHA256=$(shasum -a 256 "$ENVVAR_FILE" | cut -d' ' -f1)
    else
        echo -e "${RED}Warning: No SHA256 tool available${NC}"
        exit 1
    fi
    
    echo -e "${GREEN}SHA256: $SHA256${NC}"
    echo
    echo "To use this in the build process, set the environment variable:"
    echo -e "${YELLOW}export ENVVAR_SHA256=\"$SHA256\"${NC}"
    echo
    echo "Or add to your build script:"
    echo "ENVVAR_SHA256=\"$SHA256\""
    
    # Also save to a file for reference
    echo "$SHA256" > installer/EnvVarUpdate.sha256
    
else
    echo -e "${RED}Failed to download EnvVarUpdate.nsh${NC}"
    echo "You may need to download it manually from:"
    echo "https://nsis.sourceforge.io/EnvVarUpdate"
    exit 1
fi