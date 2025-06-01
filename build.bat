@echo off
echo Building AHCLI for deployment...

:: Clean and create build directory
if exist build (
    echo Cleaning existing build directory...
    rmdir /s /q build 2>nul
    if exist build (
        echo Some files locked - force cleaning...
        del /f /q build\*.exe~ >nul 2>&1
        del /f /q build\*.dll >nul 2>&1
        del /f /q build\*.exe >nul 2>&1
    )
)
echo Creating fresh build directory...
if not exist build mkdir build
if not exist build\server mkdir build\server
if not exist build\client mkdir build\client

:: Build server (with console)
echo Building server...
cd server
go build -o ../build/server/server.exe .
if errorlevel 1 (
    echo Server build failed!
    pause
    exit /b 1
)
cd ..

:: Build client (GUI mode - no console)
echo Building client...
cd client
go build -ldflags "-H=windowsgui" -o ../build/client/client.exe .
if errorlevel 1 (
    echo Client build failed!
    pause
    exit /b 1
)
cd ..

:: Copy server config to server folder
echo Copying server config...
copy server\config.json build\server\ >nul

:: Copy client config and dependencies to client folder
echo Copying client config...
copy client\settings.config build\client\ >nul

:: Handle PortAudio DLL with proper renaming (to client folder)
echo Processing PortAudio library...
if exist "client\libportaudio64bit.dll" (
    echo Copying libportaudio64bit.dll as libportaudio.dll...
    copy "client\libportaudio64bit.dll" "build\client\libportaudio.dll" >nul
    if exist "build\client\libportaudio.dll" (
        echo SUCCESS: PortAudio DLL renamed for deployment
    ) else (
        echo ERROR: Failed to copy PortAudio DLL
    )
) else (
    echo WARNING: libportaudio64bit.dll not found in client folder
)

:: Copy any other DLLs to client folder (except the 64bit one we already renamed)
echo Copying other runtime dependencies...
for %%f in ("client\*.dll") do (
    if not "%%~nxf"=="libportaudio64bit.dll" (
        echo Copying %%~nxf...
        copy "%%f" "build\client\" >nul
    )
)

:: Copy chromium for deployment (to root of build folder)
echo Copying Chromium browser...
if exist chromium (
    if not exist build\chromium mkdir build\chromium
    xcopy chromium build\chromium /E /I /Q >nul
    if exist "build\chromium\chrome-win64\chrome.exe" (
        echo SUCCESS: Chromium bundle copied
    ) else (
        echo WARNING: Chromium copy may have failed
    )
) else (
    echo WARNING: Chromium folder not found
)

:: Copy executables to dev environment for local testing
echo Copying executables to dev folders for local testing...
copy build\server\server.exe server\ >nul
copy build\client\client.exe client\ >nul
echo SUCCESS: Executables copied to dev folders

:: Clean up any backup files
echo Cleaning backup files...
del build\server\*.exe~ >nul 2>&1
del build\client\*.exe~ >nul 2>&1
del build\*~ >nul 2>&1

echo.
echo Deployment build complete! Structure:
echo build/
echo   server/
echo     server.exe
echo     config.json
echo   client/
echo     client.exe
echo     settings.config
echo     libportaudio.dll
echo   chromium/
echo     [browser files]
echo.
echo Dev environment updated:
echo   server/server.exe (ready to run)
echo   client/client.exe (ready to run)
echo.
echo For deployment: Copy entire build/ folder to target machine
echo For dev testing: Use run-server.bat and run-client.bat
pause