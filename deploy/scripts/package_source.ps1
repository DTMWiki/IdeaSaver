param(
    [string]$Output = ""
)

$ErrorActionPreference = "Stop"

$repoRoot = (Resolve-Path (Join-Path $PSScriptRoot "..\\..")).Path
if ([string]::IsNullOrWhiteSpace($Output)) {
    $stamp = Get-Date -Format "yyyyMMdd-HHmmss"
    $Output = Join-Path $repoRoot ("IdeaSaver-source-" + $stamp + ".7z")
}

if (-not (Get-Command 7z -ErrorAction SilentlyContinue)) {
    throw "7z command not found. Install 7-Zip first."
}

Push-Location $repoRoot
try {
    if (Test-Path $Output) {
        Remove-Item -Force $Output
    }

    & 7z a -t7z $Output .\* `
        -xr!.git `
        -xr!.vscode `
        -xr!web\node_modules `
        -xr!web\dist `
        -xr!*.log

    Write-Host "Package created: $Output"
} finally {
    Pop-Location
}
