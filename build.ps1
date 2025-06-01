# build.ps1
# Build script for AHCLI desktop application

param(
    [switch]$Debug,    # Include debug console
    [switch]$Clean     # Clean build directory first
)

Write-Host "AHCLI Build Script" -ForegroundColor Green
Write-Host "==================" -ForegroundColor Green

# Clean build directory if requested
if ($Clean -and (Test-Path "build")) {
    Write-Host "Cleaning build directory..." -ForegroundColor Yellow
    Remove-Item "build" -Recurse -Force
}

# Create build directory
if (!(Test-Path "build")) {
    New-Item -ItemType Directory -Path "build" | Out-Null
}

# Build server (always with console for admin use)
Write-Host "Building server..." -ForegroundColor Yellow
Set-Location "server"
go build -o "../build/server.exe" .
if ($LASTEXITCODE -ne 0) {
    Write-Host "Server build failed!" -ForegroundColor Red
    exit 1
}
Set-Location ".."

# Build client
Write-Host "Building client..." -ForegroundColor Yellow
Set-Location "client"

if ($Debug) {
    Write-Host "Building with debug console..." -ForegroundColor Cyan
    go build -o "../build/client.exe" .
} else {
    Write-Host "Building as Windows GUI application..." -ForegroundColor Cyan
    go build -ldflags "-H=windowsgui" -o "../build/client.exe" .
}

if ($LASTEXITCODE -ne 0) {
    Write-Host "Client build failed!" -ForegroundColor Red
    exit 1
}
Set-Location ".."

# Copy config files
Write-Host "Copying configuration files..." -ForegroundColor Yellow
Copy-Item "server/config.json" "build/" -Force
Copy-Item "client/settings.config" "build/" -Force

Write-Host ""
Write-Host "Build complete!" -ForegroundColor Green
Write-Host "Output directory: build/" -ForegroundColor Cyan
Write-Host ""
Write-Host "Files created:" -ForegroundColor White
Get-ChildItem "build" | ForEach-Object {
    $size = if ($_.Length -gt 1MB) { "$([math]::Round($_.Length/1MB, 1)) MB" } else { "$([math]::Round($_.Length/1KB, 1)) KB" }
    Write-Host "  $($_.Name) - $size" -ForegroundColor Gray
}
Write-Host ""

if ($Debug) {
    Write-Host "Debug build - console window will show" -ForegroundColor Yellow
} else {
    Write-Host "GUI build - no console window (use -Debug flag to enable console)" -ForegroundColor Green
}

Write-Host ""
Write-Host "Usage:" -ForegroundColor Cyan
Write-Host "  Server: .\build\server.exe" -ForegroundColor Gray
Write-Host "  Client: .\build\client.exe" -ForegroundColor Gray