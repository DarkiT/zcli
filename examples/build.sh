#!/bin/bash

# Get build information
VERSION="1.0.0"
GIT_COMMIT=$(git rev-parse HEAD)
GIT_BRANCH=$(git rev-parse --abbrev-ref HEAD)
GIT_TAG=$(git describe --tags --abbrev=0 2>/dev/null || echo "v0.0.0")
BUILD_TIME=$(date -u '+%Y-%m-%dT%H:%M:%SZ')

# Build flags
BUILD_FLAGS=(
    "-X main.version=${VERSION}"
    "-X buildInfo.buildTime=${BUILD_TIME}"
    "-X buildInfo.gitCommit=${GIT_COMMIT}"
    "-X buildInfo.gitBranch=${GIT_BRANCH}"
    "-X buildInfo.gitTag=${GIT_TAG}"
)

# Build for different platforms
PLATFORMS=("windows/amd64" "linux/amd64" "darwin/amd64")

for PLATFORM in "${PLATFORMS[@]}"; do
    GOOS=${PLATFORM%/*}
    GOARCH=${PLATFORM#*/}
    BIN_NAME="myapp"
    if [ "$GOOS" = "windows" ]; then
        BIN_NAME="${BIN_NAME}.exe"
    fi

    echo "Building for $GOOS/$GOARCH..."
    GOOS=$GOOS GOARCH=$GOARCH go build \
        -ldflags "${BUILD_FLAGS[*]}" \
        -o "bin/${GOOS}-${GOARCH}/${BIN_NAME}" \
        .
    chmod +x "bin/${GOOS}-${GOARCH}/${BIN_NAME}"
done

echo "Build complete!"