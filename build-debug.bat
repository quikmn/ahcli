@echo off
echo Building AHCLI (Debug Mode)...

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

:: Build client (with console for debugging)
echo Building client (debug mode)...
cd client
go build -o ../build/client-debug.exe .
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

echo.
echo Debug build complete! Files in build/ directory:
dir build /b

echo.
echo Debug client built - console window will show for debugging
pause