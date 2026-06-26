# Windows/PowerShell version of run.sh. Downloads the pinned release binary on
# first run, verifies its SHA-256, caches it, then runs it.
# Not used by default: plugin.json launches run.sh via bash (Git Bash on Windows).
# Switch to this with: "command": "cmd", "args": ["/c", "powershell",
# "-ExecutionPolicy", "Bypass", "-File", "${CLAUDE_PLUGIN_ROOT}/bin/run.ps1"]

$ErrorActionPreference = 'Stop'

$Repo    = 'dimitar-grigorov/mcp-file-tools'
$Version = 'v1.7.0'   # bump on each plugin release

$dataRoot = if ($env:CLAUDE_PLUGIN_DATA) { $env:CLAUDE_PLUGIN_DATA } else { Join-Path $env:LOCALAPPDATA 'mcp-file-tools' }
$binDir   = Join-Path $dataRoot 'bin'
New-Item -ItemType Directory -Force -Path $binDir | Out-Null

$arch  = if ([System.Runtime.InteropServices.RuntimeInformation]::ProcessArchitecture -eq 'Arm64') { 'arm64' } else { 'amd64' }
$asset = "mcp-file-tools_windows_$arch.exe"
$bin   = Join-Path $binDir "mcp-file-tools-$Version-windows-$arch.exe"

if (-not (Test-Path $bin)) {
    $base = "https://github.com/$Repo/releases/download/$Version"
    $tmp  = New-Item -ItemType Directory -Path (Join-Path ([System.IO.Path]::GetTempPath()) ([System.Guid]::NewGuid()))
    try {
        [Console]::Error.WriteLine("mcp-file-tools: downloading $Version (windows/$arch)...")
        $assetPath = Join-Path $tmp $asset
        Invoke-WebRequest -Uri "$base/$asset" -OutFile $assetPath
        $sumPath = Join-Path $tmp 'checksums.txt'
        Invoke-WebRequest -Uri "$base/checksums.txt" -OutFile $sumPath

        $want = (Select-String -Path $sumPath -Pattern ([regex]::Escape($asset) + '$') | Select-Object -First 1).Line -replace '\s.*$', ''
        $got  = (Get-FileHash -Path $assetPath -Algorithm SHA256).Hash.ToLower()
        if (-not $want -or $want.ToLower() -ne $got) {
            throw "checksum mismatch for $asset (want=$want got=$got)"
        }
        Move-Item -Force $assetPath $bin
    }
    finally { Remove-Item -Recurse -Force $tmp }
}

& $bin @args
