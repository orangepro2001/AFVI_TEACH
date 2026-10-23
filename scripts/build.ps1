# build.ps1 - One-click build on native Windows: checks, the three programs, (optionally) the
# self-test, and the installer dist\AFVI-Teach-Setup-<version>.exe (scripts\package.ps1, which builds
# its own windowsgui copies of the programs in dist\bin).
# Windows PowerShell 5.1 compatible: no &&, ||, ternary or ?? operators.
#
#   .\scripts\build.ps1 [-Selftest] [-Out DIR] [-Offline] [-NoInstaller] [-Version V] [-Reference DIR]
#                       [-Commit MESSAGE] [-Push] [-Desktop] [-DesktopDir DIR] [-Fetch]
#
#   -NoInstaller   stop after the three programs and the optional self-test
#   -Version V     version shown by the installer (default: git describe, else "dev")
#   -Reference DIR also ship this reference data tree in the installer (never included otherwise)
#
# Hand-over steps, all off by default (scripts\ship.ps1 does the work; read its header for details):
#   -Commit MESSAGE  once the checks passed and the programs are built, commit the working tree
#                    (ONE commit; nothing is committed when a check fails; an already staged index is refused)
#   -Push            then push the branch to origin (never implied by -Commit)
#   -Desktop         copy the installer to the Windows desktop (sha256 checked)
#   -DesktopDir DIR  copy it to DIR instead (implies -Desktop)
#   -Fetch           wait for the build server's installer of the pushed commit (GitHub Release nightly,
#                    up to 15 minutes) and download it to the desktop / DIR instead of copying the local one
[CmdletBinding()]
param(
    [switch]$Selftest,
    [string]$Out = '',
    [switch]$Offline,
    [switch]$NoInstaller,
    [string]$Version = '',
    [string]$Reference = '',
    [string]$Commit = '',
    [switch]$Push,
    [switch]$Desktop,
    [string]$DesktopDir = '',
    [switch]$Fetch
)

$ErrorActionPreference = 'Stop'

# Always run from the project root (this script lives in scripts/).
$Root = Split-Path -Parent $PSScriptRoot
Set-Location -LiteralPath $Root
# $PSScriptRoot is empty inside param() defaults on Windows PowerShell 5.1, so resolve here.
if ($Out -eq '') { $Out = Join-Path $Root 'selftest-out' }

# --- Toolchain: MinGW-w64 (gcc) is required for cgo. -------------------------
$mingwBin = Join-Path $env:USERPROFILE 'scoop\apps\mingw\current\bin'
if (-not (Test-Path -LiteralPath $mingwBin)) {
    Write-Error "MinGW-w64 not found at $mingwBin. Install it with: scoop install mingw"
    exit 1
}
$env:PATH = $mingwBin + ';' + $env:PATH
$env:CGO_ENABLED = '0'

if ($Offline) {
    $env:GOPROXY = 'off'
    $env:GOFLAGS = '-mod=readonly'
}

# --- 1. gofmt ----------------------------------------------------------------
Write-Host '==> gofmt -l cmd internal'
$unformatted = & gofmt -l cmd internal
if ($LASTEXITCODE -ne 0) {
    Write-Error 'gofmt failed to run.'
    exit 1
}
if ($unformatted) {
    Write-Host 'The following files are not gofmt-clean:'
    $unformatted | ForEach-Object { Write-Host "  $_" }
    exit 1
}

# --- 2. go vet ---------------------------------------------------------------
Write-Host '==> go vet ./...'
& go vet ./...
if ($LASTEXITCODE -ne 0) {
    Write-Error 'go vet failed.'
    exit 1
}

# --- 3. config, core and inspect tests (pure Go, CGO_ENABLED has no effect) --
Write-Host '==> go test ./internal/config/... ./internal/core/... ./internal/inspect/... ./internal/model/... ./internal/snap/... ./internal/camera/... ./internal/link/... ./internal/appicon/... ./internal/installer/... ./internal/access/... ./internal/golden/...'
& go test ./internal/config/... ./internal/core/... ./internal/inspect/... ./internal/model/... ./internal/snap/... ./internal/camera/... ./internal/link/... ./internal/appicon/... ./internal/installer/... ./internal/access/... ./internal/golden/...
if ($LASTEXITCODE -ne 0) {
    Write-Error 'go test ./internal/config/... ./internal/core/... ./internal/inspect/... ./internal/model/... ./internal/snap/... ./internal/camera/... ./internal/link/... ./internal/appicon/... ./internal/installer/... ./internal/access/... ./internal/golden/... failed.'
    exit 1
}

# --- 3b. application icons (Windows resources) -------------------------------
# windres comes with the scoop mingw package; without it the executables get the default icon.
$windres = Join-Path $mingwBin 'windres.exe'
if (Test-Path -LiteralPath $windres) {
    Write-Host '==> windres (application icons)'
    & $windres -O coff -o cmd\teach-imgui\rsrc_windows_amd64.syso cmd\teach-imgui\teach.rc
    if ($LASTEXITCODE -ne 0) {
        Write-Error 'windres failed for teach-imgui.'
        exit 1
    }
    & $windres -O coff -o cmd\camera-gui\rsrc_windows_amd64.syso cmd\camera-gui\studio.rc
    if ($LASTEXITCODE -ne 0) {
        Write-Error 'windres failed for camera-gui.'
        exit 1
    }
} else {
    Write-Host 'WARNING: windres.exe not found, the executables get no icon'
}

# --- 4. build ----------------------------------------------------------------
Write-Host '==> go build'
$binDir = Join-Path $Root 'bin'
if (-not (Test-Path -LiteralPath $binDir)) {
    New-Item -ItemType Directory -Path $binDir -Force | Out-Null
}
& go build -tags production -ldflags "-s -w -H windowsgui" -o bin\teach-imgui.exe ./cmd/teach-imgui
if ($LASTEXITCODE -ne 0) {
    Write-Error 'go build failed.'
    exit 1
}

$exe = Join-Path $Root 'bin\teach-imgui.exe'
$sizeMB = [math]::Round((Get-Item -LiteralPath $exe).Length / 1MB, 2)
Write-Host ("Built {0} ({1} MB)" -f $exe, $sizeMB)

# --- 4b. camera-gui (studio, cgo like the teach program) ---------------------
Write-Host '==> go build camera-gui'
& go build -tags production -ldflags "-s -w -H windowsgui" -o bin\camera-gui.exe ./cmd/camera-gui
if ($LASTEXITCODE -ne 0) {
    Write-Error 'go build camera-gui failed.'
    exit 1
}
$studioExe = Join-Path $Root 'bin\camera-gui.exe'
$studioSizeMB = [math]::Round((Get-Item -LiteralPath $studioExe).Length / 1MB, 2)
Write-Host ("Built {0} ({1} MB)" -f $studioExe, $studioSizeMB)

# --- 4c. camera (pure Go, no cgo) --------------------------------------------
Write-Host '==> go build camera (pure Go, no cgo)'
$env:CGO_ENABLED = '0'
& go build -ldflags "-s -w" -o bin\camera.exe ./cmd/camera
$cameraExit = $LASTEXITCODE
$env:CGO_ENABLED = '0'
if ($cameraExit -ne 0) {
    Write-Error 'go build camera failed.'
    exit 1
}
$cameraExe = Join-Path $Root 'bin\camera.exe'
$cameraSizeMB = [math]::Round((Get-Item -LiteralPath $cameraExe).Length / 1MB, 2)
Write-Host ("Built {0} ({1} MB)" -f $cameraExe, $cameraSizeMB)

# --- 5. optional self-test ---------------------------------------------------
if ($Selftest) {
    Write-Warning 'The ImGui -selftest is gone; use scripts/e2e-wsl.sh (Playwright) instead.'
    $Selftest = $false
}
if ($Selftest) {
    if (-not (Test-Path -LiteralPath $Out)) {
        New-Item -ItemType Directory -Path $Out -Force | Out-Null
    }
    Write-Host "==> selftest -> $Out"
    $raw = & $exe -selftest -out $Out
    if ($LASTEXITCODE -ne 0) {
        Write-Error 'selftest process exited with a non-zero code.'
        exit 1
    }
    $text = ($raw | Out-String)
    $metrics = $null
    try {
        $metrics = $text | ConvertFrom-Json
    } catch {
        Write-Error "Could not parse selftest output as JSON: $text"
        exit 1
    }
    $errCount = 0
    if ($null -ne $metrics.errors) {
        $errCount = @($metrics.errors).Count
    }
    $shotCount = 0
    if ($null -ne $metrics.screenshots) {
        $shotCount = @($metrics.screenshots).Count
    }
    Write-Host ("selftest: {0} errors, {1} screenshots" -f $errCount, $shotCount)
    if ($errCount -gt 0) {
        Write-Error 'selftest reported errors.'
        exit 1
    }
}

# --- 5b. hand-over: commit and push -------------------------------------------
# Only after gofmt, vet, the tests, the builds and the optional self-test above passed ($ErrorActionPreference
# stops the script otherwise), and before the installer, so its version names the new commit instead of "-dirty".
$ship = Join-Path $PSScriptRoot 'ship.ps1'
if ($Commit -ne '') {
    Write-Host '==> commit (scripts\ship.ps1 commit)'
    & $ship commit -Message $Commit
    if ($LASTEXITCODE -ne 0) {
        Write-Error 'commit failed.'
        exit 1
    }
}
if ($Push) {
    Write-Host '==> push (scripts\ship.ps1 push)'
    & $ship push
    if ($LASTEXITCODE -ne 0) {
        Write-Error 'push failed.'
        exit 1
    }
}

# --- 6. installer -------------------------------------------------------------
# package.ps1 builds the programs it ships itself (windowsgui, into dist\bin), so the console
# builds in bin\ that the self-test needs are left alone. It runs only after the checks above passed.
if (-not $NoInstaller) {
    Write-Host '==> installer (scripts\package.ps1)'
    $pkg = @{}
    if ($Offline) { $pkg['Offline'] = $true }
    if ($Version -ne '') { $pkg['Version'] = $Version }
    if ($Reference -ne '') { $pkg['Reference'] = $Reference }
    & (Join-Path $PSScriptRoot 'package.ps1') @pkg
    if ($LASTEXITCODE -ne 0) {
        Write-Error 'package.ps1 failed.'
        exit 1
    }
}

# --- 7. hand-over: installer to the desktop -------------------------------------
$shipArgs = @{}
if ($DesktopDir -ne '') { $shipArgs['Dir'] = $DesktopDir }
if ($Fetch) {
    Write-Host '==> fetch the server build (scripts\ship.ps1 fetch)'
    & $ship fetch -Wait 900 @shipArgs
    if ($LASTEXITCODE -ne 0) {
        Write-Error 'fetch failed.'
        exit 1
    }
} elseif ($Desktop -or $DesktopDir -ne '') {
    if ($NoInstaller) {
        Write-Error '-Desktop needs the installer (drop -NoInstaller) or use -Fetch.'
        exit 1
    }
    Write-Host '==> installer to the desktop (scripts\ship.ps1 desktop)'
    & $ship desktop @shipArgs
    if ($LASTEXITCODE -ne 0) {
        Write-Error 'desktop copy failed.'
        exit 1
    }
}

Write-Host 'OK'
