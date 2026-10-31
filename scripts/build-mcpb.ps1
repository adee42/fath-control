# Builds the Windows MCP server and packages it, with extension/manifest.json,
# into bin/fath-control-<version>.mcpb (a zip with manifest.json at its root),
# plus bin/SHA256SUMS. Run from anywhere: pwsh scripts/build-mcpb.ps1
$ErrorActionPreference = 'Stop'
Set-StrictMode -Version Latest

Add-Type -AssemblyName System.IO.Compression
Add-Type -AssemblyName System.IO.Compression.FileSystem

$root = Split-Path -Parent $PSScriptRoot
$manifestPath = Join-Path $root 'extension\manifest.json'
$version = (Get-Content $manifestPath -Raw | ConvertFrom-Json).version

$dist = Join-Path $root 'bin'
$stage = Join-Path $dist 'stage'
if (Test-Path $dist) { Remove-Item $dist -Recurse -Force }
New-Item -ItemType Directory -Path (Join-Path $stage 'server') | Out-Null

# -trimpath drops local build paths from the binary; -buildvcs=false drops VCS
# metadata; -s -w strips symbol and debug tables.
$env:GOOS = 'windows'; $env:GOARCH = 'amd64'; $env:CGO_ENABLED = '0'
Push-Location $root
try {
    go build -trimpath -buildvcs=false -ldflags '-s -w' -o (Join-Path $stage 'server\fath-mcp-server.exe') ./cmd/fath-mcp-server
    if ($LASTEXITCODE -ne 0) { throw 'go build failed' }
} finally {
    Pop-Location
    Remove-Item Env:GOOS, Env:GOARCH, Env:CGO_ENABLED -ErrorAction SilentlyContinue
}
Copy-Item $manifestPath (Join-Path $stage 'manifest.json')

# Zip with forward-slash entry names (Compress-Archive writes backslashes).
$mcpb = Join-Path $dist "fath-control-$version.mcpb"
$zip = [System.IO.Compression.ZipFile]::Open($mcpb, [System.IO.Compression.ZipArchiveMode]::Create)
try {
    foreach ($f in 'manifest.json', 'server/fath-mcp-server.exe') {
        $entry = $zip.CreateEntry($f, [System.IO.Compression.CompressionLevel]::Optimal)
        $out = $entry.Open()
        $in = [System.IO.File]::OpenRead((Join-Path $stage ($f -replace '/', '\')))
        try { $in.CopyTo($out) } finally { $in.Dispose(); $out.Dispose() }
    }
} finally {
    $zip.Dispose()
}
Remove-Item $stage -Recurse -Force

$hash = (Get-FileHash $mcpb -Algorithm SHA256).Hash.ToLower()
"$hash  $(Split-Path -Leaf $mcpb)" | Set-Content (Join-Path $dist 'SHA256SUMS') -Encoding ascii
Write-Host "Built $mcpb"
Write-Host "SHA256 $hash"
