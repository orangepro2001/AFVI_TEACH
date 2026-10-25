# fetch-latest.ps1 - copy the newest AFVI installer from the build server to this PC's desktop.
# Windows PowerShell 5.1 compatible (no &&, ||, ternary or ?? operators); needs the OpenSSH client
# (ssh.exe, scp.exe: built into Windows 10/11) and key-based login to the server.
#
#   .\scripts\fetch-latest.ps1 [-SshHost AI] [-RemoteDir '~/afvi-latest'] [-Dest DIR] [-Force] [-Quiet]
#
# The server side is scripts/publish-latest.sh (build-wsl.sh --publish-latest): it keeps LATEST.txt
# (the newest installer's name) and SHA256SUMS.txt in the remote folder. This script reads both over
# ssh, does nothing when the newest installer was already fetched, otherwise downloads it with scp,
# verifies the sha256 and puts it on the desktop (the real one, also when OneDrive redirects it).
# The last fetched name is remembered in %LOCALAPPDATA%\AFVI\fetch-state.txt, every run is logged to
# %LOCALAPPDATA%\AFVI\fetch-latest.log. Exit code: 0 ok / nothing new, 1 failure.
# scripts\register-fetch-task.ps1 runs this on a schedule.
[CmdletBinding()]
param(
    [string]$SshHost = 'AI',
    [string]$RemoteDir = '~/afvi-latest',
    [string]$Dest = '',
    [switch]$Force,
    [switch]$Quiet
)

$ErrorActionPreference = 'Stop'
$stateDir = Join-Path $env:LOCALAPPDATA 'AFVI'
if (-not (Test-Path -LiteralPath $stateDir)) {
    New-Item -ItemType Directory -Path $stateDir -Force | Out-Null
}
$logFile = Join-Path $stateDir 'fetch-latest.log'
$stateFile = Join-Path $stateDir 'fetch-state.txt'

function Write-Log {
    param([string]$Text)
    $line = '{0}  {1}' -f (Get-Date -Format 'yyyy-MM-dd HH:mm:ss'), $Text
    Add-Content -LiteralPath $logFile -Value $line
    if (-not $Quiet) {
        Write-Host $Text
    }
}

# keep the log small
if ((Test-Path -LiteralPath $logFile) -and ((Get-Item -LiteralPath $logFile).Length -gt 200KB)) {
    Get-Content -LiteralPath $logFile -Tail 200 | Set-Content -LiteralPath $logFile
}

function Invoke-Ssh {
    param([string]$Command)
    # -n: stdin from NUL, so ssh can never wait for console input. A run started from the scheduled
    # task (wscript.exe -> powershell.exe, no console at all) would otherwise stall on a prompt.
    $out = & ssh -n -o BatchMode=yes -o ConnectTimeout=10 -o StrictHostKeyChecking=accept-new $SshHost $Command 2>&1
    if ($LASTEXITCODE -ne 0) {
        throw ("ssh {0} failed (exit {1}): {2}" -f $SshHost, $LASTEXITCODE, ($out -join ' '))
    }
    return $out
}

try {
    foreach ($tool in @('ssh', 'scp')) {
        if (-not (Get-Command $tool -ErrorAction SilentlyContinue)) {
            throw "$tool.exe not found: enable the 'OpenSSH Client' optional feature of Windows."
        }
    }

    if ($Dest -eq '') {
        $Dest = [Environment]::GetFolderPath('Desktop')
    }
    if (-not (Test-Path -LiteralPath $Dest)) {
        throw "destination folder not found: $Dest"
    }

    # 1. what is the newest installer on the server, and what is its sha256
    $name = ((Invoke-Ssh "cat $RemoteDir/LATEST.txt") -join '').Trim()
    if ($name -notmatch '^AFVI-Teach-Setup-[A-Za-z0-9._+-]+\.exe$') {
        throw "unexpected installer name from the server: '$name'"
    }
    $sums = Invoke-Ssh "cat $RemoteDir/SHA256SUMS.txt"
    $expected = ''
    foreach ($line in $sums) {
        $parts = ($line -as [string]).Trim() -split '\s+', 2
        if ($parts.Count -eq 2 -and $parts[1].TrimStart('*') -eq $name) {
            $expected = $parts[0].ToLower()
        }
    }
    if ($expected -eq '') {
        throw "no sha256 for $name in SHA256SUMS.txt"
    }
    $key = "$name $expected"

    # 2. already fetched?
    $target = Join-Path $Dest $name
    $known = ''
    if (Test-Path -LiteralPath $stateFile) {
        $known = (Get-Content -LiteralPath $stateFile -Raw).Trim()
    }
    if ((-not $Force) -and ($known -eq $key)) {
        Write-Log "up to date: $name"
        exit 0
    }

    # 3. download to a temp file, verify, then move onto the desktop
    $tmp = Join-Path $stateDir ($name + '.part')
    Write-Log "downloading $name from ${SshHost}:$RemoteDir ..."
    # -B batch mode: no passphrase/password prompts, so nothing ever waits for console input
    & scp -B -o BatchMode=yes -o ConnectTimeout=10 "${SshHost}:$RemoteDir/$name" $tmp 2>&1 | Out-Null
    if ($LASTEXITCODE -ne 0) {
        throw "scp failed (exit $LASTEXITCODE)"
    }
    $actual = (Get-FileHash -LiteralPath $tmp -Algorithm SHA256).Hash.ToLower()
    if ($actual -ne $expected) {
        Remove-Item -LiteralPath $tmp -Force -ErrorAction SilentlyContinue
        throw "sha256 mismatch for $name (expected $expected, got $actual)"
    }
    Move-Item -LiteralPath $tmp -Destination $target -Force
    Set-Content -LiteralPath $stateFile -Value $key
    Write-Log "OK: $target (sha256 $($expected.Substring(0, 12))...)"

    # a short notification when run by the scheduled task (best effort, never fails the run)
    if ($Quiet) {
        try {
            Add-Type -AssemblyName System.Windows.Forms
            $icon = New-Object System.Windows.Forms.NotifyIcon
            $icon.Icon = [System.Drawing.SystemIcons]::Information
            $icon.BalloonTipTitle = 'AFVI installer'
            $icon.BalloonTipText = "New installer on the desktop: $name"
            $icon.Visible = $true
            $icon.ShowBalloonTip(5000)
            Start-Sleep -Seconds 6
            $icon.Dispose()
        } catch {
            # no balloon, no problem
        }
    }
    exit 0
} catch {
    Write-Log ("FAILED: " + $_.Exception.Message)
    exit 1
}
