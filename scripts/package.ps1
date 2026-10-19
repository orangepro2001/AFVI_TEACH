# package.ps1 - build the installer dist\AFVI-Teach-Setup-<version>.exe on native Windows.
# Windows PowerShell 5.1 compatible: no &&, ||, ternary or ?? operators.
#
# The installer is one self-extracting program (cmd\setup, pure Go) with the payload appended by
# cmd\mkinstaller: teach-imgui.exe, camera-gui.exe, camera.exe, the documents and the example
# configuration, optionally the reference data. The GUIs are built here as Windows GUI programs
# (-H windowsgui) with the MinGW runtime linked statically, into dist\bin, so the console builds
# that scripts\build.ps1 puts in bin\ are left alone.
#
#   .\scripts\package.ps1 [-Offline] [-Version V] [-Out FILE] [-Reference DIR]
#
# Requires what build.ps1 requires (scoop mingw, which also brings windres).
[CmdletBinding()]
param(
    [switch]$Offline,
    [string]$Version = '',
    [string]$Out = '',
    [string]$Reference = ''
)

$ErrorActionPreference = 'Stop'

$Root = Split-Path -Parent $PSScriptRoot
Set-Location -LiteralPath $Root

function Invoke-Checked {
    param([string]$What, [scriptblock]$Block)
    & $Block
    if ($LASTEXITCODE -ne 0) {
        Write-Error "$What failed."
        exit 1
    }
}

# --- Toolchain: MinGW-w64 (gcc) for the GUIs, windres for the icons ----------
$mingwBin = Join-Path $env:USERPROFILE 'scoop\apps\mingw\current\bin'
if (-not (Test-Path -LiteralPath $mingwBin)) {
    Write-Error "MinGW-w64 not found at $mingwBin. Install it with: scoop install mingw"
    exit 1
}
$windres = Join-Path $mingwBin 'windres.exe'
if (-not (Test-Path -LiteralPath $windres)) {
    Write-Error 'windres.exe not found: without the manifest in setup.exe Windows asks for administrator rights.'
    exit 1
}
$env:PATH = $mingwBin + ';' + $env:PATH
$env:CGO_ENABLED = '1'
$env:GOOS = 'windows'
$env:GOARCH = 'amd64'

if ($Offline) {
    $env:GOPROXY = 'off'
    $env:GOFLAGS = '-mod=readonly'
}

if ($Version -eq '') {
    $described = & git describe --tags --always --dirty 2>$null
    if ($LASTEXITCODE -eq 0 -and $described) {
        $Version = ($described | Select-Object -First 1).Trim()
    } else {
        $Version = 'dev'
    }
}
# The version ends up in a file name and in the registry: keep it plain.
$Version = [regex]::Replace($Version, '[^A-Za-z0-9._+-]', '-')
if ($Out -eq '') { $Out = Join-Path $Root ("dist\AFVI-Teach-Setup-{0}.exe" -f $Version) }
if ($Reference -ne '' -and -not (Test-Path -LiteralPath $Reference -PathType Container)) {
    Write-Error "-Reference $Reference is not a folder."
    exit 1
}

$distDir = Join-Path $Root 'dist'
$binDir = Join-Path $distDir 'bin'
New-Item -ItemType Directory -Path $binDir -Force | Out-Null

# --- 1. icons and manifest ---------------------------------------------------
Write-Host '==> windres (icons, setup manifest)'
Invoke-Checked 'windres teach-imgui' { & $windres -O coff -o cmd\teach-imgui\rsrc_windows_amd64.syso cmd\teach-imgui\teach.rc }
Invoke-Checked 'windres camera-gui' { & $windres -O coff -o cmd\camera-gui\rsrc_windows_amd64.syso cmd\camera-gui\studio.rc }
Invoke-Checked 'windres setup' { & $windres -O coff -o cmd\setup\rsrc_windows_amd64.syso cmd\setup\setup.rc }

# --- 2. the programs ---------------------------------------------------------
Write-Host '==> go build teach-imgui, camera-gui (static, windowsgui)'
Invoke-Checked 'go build teach-imgui' { & go build -ldflags '-s -w -H windowsgui -extldflags=-static' -o (Join-Path $binDir 'teach-imgui.exe') ./cmd/teach-imgui }
Invoke-Checked 'go build camera-gui' { & go build -ldflags '-s -w -H windowsgui -extldflags=-static' -o (Join-Path $binDir 'camera-gui.exe') ./cmd/camera-gui }

Write-Host '==> go build camera, setup stub (pure Go, no cgo)'
$env:CGO_ENABLED = '0'
Invoke-Checked 'go build camera' { & go build -ldflags '-s -w' -o (Join-Path $binDir 'camera.exe') ./cmd/camera }
$stub = Join-Path $distDir 'setup-stub.exe'
Invoke-Checked 'go build setup' { & go build -ldflags '-s -w -H windowsgui' -o $stub ./cmd/setup }

# --- 3. append the payload (a program for this machine, not for the target) ---
Write-Host '==> mkinstaller'
$mk = @('-root', $Root, '-bin', $binDir, '-stub', $stub, '-out', $Out, '-version', $Version)
if ($Reference -ne '') { $mk += @('-reference', $Reference) }
Invoke-Checked 'mkinstaller' { & go run ./cmd/mkinstaller @mk }
Remove-Item -LiteralPath $stub -Force

Write-Host "OK: $Out"
