# Save as: dump-ahcli-minimal.ps1
# Ultra minimal dump - source code ONLY
Add-Type -AssemblyName System.Windows.Forms

$root = Get-Location
$sb = [System.Text.StringBuilder]::new()

$null = $sb.AppendLine("=== ahcli Minimal Source Dump ===")

# Only show actual source files in tree
$null = $sb.AppendLine("`n--- Source Files Only ---")
Get-ChildItem -Recurse -File | Where-Object {
    # ONLY include actual source files we care about
    ($_.Extension -in @('.go', '.html', '.css', '.js')) -and
    ($_.DirectoryName -match '(client|server|common)') -and
    ($_.DirectoryName -notmatch '(build|chromium|vendor|\.git|node_modules)')
} | ForEach-Object {
    $relativePath = $_.FullName.Replace($root, '').TrimStart('\')
    $null = $sb.AppendLine($relativePath)
}

# go.mod only (skip go.sum - too big)
if (Test-Path "$root\go.mod") {
    $null = $sb.AppendLine("`n--- go.mod ---")
    $null = $sb.AppendLine((Get-Content "$root\go.mod" -Raw))
}

# Source code from main components ONLY
$components = @("client", "server", "common")
foreach ($comp in $components) {
    $path = Join-Path $root $comp
    if (Test-Path $path) {
        $null = $sb.AppendLine("`n=== $comp/ Source Files ===")
        
        # Get .go files only
        Get-ChildItem "$path\*.go" -Recurse -ErrorAction SilentlyContinue | ForEach-Object {
            $relativePath = $_.FullName.Replace($root, '').TrimStart('\')
            $null = $sb.AppendLine("`n--- $relativePath ---")
            $null = $sb.AppendLine((Get-Content $_.FullName -Raw))
        }
        
        # Get web files only from client/web/
        if ($comp -eq "client" -and (Test-Path "$path\web")) {
            Get-ChildItem "$path\web\*" -Include "*.html", "*.css", "*.js" -ErrorAction SilentlyContinue | ForEach-Object {
                $relativePath = $_.FullName.Replace($root, '').TrimStart('\')
                $null = $sb.AppendLine("`n--- $relativePath ---")
                $null = $sb.AppendLine((Get-Content $_.FullName -Raw))
            }
        }
    }
}

# Only the two config files we actually use
$configs = @(
    @{Path="client\settings.config"; Name="Client Config"},
    @{Path="server\config.json"; Name="Server Config"}
)
foreach ($cfg in $configs) {
    $fullPath = Join-Path $root $cfg.Path
    if (Test-Path $fullPath) {
        $null = $sb.AppendLine("`n=== $($cfg.Name): $($cfg.Path) ===")
        $null = $sb.AppendLine((Get-Content $fullPath -Raw))
    }
}

$null = $sb.AppendLine("`n=== Source Dump Complete ===")

# Copy to clipboard
[System.Windows.Forms.Clipboard]::SetText($sb.ToString())
Write-Host "Minimal source dump copied to clipboard."
Write-Host "Included: .go files, web files (HTML/CSS/JS), core configs only"
Write-Host "Excluded: ALL binaries, build artifacts, chromium, vendor, etc."