# ship.ps1 - Hand work over: commit the working tree, push it, put an installer on the Windows desktop.
# Windows PowerShell 5.1 compatible: no &&, ||, ternary or ?? operators.
#
#   .\scripts\ship.ps1 commit -Message MESSAGE
#   .\scripts\ship.ps1 push
#   .\scripts\ship.ps1 desktop [-File FILE] [-Dir DIR]
#   .\scripts\ship.ps1 fetch   [-Ref nightly|vX.Y.Z] [-Wait SECONDS] [-Any] [-Dir DIR]
#
# Same behaviour as scripts/ship-wsl.sh (read its header for the details): commit stages the working
# tree (tracked changes, deletions, new files that .gitignore does not hide; never an already staged
# index, a file over 5 MB or a detached HEAD) and makes ONE commit; push is `git push origin <branch>`;
# desktop copies dist\.last-installer (written by package.ps1) or -File to the desktop or -Dir and
# checks the sha256; fetch downloads the build server's installer from the GitHub Release -Ref
# (default nightly, which must belong to the commit at HEAD unless -Any) and verifies it against
# SHA256SUMS.txt. build.ps1 calls these through -Commit, -Push, -Desktop and -Fetch. The repository comes
# from origin (override with $env:AFVI_REPO = 'owner/name'). Commit and push only run when you ask.
[CmdletBinding()]
param(
    [Parameter(Mandatory = $true, Position = 0)]
    [ValidateSet('commit', 'push', 'desktop', 'fetch')]
    [string]$Action,
    [string]$Message = '',
    [string]$File = '',
    [string]$Dir = '',
    [string]$Ref = 'nightly',
    [int]$Wait = 0,
    [switch]$Any
)

$ErrorActionPreference = 'Stop'

$Root = Split-Path -Parent $PSScriptRoot
Set-Location -LiteralPath $Root

function Fail([string]$Text) {
    Write-Host "ERROR: $Text"
    exit 1
}

# Run git, return its output lines, stop on a non-zero exit code.
function Invoke-Git {
    $out = & git @args
    if ($LASTEXITCODE -ne 0) {
        Fail ("git " + ($args -join ' ') + " failed.")
    }
    return $out
}

function Get-Branch {
    $b = & git symbolic-ref --short -q HEAD
    if ($LASTEXITCODE -ne 0 -or -not $b) {
        Fail 'detached HEAD, switch to a branch first'
    }
    return $b
}

function Get-Sha256([string]$Path) {
    return (Get-FileHash -LiteralPath $Path -Algorithm SHA256).Hash.ToLowerInvariant()
}

# Copy through a temporary name, compare the sha256, then rename into place.
function Install-File([string]$Src, [string]$Folder) {
    if (-not (Test-Path -LiteralPath $Src -PathType Leaf)) { Fail "$Src does not exist" }
    if (-not (Test-Path -LiteralPath $Folder)) { New-Item -ItemType Directory -Path $Folder -Force | Out-Null }
    $name = Split-Path -Leaf $Src
    $tmp = Join-Path $Folder ('.' + $name + '.part')
    $dst = Join-Path $Folder $name
    Copy-Item -LiteralPath $Src -Destination $tmp -Force
    $want = Get-Sha256 $Src
    $got = Get-Sha256 $tmp
    if ($want -ne $got) {
        Remove-Item -LiteralPath $tmp -Force
        Fail "the copy of $name is damaged (sha256 differs)"
    }
    Move-Item -LiteralPath $tmp -Destination $dst -Force
    $mb = [math]::Round((Get-Item -LiteralPath $dst).Length / 1MB, 1)
    Write-Host ("OK: {0}  sha256 {1}...  {2} MB" -f $dst, $got.Substring(0, 16), $mb)
}

function Get-DesktopDir {
    $d = [Environment]::GetFolderPath('Desktop')
    if (-not $d -or -not (Test-Path -LiteralPath $d)) { Fail 'cannot find the desktop folder; pass -Dir' }
    return $d
}

function Invoke-Commit {
    if (-not $Message) { Fail 'commit needs -Message' }
    $null = Get-Branch
    & git diff --cached --quiet
    if ($LASTEXITCODE -ne 0) {
        Fail 'the index already holds staged changes; commit or unstage them first (this commits the working tree and nothing else)'
    }
    $added = @(& git ls-files -o --exclude-standard)
    foreach ($f in $added) {
        if ((Get-Item -LiteralPath $f).Length -gt 5242880) {
            Fail "new file $f is larger than 5 MB; git add it yourself if it belongs in the repository"
        }
    }
    if ($added.Count -gt 0) { $null = Invoke-Git add -- @added }
    $null = Invoke-Git add -u -- .
    & git diff --cached --quiet
    if ($LASTEXITCODE -eq 0) {
        Write-Host 'Nothing to commit.'
        return
    }
    & git diff --cached --stat
    $null = Invoke-Git commit -q -m $Message
    & git log -1 --oneline
}

function Invoke-Push {
    $branch = Get-Branch
    $null = Invoke-Git push origin $branch
}

function Invoke-Desktop {
    $src = $File
    if (-not $src) {
        $rec = Join-Path $Root 'dist\.last-installer'
        if (-not (Test-Path -LiteralPath $rec)) { Fail 'no installer recorded (dist\.last-installer); build one first or pass -File' }
        $src = (Get-Content -LiteralPath $rec -TotalCount 1).Trim()
        if (-not [System.IO.Path]::IsPathRooted($src)) { $src = Join-Path $Root $src }
    }
    $target = $Dir
    if (-not $target) { $target = Get-DesktopDir }
    Install-File $src $target
}

function Get-RepoSlug {
    $url = $env:AFVI_REPO
    if (-not $url) {
        # 2>$null turns git's stderr into a terminating error under 'Stop'.
        $prev = $ErrorActionPreference
        $ErrorActionPreference = 'Continue'
        $url = (& git remote get-url origin 2>$null)
        $ErrorActionPreference = $prev
    }
    $slug = [string]$url
    $slug = $slug -replace '^(git@github\.com:|https?://github\.com/|ssh://git@github\.com/)', ''
    $slug = $slug -replace '\.git$', ''
    if ($slug -notmatch '^[A-Za-z0-9_.-]+/[A-Za-z0-9_.-]+$') {
        Fail "cannot find the GitHub repository (origin is '$url'); set `$env:AFVI_REPO = 'owner/name'"
    }
    return $slug
}

function Invoke-Fetch {
    if ($Ref -notmatch '^[A-Za-z0-9._-]+$') { Fail "bad release tag: $Ref" }
    [Net.ServicePointManager]::SecurityProtocol = [Net.SecurityProtocolType]::Tls12
    $base = 'https://github.com/' + (Get-RepoSlug) + '/releases/download/' + $Ref
    $head = (Invoke-Git rev-parse HEAD) | Select-Object -First 1
    $tmpDir = Join-Path ([IO.Path]::GetTempPath()) ('afvi-ship-' + [Guid]::NewGuid().ToString('N'))
    New-Item -ItemType Directory -Path $tmpDir -Force | Out-Null
    try {
        $deadline = (Get-Date).AddSeconds($Wait)
        $name = ''
        $want = ''
        while ($true) {
            $why = ''
            $sums = Join-Path $tmpDir 'SHA256SUMS.txt'
            $got = $false
            try {
                Invoke-WebRequest -UseBasicParsing -Uri ($base + '/SHA256SUMS.txt') -OutFile $sums
                $got = $true
            } catch {
                $why = "release '$Ref' is not published (yet) or is being replaced"
            }
            if ($got) {
                $line = Get-Content -LiteralPath $sums | Where-Object { $_ -match '^[0-9a-f]{64} [ *]AFVI-Teach-Setup-\S+\.exe$' } | Select-Object -First 1
                if (-not $line) {
                    $why = "SHA256SUMS.txt of release '$Ref' lists no installer"
                } else {
                    $parts = $line -split '\s+'
                    $want = $parts[0]
                    $name = $parts[1].TrimStart('*')
                    $sha = ''
                    if ($name -match '-g([0-9a-f]{7,40})\.exe$') { $sha = $Matches[1] }
                    if ($Any -or $Ref -ne 'nightly' -or ($sha -and $head.StartsWith($sha))) {
                        break
                    }
                    $why = "release '$Ref' holds $name, not commit $($head.Substring(0, 7)) (not pushed yet, or the server is still building)"
                }
            }
            if ((Get-Date) -ge $deadline) { Fail $why }
            Write-Host "waiting: $why"
            Start-Sleep -Seconds 15
        }
        Write-Host "==> downloading $name"
        $exe = Join-Path $tmpDir $name
        Invoke-WebRequest -UseBasicParsing -Uri ($base + '/' + $name) -OutFile $exe
        $have = Get-Sha256 $exe
        if ($have -ne $want) { Fail "$name does not match SHA256SUMS.txt (got $have, want $want)" }
        $target = $Dir
        if (-not $target) { $target = Get-DesktopDir }
        Install-File $exe $target
    } finally {
        Remove-Item -LiteralPath $tmpDir -Recurse -Force -ErrorAction SilentlyContinue
    }
}

switch ($Action) {
    'commit' { Invoke-Commit }
    'push' { Invoke-Push }
    'desktop' { Invoke-Desktop }
    'fetch' { Invoke-Fetch }
}
