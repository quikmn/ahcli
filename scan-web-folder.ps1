# PowerShell script to scan client/web folder and copy to clipboard
# Run this from your AHCLI root folder

param(
    [string]$WebPath = "client/web"
)

function Get-FileTree {
    param([string]$Path, [int]$Indent = 0)
    
    $items = Get-ChildItem -Path $Path | Sort-Object @{Expression={$_.PSIsContainer}; Descending=$true}, Name
    
    foreach ($item in $items) {
        $prefix = "  " * $Indent
        if ($item.PSIsContainer) {
            Write-Output "$prefix$($item.Name)/"
            Get-FileTree -Path $item.FullName -Indent ($Indent + 1)
        } else {
            Write-Output "$prefix$($item.Name)"
        }
    }
}

function Get-FileContents {
    param([string]$Path, [string]$RelativePath = "")
    
    $items = Get-ChildItem -Path $Path -Recurse | Where-Object { -not $_.PSIsContainer } | Sort-Object FullName
    
    foreach ($file in $items) {
        $relativeName = $file.FullName.Substring($Path.Length + 1)
        Write-Output ""
        Write-Output "=== FILE: $relativeName ==="
        
        try {
            $content = Get-Content -Path $file.FullName -Raw -ErrorAction Stop
            if ($content) {
                Write-Output $content
            } else {
                Write-Output "[Empty file]"
            }
        } catch {
            Write-Output "[Error reading file: $($_.Exception.Message)]"
        }
    }
}

# Main execution
Write-Host "Scanning client/web folder structure..." -ForegroundColor Green

$output = @()

# Check if web folder exists
if (-not (Test-Path $WebPath)) {
    $output += "ERROR: $WebPath folder not found!"
    $output += "Current directory: $(Get-Location)"
    $output | Set-Clipboard
    Write-Host "Error copied to clipboard" -ForegroundColor Red
    exit 1
}

# Add header
$output += "=== AHCLI client/web Folder Analysis ==="
$output += "Generated: $(Get-Date)"
$output += "Path: $(Resolve-Path $WebPath)"
$output += ""

# Add file tree
$output += "=== FOLDER STRUCTURE ==="
$output += Get-FileTree -Path $WebPath
$output += ""

# Add file contents
$output += "=== FILE CONTENTS ==="
$output += Get-FileContents -Path $WebPath

# Copy to clipboard
$output -join "`n" | Set-Clipboard

Write-Host "✅ Complete web folder analysis copied to clipboard!" -ForegroundColor Green
Write-Host "📁 Scanned: $(Resolve-Path $WebPath)" -ForegroundColor Cyan
Write-Host "📋 Ready to paste anywhere" -ForegroundColor Yellow