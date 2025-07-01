# Delta CLI PowerShell Build Script
# Equivalent to the Makefile for Windows users

param(
    [Parameter(Position=0)]
    [string]$Target = "build",
    
    [Parameter()]
    [string]$Platform = "",
    
    [Parameter()]
    [string]$ReleaseVersion = "",
    
    [Parameter()]
    [switch]$Clean,
    
    [Parameter()]
    [switch]$Help
)

# Script configuration
$ErrorActionPreference = "Stop"
$BinaryName = "delta"
$BuildDir = "build"
$GoFlags = "-v"

# Version information
$Version = & {
    $tag = git describe --tags --abbrev=0 2>$null
    if ($LASTEXITCODE -eq 0 -and $tag) { $tag } else { "v0.4.5-alpha" }
}

$GitCommit = & {
    $commit = git rev-parse HEAD 2>$null
    if ($LASTEXITCODE -eq 0 -and $commit) { $commit } else { "unknown" }
}

$BuildDate = (Get-Date -Format "yyyy-MM-dd_HH:mm:ss_UTC" -AsUTC)

$IsDirty = & {
    git diff --quiet 2>$null
    if ($LASTEXITCODE -eq 1) { "true" } else { "false" }
}

# LDFLAGS for version injection
$LDFlags = "-X main.Version=$Version " +
           "-X main.GitCommit=$GitCommit " +
           "-X main.BuildDate=$BuildDate " +
           "-X main.IsDirty=$IsDirty"

# SQLite vector extension configuration
$SqliteVecVersion = "v0.1.6"
$SqliteVecUrl = "https://github.com/asg017/sqlite-vec/releases/download/$SqliteVecVersion/sqlite-vec-0.1.6-loadable-windows-x86_64.zip"

# Go source files
$GoSources = @(
    "version.go",
    "ai.go", "ai_manager.go", "ai_health_monitor.go",
    "art2_manager.go", "art2_preprocessor.go", "art2_commands.go",
    "jump_manager.go", "jump_helper.go",
    "cli.go", "help.go",
    "i18n_manager.go", "i18n_commands.go", "i18n_github_loader.go",
    "memory_manager.go", "memory_commands.go",
    "tokenizer.go", "tokenizer_commands.go",
    "inference.go", "inference_commands.go",
    "vector_db.go", "vector_commands.go",
    "onnx_runtime.go", "onnx_runtime_test.go",
    "embedding_manager.go", "embedding_commands.go",
    "speculative_decoding.go", "speculative_commands.go",
    "knowledge_extractor.go", "knowledge_commands.go", "knowledge_extractor_agent_command.go",
    "agent_types.go", "agent_manager.go", "agent_commands.go",
    "config_manager.go", "config_commands.go",
    "version_manager.go", "update_manager.go", "update_commands.go", "github_client.go", 
    "update_checker.go", "update_downloader.go", "update_installer.go", "update_ui.go", 
    "update_scheduler.go", "update_history.go", "update_validation.go",
    "spellcheck.go", "spellcheck_commands.go",
    "history_analysis.go", "history_commands.go",
    "pattern_update.go", "pattern_commands.go", "pattern_recognition.go",
    "error_learning.go",
    "suggest_command.go", "suggest_commands.go",
    "validation_commands.go", "command_validator.go"
) -join " "

# Supported platforms
$Platforms = @{
    "linux/amd64" = @{OS="linux"; Arch="amd64"; CGO="1"}
    "darwin/amd64" = @{OS="darwin"; Arch="amd64"; CGO="0"}
    "darwin/arm64" = @{OS="darwin"; Arch="arm64"; CGO="0"}
    "windows/amd64" = @{OS="windows"; Arch="amd64"; CGO="1"}
}

function Show-Help {
    Write-Host "Delta CLI Build Script"
    Write-Host ""
    Write-Host "Usage: .\build.ps1 [Target] [Options]"
    Write-Host ""
    Write-Host "Targets:"
    Write-Host "  build         Build Delta for current platform (default)"
    Write-Host "  build-all     Build Delta for all supported platforms"
    Write-Host "  clean         Clean build artifacts and dependencies"
    Write-Host "  deps          Download dependencies (SQLite vector extension)"
    Write-Host "  run           Build and run Delta"
    Write-Host "  install       Build and install Delta to system"
    Write-Host "  version-info  Show version information"
    Write-Host "  release       Create a new release"
    Write-Host "  installer     Build Windows installer"
    Write-Host "  man           Generate man pages"
    Write-Host "  install-man   Install man pages (Unix-like systems)"
    Write-Host "  preview-man   Preview man pages"
    Write-Host "  completions   Generate shell completions"
    Write-Host ""
    Write-Host "Options:"
    Write-Host "  -Platform     Specify target platform (e.g., 'linux/amd64')"
    Write-Host "  -ReleaseVersion  Version for release (e.g., 'v0.4.6-alpha')"
    Write-Host "  -Clean        Clean before building"
    Write-Host "  -Help         Show this help message"
    Write-Host ""
    Write-Host "Examples:"
    Write-Host "  .\build.ps1                    # Build for current platform"
    Write-Host "  .\build.ps1 build-all          # Build for all platforms"
    Write-Host "  .\build.ps1 build -Platform linux/amd64"
    Write-Host "  .\build.ps1 release -ReleaseVersion v0.4.6-alpha"
    Write-Host "  .\build.ps1 man                # Generate man pages"
}

function Get-Dependencies {
    Write-Host "Checking for SQLite vector extension..."
    
    if (-not (Test-Path "vec0.dll")) {
        Write-Host "Downloading SQLite vector extension $SqliteVecVersion..."
        
        try {
            # Download the file
            Invoke-WebRequest -Uri $SqliteVecUrl -OutFile "sqlite-vec.zip"
            
            # Extract the archive
            Expand-Archive -Path "sqlite-vec.zip" -DestinationPath "." -Force
            
            # Clean up
            Remove-Item "sqlite-vec.zip"
            
            Write-Host "SQLite vector extension downloaded successfully" -ForegroundColor Green
        }
        catch {
            Write-Host "Error downloading SQLite vector extension: $_" -ForegroundColor Red
            exit 1
        }
    }
    else {
        Write-Host "SQLite vector extension already exists locally" -ForegroundColor Green
    }
}

function Build-Delta {
    param(
        [string]$TargetOS = $env:GOOS,
        [string]$TargetArch = $env:GOARCH,
        [string]$OutputPath = ""
    )
    
    # Default to current platform if not specified
    if (-not $TargetOS) { $TargetOS = if ($IsWindows) { "windows" } elseif ($IsMacOS) { "darwin" } else { "linux" } }
    if (-not $TargetArch) { $TargetArch = "amd64" }
    
    $platform = "$TargetOS/$TargetArch"
    Write-Host "Building $BinaryName for $platform"
    Write-Host "Version: $Version, Commit: $GitCommit, Date: $BuildDate, Dirty: $IsDirty"
    
    # Determine output path
    if (-not $OutputPath) {
        $OutputPath = Join-Path $BuildDir $platform
    }
    
    # Create output directory
    New-Item -ItemType Directory -Force -Path $OutputPath | Out-Null
    
    # Determine binary name (add .exe for Windows)
    $outputBinary = Join-Path $OutputPath $BinaryName
    if ($TargetOS -eq "windows") {
        $outputBinary += ".exe"
    }
    
    # Set environment variables for cross-compilation
    $env:GOOS = $TargetOS
    $env:GOARCH = $TargetArch
    
    # Determine CGO setting
    if ($Platforms.ContainsKey($platform)) {
        $env:CGO_ENABLED = $Platforms[$platform].CGO
    } else {
        $env:CGO_ENABLED = if ($TargetOS -eq "linux" -or $TargetOS -eq "windows") { "1" } else { "0" }
    }
    
    # Build command
    $buildCmd = "go build $GoFlags -ldflags `"$LDFlags`" -o `"$outputBinary`" $GoSources"
    
    Write-Host "Executing: $buildCmd"
    Invoke-Expression $buildCmd
    
    if ($LASTEXITCODE -eq 0) {
        Write-Host "Successfully built $BinaryName for $platform" -ForegroundColor Green
    } else {
        Write-Host "Build failed for $platform" -ForegroundColor Red
        exit 1
    }
    
    # Clean up environment variables
    Remove-Item Env:\GOOS -ErrorAction SilentlyContinue
    Remove-Item Env:\GOARCH -ErrorAction SilentlyContinue
    Remove-Item Env:\CGO_ENABLED -ErrorAction SilentlyContinue
}

function Build-AllPlatforms {
    Write-Host "Building Delta for all supported platforms..."
    
    foreach ($platform in $Platforms.Keys) {
        $config = $Platforms[$platform]
        Build-Delta -TargetOS $config.OS -TargetArch $config.Arch
    }
}

function Clean-Build {
    Write-Host "Cleaning up build directory and dependencies..."
    
    if (Test-Path $BuildDir) {
        Remove-Item -Recurse -Force $BuildDir
        Write-Host "Removed build directory" -ForegroundColor Green
    }
    
    if (Test-Path "vec0.dll") {
        Remove-Item -Force "vec0.dll"
        Write-Host "Removed SQLite vector extension" -ForegroundColor Green
    }
    
    if (Test-Path "vec0.so") {
        Remove-Item -Force "vec0.so"
        Write-Host "Removed SQLite vector extension (Linux)" -ForegroundColor Green
    }
}

function Run-Delta {
    Build-Delta
    
    $binary = Join-Path $BuildDir "windows/amd64" "$BinaryName.exe"
    if (-not (Test-Path $binary)) {
        # Try current directory
        $binary = ".\$BinaryName.exe"
    }
    
    Write-Host "Running $BinaryName..."
    & $binary
}

function Install-Delta {
    Build-Delta
    
    $source = Join-Path $BuildDir "windows/amd64" "$BinaryName.exe"
    $destination = "C:\Program Files\Delta\$BinaryName.exe"
    
    Write-Host "Installing $BinaryName to $destination"
    
    # Create directory if it doesn't exist
    $destDir = Split-Path $destination -Parent
    if (-not (Test-Path $destDir)) {
        New-Item -ItemType Directory -Force -Path $destDir | Out-Null
    }
    
    # Copy file (requires admin privileges)
    try {
        Copy-Item -Path $source -Destination $destination -Force
        Write-Host "Delta installed successfully" -ForegroundColor Green
        Write-Host "Add 'C:\Program Files\Delta' to your PATH to use Delta from anywhere"
    }
    catch {
        Write-Host "Installation failed. Try running as Administrator." -ForegroundColor Red
        exit 1
    }
}

function Show-VersionInfo {
    Write-Host "Version Information:"
    Write-Host "  VERSION: $Version"
    Write-Host "  GIT_COMMIT: $GitCommit"
    Write-Host "  BUILD_DATE: $BuildDate"
    Write-Host "  IS_DIRTY: $IsDirty"
    Write-Host "  LDFLAGS: $LDFlags"
}

function Create-Release {
    if (-not $ReleaseVersion) {
        Write-Host "Error: ReleaseVersion not specified" -ForegroundColor Red
        Write-Host "Usage: .\build.ps1 release -ReleaseVersion v0.4.6-alpha"
        exit 1
    }
    
    Write-Host "Creating release for version $ReleaseVersion"
    Write-Host "Step 1: Creating release notes..."
    
    # Create release notes directory
    New-Item -ItemType Directory -Force -Path "RELEASE_NOTES" | Out-Null
    
    # Create release notes template
    $releaseNotesPath = "RELEASE_NOTES\RELEASE_NOTES_$ReleaseVersion.md"
    @"
# Release Notes for $ReleaseVersion

## 🚀 Highlights

TODO: Add release highlights here

## 📦 What's New

TODO: Add new features here

"@ | Set-Content $releaseNotesPath
    
    Write-Host "Release notes template created at: $releaseNotesPath" -ForegroundColor Green
    Write-Host "Please edit the release notes before continuing..."
    Write-Host ""
    Write-Host "When ready, run the following commands:"
    Write-Host "  1. git add $releaseNotesPath"
    Write-Host "  2. git commit -m 'feat: prepare $ReleaseVersion release'"
    Write-Host "  3. git tag -a $ReleaseVersion -m 'Release $ReleaseVersion'"
    Write-Host "  4. git push origin main"
    Write-Host "  5. git push origin $ReleaseVersion"
    Write-Host "  6. .\scripts\create-release.sh $ReleaseVersion"
}

function Build-Installer {
    Write-Host "Building Windows installer for Delta v$Version"
    
    # First build Windows binary
    Build-Delta -TargetOS "windows" -TargetArch "amd64"
    
    # Check for Inno Setup
    $innoSetupPaths = @(
        "C:\Program Files (x86)\Inno Setup 6\ISCC.exe",
        "C:\Program Files\Inno Setup 6\ISCC.exe",
        "${env:ProgramFiles(x86)}\Inno Setup 6\ISCC.exe",
        "${env:ProgramFiles}\Inno Setup 6\ISCC.exe"
    )
    
    $innoSetup = $null
    foreach ($path in $innoSetupPaths) {
        if (Test-Path $path) {
            $innoSetup = $path
            break
        }
    }
    
    if (-not $innoSetup) {
        Write-Host "Error: Inno Setup not found" -ForegroundColor Red
        Write-Host "Please install Inno Setup from: https://jrsoftware.org/isdl.php"
        exit 1
    }
    
    # Check for installer configuration
    $installerConfig = "installer\delta-installer.iss"
    if (-not (Test-Path $installerConfig)) {
        Write-Host "Error: Installer configuration not found at $installerConfig" -ForegroundColor Red
        exit 1
    }
    
    # Build installer
    Write-Host "Building installer using Inno Setup..."
    & $innoSetup $installerConfig
    
    if ($LASTEXITCODE -eq 0) {
        Write-Host "Installer created successfully: build\installer\delta-setup-$Version.exe" -ForegroundColor Green
    } else {
        Write-Host "Installer build failed" -ForegroundColor Red
        exit 1
    }
}

function Build-ManPages {
    Write-Host "Generating man pages..."
    
    # First ensure Delta is built
    Build-Delta
    
    # Create man directory
    New-Item -ItemType Directory -Force -Path "man" | Out-Null
    
    # Generate man pages
    $binary = Join-Path $BuildDir "windows/amd64" "$BinaryName.exe"
    if (-not (Test-Path $binary)) {
        $binary = ".\$BinaryName.exe"
    }
    
    & $binary :man generate man/
    
    if ($LASTEXITCODE -eq 0) {
        Write-Host "Man pages generated in ./man/" -ForegroundColor Green
        
        # List generated files
        $manFiles = Get-ChildItem -Path "man" -Filter "*.1"
        if ($manFiles) {
            Write-Host "`nGenerated man pages:"
            foreach ($file in $manFiles) {
                Write-Host "  - $($file.Name)"
            }
        }
        
        Write-Host "`nNote: Man pages are primarily for Unix-like systems."
        Write-Host "On Windows, use 'delta :help' for command documentation."
    } else {
        Write-Host "Failed to generate man pages" -ForegroundColor Red
        exit 1
    }
}

function Install-ManPages {
    Write-Host "Man page installation is not supported on Windows" -ForegroundColor Yellow
    Write-Host "Man pages are for Unix-like systems (Linux, macOS, etc.)"
    Write-Host ""
    Write-Host "On Windows, you can:"
    Write-Host "1. Use 'delta :help' for built-in help"
    Write-Host "2. Generate man pages with '.\build.ps1 man' and copy to a Unix system"
    Write-Host "3. Use WSL (Windows Subsystem for Linux) to install man pages"
}

function Preview-ManPage {
    Write-Host "Previewing man pages..."
    
    # First ensure Delta is built
    Build-Delta
    
    $binary = Join-Path $BuildDir "windows/amd64" "$BinaryName.exe"
    if (-not (Test-Path $binary)) {
        $binary = ".\$BinaryName.exe"
    }
    
    & $binary :man preview
    
    if ($LASTEXITCODE -ne 0) {
        Write-Host "Failed to preview man page" -ForegroundColor Red
        exit 1
    }
}

function Generate-Completions {
    Write-Host "Generating shell completions..."
    
    # First ensure Delta is built
    Build-Delta
    
    # Create completions directory
    New-Item -ItemType Directory -Force -Path "completions" | Out-Null
    
    $binary = Join-Path $BuildDir "windows/amd64" "$BinaryName.exe"
    if (-not (Test-Path $binary)) {
        $binary = ".\$BinaryName.exe"
    }
    
    # Generate bash completions
    & $binary :man completions bash | Out-File -FilePath "completions\delta.bash" -Encoding UTF8
    Write-Host "Bash completions saved to completions\delta.bash" -ForegroundColor Green
    
    # Generate PowerShell completions (if supported)
    # Note: This would require implementing PowerShell completion generation
    Write-Host ""
    Write-Host "Note: PowerShell completions are not yet implemented."
    Write-Host "For PowerShell, use tab completion with the built-in features."
}

# Main script logic
if ($Help) {
    Show-Help
    exit 0
}

if ($Clean) {
    Clean-Build
    exit 0
}

switch ($Target.ToLower()) {
    "build" {
        Get-Dependencies
        if ($Platform) {
            $parts = $Platform.Split('/')
            if ($parts.Count -eq 2) {
                Build-Delta -TargetOS $parts[0] -TargetArch $parts[1]
            } else {
                Write-Host "Invalid platform format. Use OS/ARCH (e.g., linux/amd64)" -ForegroundColor Red
                exit 1
            }
        } else {
            Build-Delta
        }
    }
    "build-all" {
        Get-Dependencies
        Build-AllPlatforms
    }
    "clean" {
        Clean-Build
    }
    "deps" {
        Get-Dependencies
    }
    "run" {
        Get-Dependencies
        Run-Delta
    }
    "install" {
        Get-Dependencies
        Install-Delta
    }
    "version-info" {
        Show-VersionInfo
    }
    "release" {
        Create-Release
    }
    "installer" {
        Build-Installer
    }
    "man" {
        Build-ManPages
    }
    "install-man" {
        Install-ManPages
    }
    "preview-man" {
        Preview-ManPage
    }
    "completions" {
        Generate-Completions
    }
    default {
        Write-Host "Unknown target: $Target" -ForegroundColor Red
        Write-Host "Run '.\build.ps1 -Help' for usage information"
        exit 1
    }
}