$ErrorActionPreference = "Stop"

Write-Host "======================================================="
Write-Host "   Bandit 2-Step Hybrid Build (Go + Rust)"
Write-Host "   Target: Single Executable Binary (.\bandit.exe)"
Write-Host "======================================================="
Write-Host ""

Write-Host "[1/2] Compiling Rust core engine static library (bandit_engine)..."
Set-Location "$PSScriptRoot\bandit_engine"
cargo build --release
Set-Location "$PSScriptRoot"

Write-Host ""
Write-Host "[2/2] Statically linking Go CLI binary with CGO..."
$env:CGO_ENABLED="1"
go build -o bandit.exe main.go

Write-Host ""
Write-Host "[3/3] Installing binary globally..."
$gopath = go env GOPATH
$gopathBin = Join-Path $gopath "bin"
if (!(Test-Path $gopathBin)) { New-Item -ItemType Directory -Path $gopathBin }
Copy-Item .\bandit.exe (Join-Path $gopathBin "bandit.exe") -Force
$cargoBin = Join-Path $env:USERPROFILE ".cargo\bin\bandit.exe"
if (Test-Path $cargoBin) {
    Copy-Item .\bandit.exe $cargoBin -Force
    Write-Host " Updated $cargoBin"
}
Write-Host " Updated $(Join-Path $gopathBin 'bandit.exe')"

Write-Host ""
Write-Host "======================================================="
Write-Host " ✓ Build Successful! Single binary created: .\bandit.exe"
Write-Host "======================================================="
