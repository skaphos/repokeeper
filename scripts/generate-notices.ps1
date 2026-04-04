# SPDX-License-Identifier: MIT

$ErrorActionPreference = "Stop"

$RepoRoot = Split-Path -Parent $PSScriptRoot
$GoLicensesPackage = "github.com/google/go-licenses/v2@v2.0.1"
$OutputRoot = Join-Path $RepoRoot "third_party_licenses"

$RuntimeOutputDir = Join-Path $OutputRoot "runtime"
$RuntimeReport = Join-Path $OutputRoot "runtime-report.csv"

New-Item -ItemType Directory -Force -Path $OutputRoot | Out-Null

foreach ($Path in @($RuntimeOutputDir)) {
    if (Test-Path -LiteralPath $Path) {
        Remove-Item -LiteralPath $Path -Recurse -Force
    }
}

function Invoke-NoticeGeneration {
    param(
        [string]$ModuleDir,
        [string]$PackageArg,
        [string]$IgnorePrefix,
        [string]$ReportPath,
        [string]$SavePath
    )

    Push-Location $ModuleDir
    try {
        $report = go run $GoLicensesPackage report $PackageArg --ignore $IgnorePrefix
        Set-Content -LiteralPath $ReportPath -Value $report
        go run $GoLicensesPackage save $PackageArg --ignore $IgnorePrefix --save_path $SavePath
    }
    finally {
        Pop-Location
    }
}

Invoke-NoticeGeneration -ModuleDir $RepoRoot -PackageArg "." -IgnorePrefix "github.com/skaphos/repokeeper" -ReportPath $RuntimeReport -SavePath $RuntimeOutputDir

Write-Host ("Updated third-party notices in {0}" -f $OutputRoot)
