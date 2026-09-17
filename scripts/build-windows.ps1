param(
    [string]$Compiler = "gcc",
    [string]$ResourceCompiler = "windres"
)
$ErrorActionPreference = "Stop"
$projectRoot = Split-Path -Parent $PSScriptRoot
Push-Location $projectRoot
try {
    Get-Command go -ErrorAction Stop | Out-Null
    Get-Command $Compiler -ErrorAction Stop | Out-Null
    Get-Command $ResourceCompiler -ErrorAction Stop | Out-Null
    New-Item -ItemType Directory -Force dist | Out-Null
    $env:GOOS = "windows"
    $env:GOARCH = "amd64"
    $env:CGO_ENABLED = "1"
    $env:CC = $Compiler
    & $ResourceCompiler -I scripts -i scripts/windows.rc -o cmd/regexfileextractor/resource_windows_amd64.syso -O coff
    if ($LASTEXITCODE -ne 0) { throw "Windows resource compilation failed" }
    go build -tags 'migrated_fynedo,no_emoji' -trimpath -ldflags '-s -w -H=windowsgui -extldflags=-static' -o dist/RegexFileExtractor.exe ./cmd/regexfileextractor
    if ($LASTEXITCODE -ne 0) { throw "Build failed" }
    $hash = (Get-FileHash dist/RegexFileExtractor.exe -Algorithm SHA256).Hash.ToLowerInvariant()
    "$hash  RegexFileExtractor.exe" | Set-Content -Encoding ascii dist/SHA256SUMS.txt
    Write-Host "Built dist/RegexFileExtractor.exe"
} finally {
    Pop-Location
}
