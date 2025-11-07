@echo off
echo GoLANshare Installation Verification
echo ====================================
echo.

REM Check Go installation
echo [1/4] Checking Go installation...
go version >nul 2>&1
if %errorlevel% neq 0 (
    echo ❌ Go is not installed or not in PATH
    echo Please install Go from https://golang.org/dl/
    goto :error
) else (
    echo ✅ Go is installed
    go version
)
echo.

REM Check project structure
echo [2/4] Checking project structure...
if not exist "backend\main.go" (
    echo ❌ Backend files missing
    goto :error
) else (
    echo ✅ Backend files found
)

if not exist "frontend\index.html" (
    echo ❌ Frontend files missing
    goto :error
) else (
    echo ✅ Frontend files found
)

if not exist "config.json" (
    echo ❌ Configuration file missing
    goto :error
) else (
    echo ✅ Configuration file found
)
echo.

REM Test build
echo [3/4] Testing build...
cd backend
go mod download >nul 2>&1
if %errorlevel% neq 0 (
    echo ❌ Failed to download dependencies
    goto :error
)

go build -o golanshare-test.exe >nul 2>&1
if %errorlevel% neq 0 (
    echo ❌ Build failed
    goto :error
) else (
    echo ✅ Build successful
    del golanshare-test.exe >nul 2>&1
)
cd ..
echo.

REM Check network
echo [4/4] Checking network configuration...
netstat -an | findstr ":8081" >nul 2>&1
if %errorlevel% equ 0 (
    echo ⚠️  Port 8081 is already in use
    echo You may need to change the port in config.json
) else (
    echo ✅ Port 8081 is available
)
echo.

echo ✅ Installation verification complete!
echo.
echo Ready to start GoLANshare:
echo   1. Run start.bat
echo   2. Open http://localhost:8081 in your browser
echo   3. Login with admin/admin
echo.
pause
goto :end

:error
echo.
echo ❌ Installation verification failed!
echo Please check the issues above and try again.
echo.
pause

:end