# convert-logging.ps1
# Converts fmt.Printf/fmt.Println calls to appropriate Log* functions

param(
    [switch]$DryRun,  # Just show what would be changed
    [switch]$Backup   # Create .bak files before modifying
)

Write-Host "AHCLI Logging Converter" -ForegroundColor Green
Write-Host "======================" -ForegroundColor Green

# Find all .go files in current directory and subdirectories
$goFiles = Get-ChildItem -Recurse -Filter "*.go" | Where-Object { $_.Name -notlike "*_test.go" }

Write-Host "Found $($goFiles.Count) Go files to process" -ForegroundColor Yellow

# Define replacement patterns with smart categorization
$replacements = @(
    # PTT-related logs
    @{ Pattern = 'fmt\.Printf\("(\[PTT\][^"]*)", ([^)]*)\)'; Replacement = 'LogPTT("$1", $2)' }
    @{ Pattern = 'fmt\.Println\("(\[PTT\][^"]*)"([^)]*)\)'; Replacement = 'LogPTT("$1"$2)' }
    
    # Audio-related logs  
    @{ Pattern = 'fmt\.Printf\("(\[AUDIO\][^"]*)", ([^)]*)\)'; Replacement = 'LogAudio("$1", $2)' }
    @{ Pattern = 'fmt\.Println\("(\[AUDIO\][^"]*)"([^)]*)\)'; Replacement = 'LogAudio("$1"$2)' }
    @{ Pattern = 'fmt\.Printf\("(\[PLAYBACK\][^"]*)", ([^)]*)\)'; Replacement = 'LogPlayback("$1", $2)' }
    @{ Pattern = 'fmt\.Println\("(\[PLAYBACK\][^"]*)"([^)]*)\)'; Replacement = 'LogPlayback("$1"$2)' }
    
    # Network-related logs
    @{ Pattern = 'fmt\.Printf\("(\[NET\][^"]*)", ([^)]*)\)'; Replacement = 'LogNet("$1", $2)' }
    @{ Pattern = 'fmt\.Println\("(\[NET\][^"]*)"([^)]*)\)'; Replacement = 'LogNet("$1"$2)' }
    @{ Pattern = 'fmt\.Printf\("(\[SEND\][^"]*)", ([^)]*)\)'; Replacement = 'LogSend("$1", $2)' }
    @{ Pattern = 'fmt\.Println\("(\[SEND\][^"]*)"([^)]*)\)'; Replacement = 'LogSend("$1"$2)' }
    
    # Test-related logs
    @{ Pattern = 'fmt\.Printf\("(\[TEST\][^"]*)", ([^)]*)\)'; Replacement = 'LogTest("$1", $2)' }
    @{ Pattern = 'fmt\.Println\("(\[TEST\][^"]*)"([^)]*)\)'; Replacement = 'LogTest("$1"$2)' }
    
    # Main-related logs  
    @{ Pattern = 'fmt\.Printf\("(\[MAIN\][^"]*)", ([^)]*)\)'; Replacement = 'LogMain("$1", $2)' }
    @{ Pattern = 'fmt\.Println\("(\[MAIN\][^"]*)"([^)]*)\)'; Replacement = 'LogMain("$1"$2)' }
    
    # Client-related logs (server files)
    @{ Pattern = 'fmt\.Printf\("(\[CLIENT\][^"]*)", ([^)]*)\)'; Replacement = 'LogClient("$1", $2)' }
    @{ Pattern = 'fmt\.Println\("(\[CLIENT\][^"]*)"([^)]*)\)'; Replacement = 'LogClient("$1"$2)' }
    
    # Error patterns
    @{ Pattern = 'fmt\.Printf\("([^"]*error[^"]*)", ([^)]*)\)'; Replacement = 'LogError("$1", $2)' }
    @{ Pattern = 'fmt\.Println\("([^"]*error[^"]*)"([^)]*)\)'; Replacement = 'LogError("$1"$2)' }
    @{ Pattern = 'fmt\.Printf\("([^"]*Error[^"]*)", ([^)]*)\)'; Replacement = 'LogError("$1", $2)' }
    @{ Pattern = 'fmt\.Println\("([^"]*Error[^"]*)"([^)]*)\)'; Replacement = 'LogError("$1"$2)' }
    @{ Pattern = 'fmt\.Printf\("([^"]*failed[^"]*)", ([^)]*)\)'; Replacement = 'LogError("$1", $2)' }
    @{ Pattern = 'fmt\.Println\("([^"]*failed[^"]*)"([^)]*)\)'; Replacement = 'LogError("$1"$2)' }
    
    # Generic debug patterns (catch-all for anything with brackets)
    @{ Pattern = 'fmt\.Printf\("(\[[A-Z]+\][^"]*)", ([^)]*)\)'; Replacement = 'LogDebug("$1", $2)' }
    @{ Pattern = 'fmt\.Println\("(\[[A-Z]+\][^"]*)"([^)]*)\)'; Replacement = 'LogDebug("$1"$2)' }
    
    # Generic info patterns (no brackets, not errors)
    @{ Pattern = 'fmt\.Printf\("([^"\[\]]*)", ([^)]*)\)'; Replacement = 'LogInfo("$1", $2)' }
    @{ Pattern = 'fmt\.Println\("([^"\[\]]*)"([^)]*)\)'; Replacement = 'LogInfo("$1"$2)' }
)

foreach ($file in $goFiles) {
    Write-Host "`nProcessing: $($file.FullName)" -ForegroundColor Cyan
    
    $content = Get-Content $file.FullName -Raw
    $originalContent = $content
    $changesMade = 0
    
    # Apply each replacement pattern
    foreach ($replacement in $replacements) {
        $matches = [regex]::Matches($content, $replacement.Pattern)
        if ($matches.Count -gt 0) {
            $content = [regex]::Replace($content, $replacement.Pattern, $replacement.Replacement)
            $changesMade += $matches.Count
            
            if ($DryRun) {
                foreach ($match in $matches) {
                    Write-Host "  WOULD CHANGE: $($match.Value)" -ForegroundColor Yellow
                    Write-Host "             TO: $($match.Value -replace $replacement.Pattern, $replacement.Replacement)" -ForegroundColor Green
                }
            }
        }
    }
    
    if ($changesMade -gt 0) {
        Write-Host "  Found $changesMade replacements" -ForegroundColor Green
        
        if (-not $DryRun) {
            # Create backup if requested
            if ($Backup) {
                $backupPath = $file.FullName + ".bak"
                Copy-Item $file.FullName $backupPath
                Write-Host "  Created backup: $backupPath" -ForegroundColor Blue
            }
            
            # Write the modified content
            Set-Content $file.FullName $content -NoNewline
            Write-Host "  File updated successfully" -ForegroundColor Green
        }
    } else {
        Write-Host "  No changes needed" -ForegroundColor Gray
    }
}

if ($DryRun) {
    Write-Host "`nDRY RUN COMPLETE - No files were modified" -ForegroundColor Yellow
    Write-Host "Run without -DryRun to apply changes" -ForegroundColor Yellow
} else {
    Write-Host "`nConversion complete!" -ForegroundColor Green
    Write-Host "Don't forget to add the logger.go files to your client/ and server/ folders" -ForegroundColor Yellow
}

Write-Host "`nUsage examples:" -ForegroundColor Cyan
Write-Host "  .\convert-logging.ps1 -DryRun          # Preview changes" -ForegroundColor White
Write-Host "  .\convert-logging.ps1 -Backup          # Apply with backups" -ForegroundColor White
Write-Host "  .\convert-logging.ps1                  # Apply changes" -ForegroundColor White