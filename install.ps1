$ErrorActionPreference = 'Stop'
$version = "1.0.0"
$repo = "NichuSPN/bandit"
$url = "https://github.com/$repo/releases/download/v$version/bandit-v$version-windows-amd64.zip"
$installDir = "$env:LOCALAPPDATA\bandit"

Write-Host "=======================================================" -ForegroundColor Cyan
Write-Host "   Installing Bandit v$version on Windows..." -ForegroundColor Cyan
Write-Host "=======================================================" -ForegroundColor Cyan

if (!(Test-Path $installDir)) {
    New-Item -ItemType Directory -Force -Path $installDir | Out-Null
}

$zipPath = "$installDir\bandit.zip"
Write-Host "Downloading $url..." -ForegroundColor Yellow
Invoke-WebRequest -Uri $url -OutFile $zipPath

Write-Host "Extracting executable..." -ForegroundColor Yellow
Expand-Archive -Path $zipPath -DestinationPath $installDir -Force
Remove-Item $zipPath

$userPath = [Environment]::GetEnvironmentVariable("Path", "User")
if ($userPath -notlike "*$installDir*") {
    [Environment]::SetEnvironmentVariable("Path", "$userPath;$installDir", "User")
    Write-Host "Added $installDir to PATH" -ForegroundColor Green
}

Write-Host ""
Write-Host "=======================================================" -ForegroundColor Green
Write-Host " ✓ Bandit installed successfully!" -ForegroundColor Green
Write-Host " Please restart your terminal and run 'bandit'." -ForegroundColor Green
Write-Host "=======================================================" -ForegroundColor Green
