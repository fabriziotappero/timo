#!/bin/bash
# build.sh - Build timo for Linux and Windows with version info using ldflags


VERSION="0.4.0"

LINUX_OUTPUT=build/timo
WIN_OUTPUT=build/timo.exe

BUILD_DATE=$(date -u +"%Y-%m-%dT%H:%M:%SZ")

mkdir -p build

# Build for Linux (amd64)
GOOS=linux GOARCH=amd64 go build -ldflags "-X main.version=v$VERSION -X main.buildDate=$BUILD_DATE" -o $LINUX_OUTPUT .
echo "Built $LINUX_OUTPUT with version v$VERSION (Linux amd64)"

# Build for Windows (amd64)
GOOS=windows GOARCH=amd64 go build -ldflags "-X main.version=v$VERSION" -o $WIN_OUTPUT .
echo "Built $WIN_OUTPUT with version v$VERSION (Windows amd64)"
