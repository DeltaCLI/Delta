#!/bin/bash
# Build Windows installer on Linux without Wine
# Uses NSIS instead of Inno Setup for better Linux compatibility

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

echo -e "${GREEN}Delta Windows Installer Builder (Linux)${NC}"

# Check if NSIS is installed
if ! command -v makensis &> /dev/null; then
    echo -e "${RED}Error: NSIS (makensis) is not installed.${NC}"
    echo "Install with:"
    echo "  Ubuntu/Debian: sudo apt-get install nsis"
    echo "  Fedora: sudo dnf install mingw32-nsis"
    echo "  Arch: sudo pacman -S mingw-w64-nsis"
    exit 1
fi

# Get version from git tag or use default
VERSION=$(git describe --tags --abbrev=0 2>/dev/null || echo "v0.4.5-alpha")
VERSION_NUMBER=${VERSION#v}  # Remove 'v' prefix

echo -e "${YELLOW}Building installer for Delta ${VERSION}${NC}"

# Expected SHA256 for EnvVarUpdate.nsh (can be overridden by environment)
: ${ENVVAR_SHA256:="99bee48a2f7ad708b6ea202dbabf27246f5459b04cfc13c29430d9c4086a3947"}

# Download and verify EnvVarUpdate.nsh if not present
ENVVAR_FILE="installer/EnvVarUpdate.nsh"
if [ ! -f "$ENVVAR_FILE" ]; then
    echo -e "${YELLOW}EnvVarUpdate.nsh not found. Downloading...${NC}"
    ./scripts/download-envvarupdate.sh
fi

# Verify checksum if file exists
if [ -f "$ENVVAR_FILE" ]; then
    if command -v sha256sum &> /dev/null; then
        ACTUAL_SHA=$(sha256sum "$ENVVAR_FILE" | cut -d' ' -f1)
    elif command -v shasum &> /dev/null; then
        ACTUAL_SHA=$(shasum -a 256 "$ENVVAR_FILE" | cut -d' ' -f1)
    fi
    
    if [ -n "$ACTUAL_SHA" ]; then
        if [ "$ACTUAL_SHA" != "$ENVVAR_SHA256" ]; then
            echo -e "${RED}Error: EnvVarUpdate.nsh checksum mismatch!${NC}"
            echo "Expected: $ENVVAR_SHA256"
            echo "Actual:   $ACTUAL_SHA"
            echo "Please verify the file or update ENVVAR_SHA256"
            exit 1
        else
            echo -e "${GREEN}✓ EnvVarUpdate.nsh checksum verified${NC}"
        fi
    fi
fi

# Build Windows executable if not exists
if [ ! -f "build/windows/amd64/delta.exe" ]; then
    echo "Building Windows executable..."
    make build-all
fi

# Create NSIS installer script
cat > installer/delta-installer.nsi << EOF
; Delta CLI NSIS Installer Script
; Generated for Linux build environment

!define PRODUCT_NAME "Delta CLI"
!define PRODUCT_VERSION "${VERSION_NUMBER}"
!define PRODUCT_PUBLISHER "Delta Task Force"
!define PRODUCT_WEB_SITE "https://github.com/deltacli/delta"
!define PRODUCT_UNINST_KEY "Software\\Microsoft\\Windows\\CurrentVersion\\Uninstall\\\${PRODUCT_NAME}"
!define PRODUCT_UNINST_ROOT_KEY "HKLM"

!include "MUI2.nsh"
!include "EnvVarUpdate.nsh"

; MUI Settings
!define MUI_ABORTWARNING
!define MUI_ICON "\${NSISDIR}\\Contrib\\Graphics\\Icons\\modern-install.ico"
!define MUI_UNICON "\${NSISDIR}\\Contrib\\Graphics\\Icons\\modern-uninstall.ico"

; Pages
!insertmacro MUI_PAGE_WELCOME
!insertmacro MUI_PAGE_LICENSE "../LICENSE.md"
!insertmacro MUI_PAGE_DIRECTORY
!insertmacro MUI_PAGE_INSTFILES
!insertmacro MUI_PAGE_FINISH

!insertmacro MUI_UNPAGE_INSTFILES

!insertmacro MUI_LANGUAGE "English"

Name "\${PRODUCT_NAME} \${PRODUCT_VERSION}"
OutFile "../build/installer/delta-setup-\${PRODUCT_VERSION}.exe"
InstallDir "\$PROGRAMFILES64\\Delta"
ShowInstDetails show
ShowUnInstDetails show

Section "MainSection" SEC01
  SetOutPath "\$INSTDIR"
  SetOverwrite try
  
  ; Main executable
  File "../build/windows/amd64/delta.exe"
  
  ; Documentation
  File "../LICENSE.md"
  File "../README.md"
  File "../UserGuide.md"
  File "../CHANGELOG.md"
  
  ; Resources
  File /r "../i18n"
  File /r "../templates"
  File /r "../embedded_patterns"
  
  ; Add to PATH using EnvVarUpdate
  \${EnvVarUpdate} \$0 "PATH" "A" "HKLM" "\$INSTDIR"
  
  ; Create shortcuts
  CreateDirectory "\$SMPROGRAMS\\Delta CLI"
  CreateShortCut "\$SMPROGRAMS\\Delta CLI\\Delta.lnk" "\$INSTDIR\\delta.exe"
  CreateShortCut "\$DESKTOP\\Delta CLI.lnk" "\$INSTDIR\\delta.exe"
SectionEnd

Section -Post
  WriteUninstaller "\$INSTDIR\\uninst.exe"
  WriteRegStr \${PRODUCT_UNINST_ROOT_KEY} "\${PRODUCT_UNINST_KEY}" "DisplayName" "\$(^Name)"
  WriteRegStr \${PRODUCT_UNINST_ROOT_KEY} "\${PRODUCT_UNINST_KEY}" "UninstallString" "\$INSTDIR\\uninst.exe"
  WriteRegStr \${PRODUCT_UNINST_ROOT_KEY} "\${PRODUCT_UNINST_KEY}" "DisplayVersion" "\${PRODUCT_VERSION}"
  WriteRegStr \${PRODUCT_UNINST_ROOT_KEY} "\${PRODUCT_UNINST_KEY}" "URLInfoAbout" "\${PRODUCT_WEB_SITE}"
  WriteRegStr \${PRODUCT_UNINST_ROOT_KEY} "\${PRODUCT_UNINST_KEY}" "Publisher" "\${PRODUCT_PUBLISHER}"
SectionEnd

Section Uninstall
  ; Remove from PATH using EnvVarUpdate
  \${un.EnvVarUpdate} \$0 "PATH" "R" "HKLM" "\$INSTDIR"
  
  Delete "\$INSTDIR\\uninst.exe"
  Delete "\$INSTDIR\\delta.exe"
  Delete "\$INSTDIR\\LICENSE.md"
  Delete "\$INSTDIR\\README.md"
  Delete "\$INSTDIR\\UserGuide.md"
  Delete "\$INSTDIR\\CHANGELOG.md"
  
  RMDir /r "\$INSTDIR\\i18n"
  RMDir /r "\$INSTDIR\\templates"
  RMDir /r "\$INSTDIR\\embedded_patterns"
  RMDir "\$INSTDIR"
  
  Delete "\$SMPROGRAMS\\Delta CLI\\Delta.lnk"
  RMDir "\$SMPROGRAMS\\Delta CLI"
  Delete "\$DESKTOP\\Delta CLI.lnk"
  
  DeleteRegKey \${PRODUCT_UNINST_ROOT_KEY} "\${PRODUCT_UNINST_KEY}"
  SetAutoClose true
SectionEnd
EOF

# Create output directory
mkdir -p build/installer

# Build installer
echo -e "${YELLOW}Running NSIS compiler...${NC}"
cd installer
makensis delta-installer.nsi
cd ..

if [ -f "build/installer/delta-setup-${VERSION_NUMBER}.exe" ]; then
    echo -e "${GREEN}✓ Installer created successfully!${NC}"
    echo -e "  Location: build/installer/delta-setup-${VERSION_NUMBER}.exe"
    echo -e "  Size: $(du -h build/installer/delta-setup-${VERSION_NUMBER}.exe | cut -f1)"
else
    echo -e "${RED}✗ Failed to create installer${NC}"
    exit 1
fi