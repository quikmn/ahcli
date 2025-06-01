# bundle-chromium.ps1
# Downloads portable Chromium for bundling with the app
param(
    [switch]$Force  # Force re-download even if chromium exists
)

Write-Host "AHCLI Chromium Bundler" -ForegroundColor Green
Write-Host "======================" -ForegroundColor Green

$chromiumDir = ".\chromium"
$chromiumZip = "chromium-portable.zip"

# Check if chromium already exists - FIXED SYNTAX
if ((Test-Path $chromiumDir) -and !$Force) {
    Write-Host "Chromium already exists in $chromiumDir" -ForegroundColor Yellow
    Write-Host "Use -Force to re-download" -ForegroundColor Yellow
    exit 0
}

# Create chromium directory
if (Test-Path $chromiumDir) {
    Write-Host "Removing existing chromium..." -ForegroundColor Yellow
    Remove-Item $chromiumDir -Recurse -Force
}

Write-Host "Creating chromium directory..." -ForegroundColor Yellow
New-Item -ItemType Directory -Path $chromiumDir | Out-Null

# Download portable Chromium from a more reliable source
Write-Host "Downloading portable Chromium..." -ForegroundColor Yellow
Write-Host "This may take a few minutes (~120MB download)" -ForegroundColor Cyan

try {
    # Using Chrome for Testing builds - more reliable than ungoogled-chromium
    # This gets the latest stable portable Chrome
    $downloadUrl = "https://edgedl.me.gvt1.com/edgedl/chrome/chrome-for-testing/121.0.6167.85/win64/chrome-win64.zip"
    
    Write-Host "Downloading from: $downloadUrl" -ForegroundColor Gray
    
    # Show progress during download
    $ProgressPreference = 'Continue'
    Invoke-WebRequest -Uri $downloadUrl -OutFile $chromiumZip -UseBasicParsing
    
    Write-Host "Extracting Chromium..." -ForegroundColor Yellow
    Expand-Archive -Path $chromiumZip -DestinationPath $chromiumDir -Force
    
    # Clean up zip file
    Remove-Item $chromiumZip -Force
    
    # The chrome-for-testing build structure is predictable
    $chromePath = "$chromiumDir\chrome-win64"
    $chromeExe = "$chromePath\chrome.exe"
    
    if (Test-Path $chromeExe) {
        Write-Host "Chromium extracted successfully!" -ForegroundColor Green
        Write-Host "Chrome executable found at: $chromeExe" -ForegroundColor Green
        
        # Create a simple batch file to launch in app mode
        $launchScript = @"
@echo off
cd /d "%~dp0\chrome-win64"
chrome.exe --app=http://localhost:%1 --disable-web-security --disable-features=TranslateUI --disable-extensions --no-first-run --disable-default-apps --disable-sync --no-default-browser-check --disable-background-timer-throttling --disable-renderer-backgrounding --disable-backgrounding-occluded-windows --user-data-dir=.\ahcli-profile
"@
        
        $launchScript | Out-File -FilePath "$chromiumDir\launch-app.bat" -Encoding ASCII
        
        Write-Host "Created launch script: $chromiumDir\launch-app.bat" -ForegroundColor Green
        
        # Test the installation
        Write-Host "Testing Chromium installation..." -ForegroundColor Yellow
        $version = & $chromeExe --version 2>$null
        if ($version) {
            Write-Host "✓ Chromium works! Version: $version" -ForegroundColor Green
        } else {
            Write-Host "⚠ Chromium extracted but version check failed" -ForegroundColor Yellow
        }
        
    } else {
        Write-Host "ERROR: chrome.exe not found at expected path: $chromeExe" -ForegroundColor Red
        Write-Host "Listing extracted contents:" -ForegroundColor Yellow
        Get-ChildItem $chromiumDir -Recurse -Name "chrome.exe"
        exit 1
    }
    
} catch {
    Write-Host "ERROR: Failed to download Chromium: $($_.Exception.Message)" -ForegroundColor Red
    Write-Host "" -ForegroundColor Yellow
    Write-Host "Alternative manual download options:" -ForegroundColor Yellow
    Write-Host "1. Chrome for Testing: https://googlechromelabs.github.io/chrome-for-testing/" -ForegroundColor Cyan
    Write-Host "2. Chromium snapshots: https://commondatastorage.googleapis.com/chromium-browser-snapshots/index.html" -ForegroundColor Cyan
    Write-Host "3. Download any portable Chrome and extract to: $chromiumDir\chrome-win64\" -ForegroundColor Cyan
    exit 1
}

Write-Host ""
Write-Host "Chromium bundle complete!" -ForegroundColor Green
Write-Host "Total size:" -ForegroundColor Cyan
$size = (Get-ChildItem $chromiumDir -Recurse | Measure-Object -Property Length -Sum).Sum / 1MB
Write-Host "  ~$([math]::Round($size, 1)) MB" -ForegroundColor White
Write-Host ""
Write-Host "Your app can now launch with bundled Chromium!" -ForegroundColor Green
Write-Host "Test it with: .\chromium\launch-app.bat 8080" -ForegroundColor Cyan