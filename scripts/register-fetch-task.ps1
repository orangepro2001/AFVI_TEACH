# register-fetch-task.ps1 - run scripts\fetch-latest.ps1 automatically with a Windows scheduled task.
# Windows PowerShell 5.1 compatible. Per-user task: no administrator rights needed; it runs when you
# are logged on (so your ssh key and desktop are available), at logon and then every N minutes.
#
#   .\scripts\register-fetch-task.ps1 [-IntervalMinutes 10] [-SshHost AI] [-RemoteDir '~/afvi-latest'] [-Dest DIR]
#   .\scripts\register-fetch-task.ps1 -Unregister
#
# No black window: a task whose action is powershell.exe gets its console window from Windows before
# PowerShell can apply -WindowStyle Hidden, so it flashed a console at every interval. The action is
# wscript.exe instead - a GUI program that never owns a console - running
# %LOCALAPPDATA%\AFVI\fetch-latest-hidden.vbs, which starts the same PowerShell command with a hidden
# window and waits for it, so the task's duration and last-run result stay the fetch's own.
#
# After registering, the task is started once at once. Look at it with Task Scheduler (task name
# "AFVI fetch latest installer") or read %LOCALAPPDATA%\AFVI\fetch-latest.log.
[CmdletBinding()]
param(
    [int]$IntervalMinutes = 10,
    [string]$SshHost = 'AI',
    [string]$RemoteDir = '~/afvi-latest',
    [string]$Dest = '',
    [switch]$Unregister
)

$ErrorActionPreference = 'Stop'
$taskName = 'AFVI fetch latest installer'
$installDir = Join-Path $env:LOCALAPPDATA 'AFVI'
$installed = Join-Path $installDir 'fetch-latest.ps1'
$launcherSource = Join-Path $PSScriptRoot 'fetch-latest-hidden.vbs'
$launcher = Join-Path $installDir 'fetch-latest-hidden.vbs'

if ($Unregister) {
    if (Get-ScheduledTask -TaskName $taskName -ErrorAction SilentlyContinue) {
        # stop a run in flight, otherwise the task cannot be replaced cleanly
        Stop-ScheduledTask -TaskName $taskName -ErrorAction SilentlyContinue
        Unregister-ScheduledTask -TaskName $taskName -Confirm:$false
        Write-Host "removed task '$taskName'"
    } else {
        Write-Host "task '$taskName' is not registered"
    }
    if (Test-Path -LiteralPath $launcher) {
        Remove-Item -LiteralPath $launcher -Force
    }
    exit 0
}

if ($IntervalMinutes -lt 1) {
    throw 'IntervalMinutes must be at least 1'
}
$script = Join-Path $PSScriptRoot 'fetch-latest.ps1'
if (-not (Test-Path -LiteralPath $script)) {
    throw "fetch-latest.ps1 not found next to this script: $script"
}
if (-not (Test-Path -LiteralPath $launcherSource)) {
    throw "fetch-latest-hidden.vbs not found next to this script: $launcherSource"
}

# Stable copies outside the repository, so the task keeps working when the repo moves or is
# switched to another branch.
if (-not (Test-Path -LiteralPath $installDir)) {
    New-Item -ItemType Directory -Path $installDir -Force | Out-Null
}
Copy-Item -LiteralPath $script -Destination $installed -Force
Copy-Item -LiteralPath $launcherSource -Destination $launcher -Force

# Arguments for fetch-latest.ps1: the script path first, then the switches.
$fetchArgs = @($installed, '-Quiet', '-SshHost', $SshHost, '-RemoteDir', $RemoteDir)
if ($Dest -ne '') {
    $fetchArgs += @('-Dest', $Dest)
}

$wscript = Join-Path $env:SystemRoot 'System32\wscript.exe'
if (Test-Path -LiteralPath $wscript) {
    # //B: no script-error dialogs, //nologo: no banner
    $launcherArgs = @($launcher) + $fetchArgs
    $quotedArgs = ($launcherArgs | ForEach-Object { '"{0}"' -f $_ }) -join ' '
    $action = New-ScheduledTaskAction -Execute $wscript -Argument ('//nologo //B ' + $quotedArgs)
} else {
    Write-Warning 'wscript.exe not found: falling back to powershell.exe, a console window will flash.'
    $psArgs = @('-NoProfile', '-NonInteractive', '-WindowStyle', 'Hidden', '-ExecutionPolicy', 'Bypass', '-File') + $fetchArgs
    $quotedArgs = ($psArgs | ForEach-Object { '"{0}"' -f $_ }) -join ' '
    $action = New-ScheduledTaskAction -Execute 'powershell.exe' -Argument $quotedArgs
}

$start = (Get-Date).AddMinutes(1)
$repeat = New-ScheduledTaskTrigger -Once -At $start -RepetitionInterval (New-TimeSpan -Minutes $IntervalMinutes) -RepetitionDuration (New-TimeSpan -Days 3650)
$logon = New-ScheduledTaskTrigger -AtLogOn -User "$env:USERDOMAIN\$env:USERNAME"
$settings = New-ScheduledTaskSettingsSet -StartWhenAvailable -AllowStartIfOnBatteries -DontStopIfGoingOnBatteries -MultipleInstances IgnoreNew -ExecutionTimeLimit (New-TimeSpan -Minutes 10)
$principal = New-ScheduledTaskPrincipal -UserId "$env:USERDOMAIN\$env:USERNAME" -LogonType Interactive -RunLevel Limited

Register-ScheduledTask -TaskName $taskName -Action $action -Trigger @($logon, $repeat) -Settings $settings -Principal $principal -Description 'Copies the newest AFVI installer from the build server to the desktop.' -Force | Out-Null
Write-Host "registered task '$taskName' (at logon and every $IntervalMinutes min)"
Write-Host "script copy: $installed"
Write-Host "launcher:    $launcher (wscript.exe, runs without any console window)"
Write-Host 'starting it once now ...'
Start-ScheduledTask -TaskName $taskName
Write-Host "log: $(Join-Path $installDir 'fetch-latest.log')"
