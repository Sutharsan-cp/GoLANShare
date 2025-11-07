#!/bin/bash

echo "Starting GoLANshare..."
echo

# Check if Go is installed
if ! command -v go &> /dev/null; then
    echo "Error: Go is not installed or not in PATH"
    echo "Please install Go from https://golang.org/dl/"
    exit 1
fi

# Navigate to backend directory
cd backend

# Install dependencies
echo "Installing dependencies..."
go mod download
if [ $? -ne 0 ]; then
    echo "Error: Failed to download dependencies"
    exit 1
fi

# Build the application
echo "Building GoLANshare..."
go build -o golanshare
if [ $? -ne 0 ]; then
    echo "Error: Failed to build application"
    exit 1
fi

# Make executable
chmod +x golanshare

# Start the server
echo
echo "Starting GoLANshare server..."
echo "Open your browser to: http://localhost:8081"
echo "Press Ctrl+C to stop the server"
echo

./golanshare