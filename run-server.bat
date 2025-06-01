REM run-server.bat
@echo off
echo Starting AHCLI Server from dev environment...
cd /d "%~dp0server"
if exist server.exe (
    server.exe
) else (
    echo ERROR: server.exe not found in server folder
    echo Run build.bat first to build the project
    pause
)