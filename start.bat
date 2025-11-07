@echo off
echo Starting GoLANshare...
echo.

REM Check if Go is installed
go version >nul 2>&1
if %errorlevel% neq 0 (
    echo Error: Go is not installed or not in PATH
    echo Please install Go from https://golang.org/dl/
    pause
    exit /b 1
)

REM Navigate to backend directory
cd backend

REM Install dependencies
echo Installing dependencies...
go mod download
if %errorlevel% neq 0 (
    echo Error: Failed to download dependencies
    pause
    exit /b 1
)

REM Build the application
echo Building GoLANshare...
go build -o golanshare.exe
if %errorlevel% neq 0 (
    echo Error: Failed to build application
    pause
    exit /b 1
)

REM Start the server
echo.
echo Starting GoLANshare server...
echo Open your browser to: http://localhost:8081
echo Press Ctrl+C to stop the server
echo.
golanshare.exe

pause