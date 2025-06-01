@echo off
echo Building AHCLI...

:: Create build directory
if not exist build mkdir build

:: Build server (with console)
echo Building server...
cd server
go build -o ../build/server.exe .
if errorlevel 1 (
    echo Server build failed!
    pause
    exit /b 1
)
cd ..

:: Build client (GUI mode - no console)
echo Building client...
cd client
go build -ldflags "-H=windowsgui" -o ../build/client.exe .
if errorlevel 1 (
    echo Client build failed!
    pause
    exit /b 1
)
cd ..

:: Copy configs
echo Copying configs...
copy server\config.json build\ >nul
copy client\settings.config build\ >nul

:: Clean up any backup files
del build\*.exe~ >nul 2>&1
del build\*~ >nul 2>&1

echo.
echo Build complete! Files in build/ directory:
dir build /b

echo.
echo GUI client built - no console window will appear
echo Use build-debug.bat if you need console output
pause