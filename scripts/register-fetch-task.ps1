# register-fetch-task.ps1 - run scripts\fetch-latest.ps1 automatically with a Windows scheduled task.
# Windows PowerShell 5.1 compatible. Per-user task: no administrator rights needed; it runs when you
# are logged on (so your ssh key and desktop are available), at logon and then every N minutes.
#
#   .\scripts\register-fetch-task.ps1 [-IntervalMinutes 10] [-SshHost AI] [-RemoteDir '~/afvi-latest'] [-Dest DIR]
#   .\scripts\register-fetch-task.ps1 -Unregister
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

if ($Unregister) {
    if (Get-ScheduledTask -TaskName $taskName -ErrorAction SilentlyContinue) {
        Unregister-ScheduledTask -TaskName $taskName -Confirm:$false
        Write-Host "removed task '$taskName'"
    } else {
        Write-Host "task '$taskName' is not registered"
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

# A stable copy outside the repository, so the task keeps working when the repo moves or is
# switched to another branch.
$installDir = Join-Path $env:LOCALAPPDATA 'AFVI'
if (-not (Test-Path -LiteralPath $installDir)) {
    New-Item -ItemType Directory -Path $installDir -Force | Out-Null
}
$installed = Join-Path $installDir 'fetch-latest.ps1'
Copy-Item -LiteralPath $script -Destination $installed -Force

$argList = '-NoProfile -NonInteractive -WindowStyle Hidden -ExecutionPolicy Bypass -File "{0}" -Quiet -SshHost "{1}" -RemoteDir "{2}"' -f $installed, $SshHost, $RemoteDir
if ($Dest -ne '') {
    $argList += (' -Dest "{0}"' -f $Dest)
}
$action = New-ScheduledTaskAction -Execute 'powershell.exe' -Argument $argList

$start = (Get-Date).AddMinutes(1)
$repeat = New-ScheduledTaskTrigger -Once -At $start -RepetitionInterval (New-TimeSpan -Minutes $IntervalMinutes) -RepetitionDuration (New-TimeSpan -Days 3650)
$logon = New-ScheduledTaskTrigger -AtLogOn -User "$env:USERDOMAIN\$env:USERNAME"
$settings = New-ScheduledTaskSettingsSet -StartWhenAvailable -AllowStartIfOnBatteries -DontStopIfGoingOnBatteries -MultipleInstances IgnoreNew -ExecutionTimeLimit (New-TimeSpan -Minutes 10)
$principal = New-ScheduledTaskPrincipal -UserId "$env:USERDOMAIN\$env:USERNAME" -LogonType Interactive -RunLevel Limited

Register-ScheduledTask -TaskName $taskName -Action $action -Trigger @($logon, $repeat) -Settings $settings -Principal $principal -Description 'Copies the newest AFVI installer from the build server to the desktop.' -Force | Out-Null
Write-Host "registered task '$taskName' (at logon and every $IntervalMinutes min)"
Write-Host "script copy: $installed"
Write-Host 'starting it once now ...'
Start-ScheduledTask -TaskName $taskName
Write-Host "log: $(Join-Path $installDir 'fetch-latest.log')"
