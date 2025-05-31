# Save as: dump-ahcli-complete.ps1
# Run from the ahcli root directory

Add-Type -AssemblyName System.Windows.Forms
$root = Get-Location
$sb = [System.Text.StringBuilder]::new()

$null = $sb.AppendLine("=== ahcli Project Dump @ $root ===")
$null = $sb.AppendLine("`n--- File Tree ---")

Get-ChildItem -Recurse | ForEach-Object {
    $null = $sb.AppendLine($_.FullName)
}

# go.mod
if (Test-Path "$root\go.mod") {
    $null = $sb.AppendLine("`n--- go.mod ---")
    $null = $sb.AppendLine((Get-Content "$root\go.mod" -Raw))
}

# .go files
$components = @("client", "server", "common")
foreach ($comp in $components) {
    $path = Join-Path $root $comp
    if (Test-Path $path) {
        $null = $sb.AppendLine("`n=== $comp/ Source Files ===")
        Get-ChildItem "$path\*.go" | ForEach-Object {
            $null = $sb.AppendLine("`n--- $($_.FullName) ---")
            $null = $sb.AppendLine((Get-Content $_.FullName -Raw))
        }
    }
}

# config files
$cfgs = @(
    "client\settings.config",
    "server\config.json"
)
foreach ($cfg in $cfgs) {
    $fullCfgPath = Join-Path $root $cfg
    if (Test-Path $fullCfgPath) {
        $null = $sb.AppendLine("`n=== Config: $cfg ===")
        $null = $sb.AppendLine((Get-Content $fullCfgPath -Raw))
    }
}

$null = $sb.AppendLine("`n=== Dump Complete ===")

# Copy to clipboard
[System.Windows.Forms.Clipboard]::SetText($sb.ToString())
Write-Host "ahcli project dump copied to clipboard."
