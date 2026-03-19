param(
    [string]$Output = "",
    [string]$SevenZipPath = ""
)

$ErrorActionPreference = "Stop"

function Resolve-SevenZip {
    param(
        [string]$ManualPath
    )

    if (-not [string]::IsNullOrWhiteSpace($ManualPath)) {
        if (Test-Path $ManualPath) {
            return (Resolve-Path $ManualPath).Path
        }
        throw "SevenZipPath not found: $ManualPath"
    }

    $cmd = Get-Command 7z -ErrorAction SilentlyContinue
    if ($cmd) {
        return $cmd.Source
    }

    $candidates = @(
        (Join-Path ${env:ProgramFiles} "7-Zip\\7z.exe"),
        (Join-Path ${env:ProgramFiles(x86)} "7-Zip\\7z.exe"),
        "C:\\Program Files\\7-Zip\\7z.exe",
        "C:\\Program Files (x86)\\7-Zip\\7z.exe"
    ) | Select-Object -Unique

    foreach ($path in $candidates) {
        if (-not [string]::IsNullOrWhiteSpace($path) -and (Test-Path $path)) {
            return $path
        }
    }

    throw @"
7z command not found.
Please install 7-Zip, then rerun this script.

Install options:
1) winget install --id 7zip.7zip -e
2) Install manually from https://www.7-zip.org/

If 7-Zip is installed but not in PATH, pass it explicitly:
.\package_source.ps1 -SevenZipPath "C:\Program Files\7-Zip\7z.exe"
"@
}

$repoRoot = (Resolve-Path (Join-Path $PSScriptRoot "..\\..")).Path
if ([string]::IsNullOrWhiteSpace($Output)) {
    $stamp = Get-Date -Format "yyyyMMdd-HHmmss"
    $Output = Join-Path $repoRoot ("IdeaSaver-source-" + $stamp + ".7z")
}

$sevenZipExe = Resolve-SevenZip -ManualPath $SevenZipPath

Push-Location $repoRoot
try {
    $outputDir = Split-Path -Parent $Output
    if (-not [string]::IsNullOrWhiteSpace($outputDir) -and -not (Test-Path $outputDir)) {
        New-Item -ItemType Directory -Path $outputDir -Force | Out-Null
    }

    if (Test-Path $Output) {
        Remove-Item -Force $Output
    }

    $excludeArgs = @(
        "-x!.git",
        "-x!.vscode",
        "-x!web\node_modules\*",
        "-x!web\dist\*",
        "-x!*.log",
        "-x!*.7z"
    )

    & $sevenZipExe "a" "-t7z" $Output ".\*" @excludeArgs
    if ($LASTEXITCODE -ne 0) {
        if (Test-Path $Output) {
            Remove-Item -Force $Output
        }
        throw "7z failed with exit code $LASTEXITCODE"
    }

    Write-Host "Package created: $Output"
    Write-Host "7z executable: $sevenZipExe"
} finally {
    Pop-Location
}
