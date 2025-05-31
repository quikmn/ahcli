# Clean and rebuild
$basepath = "C:\Users\quikmn\ahcli"
$buildpath = "$basepath\build"

Remove-Item -Path "$buildpath\*" -Force -Recurse -ErrorAction SilentlyContinue
if (!(Test-Path $buildpath)) { New-Item -ItemType Directory -Path $buildpath | Out-Null }

# Compile
go build -o "$buildpath\client.exe" "$basepath\client"
go build -o "$buildpath\server.exe" "$basepath\server"

# Deploy client.exe if build succeeded
if (Test-Path "$buildpath\client.exe") {
    Remove-Item -Path "$basepath\client\client.exe" -ErrorAction SilentlyContinue
    Copy-Item -Path "$buildpath\client.exe" -Destination "$basepath\client\client.exe"
}

# Deploy server.exe if build succeeded
if (Test-Path "$buildpath\server.exe") {
    Remove-Item -Path "$basepath\server\server.exe" -ErrorAction SilentlyContinue
    Copy-Item -Path "$buildpath\server.exe" -Destination "$basepath\server\server.exe"
}
