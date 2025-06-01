REM run-client.bat  
@echo off
echo Starting AHCLI Client from dev environment...
cd /d "%~dp0client"
if exist client.exe (
    client.exe
) else (
    echo ERROR: client.exe not found in client folder
    echo Run build.bat first to build the project
    pause
)