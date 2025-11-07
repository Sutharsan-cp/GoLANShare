#!/bin/bash

echo "GoLANshare Installation Verification"
echo "===================================="
echo

# Check Go installation
echo "[1/4] Checking Go installation..."
if ! command -v go &> /dev/null; then
    echo "❌ Go is not installed or not in PATH"
    echo "Please install Go from https://golang.org/dl/"
    exit 1
else
    echo "✅ Go is installed"
    go version
fi
echo

# Check project structure
echo "[2/4] Checking project structure..."
if [ ! -f "backend/main.go" ]; then
    echo "❌ Backend files missing"
    exit 1
else
    echo "✅ Backend files found"
fi

if [ ! -f "frontend/index.html" ]; then
    echo "❌ Frontend files missing"
    exit 1
else
    echo "✅ Frontend files found"
fi

if [ ! -f "config.json" ]; then
    echo "❌ Configuration file missing"
    exit 1
else
    echo "✅ Configuration file found"
fi
echo

# Test build
echo "[3/4] Testing build..."
cd backend
go mod download > /dev/null 2>&1
if [ $? -ne 0 ]; then
    echo "❌ Failed to download dependencies"
    exit 1
fi

go build -o golanshare-test > /dev/null 2>&1
if [ $? -ne 0 ]; then
    echo "❌ Build failed"
    exit 1
else
    echo "✅ Build successful"
    rm -f golanshare-test
fi
cd ..
echo

# Check network
echo "[4/4] Checking network configuration..."
if netstat -an 2>/dev/null | grep -q ":8081"; then
    echo "⚠️  Port 8081 is already in use"
    echo "You may need to change the port in config.json"
else
    echo "✅ Port 8081 is available"
fi
echo

echo "✅ Installation verification complete!"
echo
echo "Ready to start GoLANshare:"
echo "  1. Run ./start.sh"
echo "  2. Open http://localhost:8081 in your browser"
echo "  3. Login with admin/admin"
echo