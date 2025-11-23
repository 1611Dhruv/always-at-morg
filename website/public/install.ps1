# Always at Morg - Windows Installer
$ErrorActionPreference = 'Stop'

Write-Host "Installing Always at Morg..." -ForegroundColor Green

$BINARY = "always-at-morg-windows_amd64.exe"
$URL = "https://web.always-at-morg.bid/releases/$BINARY"
$INSTALL_DIR = "$env:USERPROFILE\.local\bin"
$INSTALL_PATH = "$INSTALL_DIR\morg.exe"

# Create install directory
New-Item -ItemType Directory -Force -Path $INSTALL_DIR | Out-Null

# Download
Write-Host "Downloading from $URL..."
Invoke-WebRequest -Uri $URL -OutFile $INSTALL_PATH

Write-Host "✓ Installed to $INSTALL_PATH" -ForegroundColor Green
Write-Host ""
Write-Host "Run with: morg" -ForegroundColor Yellow
