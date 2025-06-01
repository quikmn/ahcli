# build-tui.ps1
# Build AHCLI with TUI support

Write-Host "Building AHCLI with TUI support..." -ForegroundColor Green
Write-Host "===================================" -ForegroundColor Green

# Update dependencies
Write-Host "Getting dependencies..." -ForegroundColor Yellow
go mod tidy
if ($LASTEXITCODE -ne 0) {
    Write-Host "Failed to update dependencies" -ForegroundColor Red
    exit 1
}

# Update vendor directory to fix inconsistency
Write-Host "Updating vendor directory..." -ForegroundColor Yellow
go mod vendor
if ($LASTEXITCODE -ne 0) {
    Write-Host "Failed to update vendor directory" -ForegroundColor Red
    exit 1
}

# Build client
Write-Host "Building client..." -ForegroundColor Yellow
Push-Location client
go build -o client.exe .
if ($LASTEXITCODE -ne 0) {
    Write-Host "Failed to build client" -ForegroundColor Red
    Pop-Location
    exit 1
}
Pop-Location

# Build server  
Write-Host "Building server..." -ForegroundColor Yellow
Push-Location server
go build -o server.exe .
if ($LASTEXITCODE -ne 0) {
    Write-Host "Failed to build server" -ForegroundColor Red
    Pop-Location
    exit 1
}
Pop-Location

# Copy executables to build directory
Write-Host "Copying executables..." -ForegroundColor Yellow
if (!(Test-Path "build")) {
    New-Item -ItemType Directory -Path "build"
}

Copy-Item "client\client.exe" "build\"
Copy-Item "server\server.exe" "build\"

Write-Host ""
Write-Host "Build complete!" -ForegroundColor Green
Write-Host ""
Write-Host "Usage:" -ForegroundColor Cyan
Write-Host "  .\build\client.exe           # Start with TUI" -ForegroundColor White
Write-Host "  .\build\client.exe --no-tui  # Console mode" -ForegroundColor White  
Write-Host "  .\build\client.exe --debug   # TUI + debug logs" -ForegroundColor White
Write-Host "  .\build\server.exe           # Start server" -ForegroundColor White
Write-Host "  .\build\server.exe --debug   # Server + debug logs" -ForegroundColor White