$root = Get-Location
Write-Host "`n=== ahcli Project Dump @ $root ==="

# List all files
Write-Host "`n--- File Tree ---"
Get-ChildItem -Recurse | Format-Table -AutoSize

# Dump go.mod
if (Test-Path "$root\go.mod") {
    Write-Host "`n--- go.mod ---"
    Get-Content "$root\go.mod"
}

# Dump .go files from folders
foreach ($comp in "client", "server", "common") {
    $path = Join-Path $root $comp
    if (Test-Path $path) {
        Write-Host "`n=== $comp/ Source Files ==="
        Get-ChildItem "$path\*.go" | ForEach-Object {
            Write-Host "`n--- $($_.FullName) ---`n"
            Get-Content $_.FullName
        }
    }
}

# Dump configs
foreach ($cfg in "client\settings.config", "server\config.json") {
    $fullCfgPath = Join-Path $root $cfg
    if (Test-Path $fullCfgPath) {
        Write-Host "`n=== Config: $cfg ==="
        Get-Content $fullCfgPath
    }
}

Write-Host "`n=== Dump Complete ==="
