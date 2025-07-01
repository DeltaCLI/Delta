#!/bin/bash
# Background release builder for Delta CLI
# Runs the release process in background with nohup

set -e

VERSION="${1:-v0.4.7-alpha}"
UPLOAD="${2}"

# Check if version format is valid
if [[ ! "$VERSION" =~ ^v[0-9]+\.[0-9]+\.[0-9]+ ]]; then
    echo "Error: Invalid version format. Expected format: vX.Y.Z[-suffix]"
    exit 1
fi

# Create log file
LOG_FILE="releases/release-${VERSION}-$(date +%Y%m%d_%H%M%S).log"
mkdir -p releases

echo "Starting background release build for ${VERSION}..."
echo "Log file: ${LOG_FILE}"

# Run the release script in background
if [ "$UPLOAD" == "--upload" ]; then
    nohup ./scripts/create-release.sh "${VERSION}" --upload > "${LOG_FILE}" 2>&1 &
else
    nohup ./scripts/create-release.sh "${VERSION}" > "${LOG_FILE}" 2>&1 &
fi

PID=$!
echo "Release build started with PID: ${PID}"
echo "To check progress: tail -f ${LOG_FILE}"
echo "To check if still running: ps -p ${PID}"

# Save PID for later reference
echo "${PID}" > "releases/.release-${VERSION}.pid"

echo ""
echo "The build will continue in the background."
echo "It typically takes 5-10 minutes to build all platforms."