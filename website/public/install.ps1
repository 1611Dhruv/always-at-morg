# Always at Morg - Windows Installation Script
# Usage: iwr -useb https://always-at-morg.bid/install.ps1 | iex

param(
    [string]$InstallDir = "$env:LOCALAPPDATA\Programs\morg"
)

$ErrorActionPreference = "Stop"

$BASE_URL = "https://always-at-morg.bid/releases"

Write-Host "╔═══════════════════════════════════════╗" -ForegroundColor Green
Write-Host "║    Always at Morg - Installer         ║" -ForegroundColor Green
Write-Host "╚═══════════════════════════════════════╝" -ForegroundColor Green
Write-Host ""

# Detect architecture
$arch = if ([Environment]::Is64BitOperatingSystem) { "amd64" } else { "386" }
$platform = "windows_$arch"

Write-Host "Detected platform: $platform" -ForegroundColor Green
Write-Host "Install directory: $InstallDir" -ForegroundColor Green
Write-Host ""

# Create install directory
if (!(Test-Path $InstallDir)) {
    New-Item -ItemType Directory -Path $InstallDir -Force | Out-Null
}

# Download URL
$binaryName = "always-at-morg-${platform}.exe"
$downloadUrl = "$BASE_URL/$binaryName"
$outputPath = Join-Path $InstallDir "morg.exe"

Write-Host "Downloading from: $downloadUrl" -ForegroundColor Yellow
Write-Host ""

try {
    # Download binary
    Invoke-WebRequest -Uri $downloadUrl -OutFile $outputPath

    Write-Host ""
    Write-Host "Successfully installed Always at Morg!" -ForegroundColor Green
    Write-Host ""

    # Check if install dir is in PATH
    $userPath = [Environment]::GetEnvironmentVariable("Path", "User")
    if ($userPath -notlike "*$InstallDir*") {
        Write-Host "Note: $InstallDir is not in your PATH" -ForegroundColor Yellow
        Write-Host "Adding to PATH..." -ForegroundColor Yellow

        # Add to user PATH
        $newPath = "$userPath;$InstallDir"
        [Environment]::SetEnvironmentVariable("Path", $newPath, "User")

        Write-Host "Added to PATH! Please restart your terminal." -ForegroundColor Green
        Write-Host ""
        Write-Host "After restarting, run: morg" -ForegroundColor Cyan
    } else {
        Write-Host "Run the game with: morg" -ForegroundColor Cyan
    }

    Write-Host ""
} catch {
    Write-Host "Error: Failed to download binary" -ForegroundColor Red
    Write-Host $_.Exception.Message -ForegroundColor Red
    exit 1
}
