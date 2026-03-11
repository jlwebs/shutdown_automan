#!/bin/bash

# Color codes
GREEN='\033[0;32m'
RED='\033[0;31m'
NC='\033[0m' # No Color

# Ensure go is in PATH (common install location on macOS)
export PATH=$PATH:/usr/local/go/bin

echo -e "${GREEN}Starting build process for Windows...${NC}"

# Check if Go is installed
if ! command -v go &> /dev/null
then
    echo -e "${RED}Error: Go is not installed or not in your PATH.${NC}"
    echo "Please download and install Go from https://go.dev/dl/"
    exit 1
fi

# Create release directory
mkdir -p release

# Initialize module if not already done (though main.go exists)
if [ ! -f go.mod ]; then
    echo "Initializing Go module..."
    go mod init shutdown_automan
fi

echo "Downloading dependencies..."
go mod tidy

# Build command
# CGO_ENABLED=0 is important for cross-compilation from Mac to Windows to avoid C dependency issues
# -ldflags="-H=windowsgui" hides the console window

# Check and install rsrc for icon embedding
if ! command -v rsrc &> /dev/null; then
    echo "Installing rsrc for icon embedding..."
    go install github.com/akavel/rsrc@latest
    # Add likely install path to PATH
    export PATH=$PATH:$(go env GOPATH)/bin
fi

# Generate .syso files for resource embedding (Icon + Manifest)
if [ -f "app.ico" ]; then
    echo "Generating resources with app.ico..."
    # Generate syso for AMD64
    rsrc -manifest app.manifest -ico app.ico -arch amd64 -o rsrc_windows_amd64.syso
    # Generate syso for ARM64
    rsrc -manifest app.manifest -ico app.ico -arch arm64 -o rsrc_windows_arm64.syso
else
    echo "Warning: app.ico not found. Executable will not have an icon."
fi

# Clean old executables
rm -f release/*.exe

echo "Compiling for Windows x64 (AMD64)..."
env CGO_ENABLED=0 GOOS=windows GOARCH=amd64 go build -ldflags="-H=windowsgui" -o release/RemoteRestartService.exe

echo "Compiling for Windows ARM64..."
env CGO_ENABLED=0 GOOS=windows GOARCH=arm64 go build -ldflags="-H=windowsgui" -o release/RemoteRestartService_arm64.exe

if [ $? -eq 0 ]; then
    echo -e "${GREEN}Build successful!${NC}"
    echo "Files in release/ folder:"
    ls -F release/
    
    # Create default config in release folder
    if [ ! -f release/config.json ]; then
        echo "Creating default config.json..."
        echo '{
  "port": "8080",
  "process_list": [],
  "monitor_enabled": false,
  "monitor_interval": 60
}' > release/config.json
    fi
    
    # Copy and rename manifest for each exe
    if [ -f app.manifest ]; then
        cp app.manifest release/RemoteRestartService.exe.manifest
        cp app.manifest release/RemoteRestartService_arm64.exe.manifest
        echo "Created matching .manifest files for both executables."
    fi
else
    echo -e "${RED}One or more builds failed.${NC}"
fi
