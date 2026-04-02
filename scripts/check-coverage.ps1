# SPDX-License-Identifier: MIT

param(
    [string]$Profile = "coverage.out"
)

$ErrorActionPreference = "Stop"

$defaultThreshold = if ($env:COVERAGE_MIN_DEFAULT) { [double]$env:COVERAGE_MIN_DEFAULT } else { 80.0 }

if (-not (Test-Path -LiteralPath $Profile)) {
    Write-Error "coverage profile not found: $Profile"
}

$rows = @{}
Get-Content -LiteralPath $Profile | Select-Object -Skip 1 | ForEach-Object {
    $line = $_.Trim()
    if ($line -eq "") {
        return
    }

    $parts = $line -split '[: ,]+'
    if ($parts.Count -lt 5) {
        return
    }

    $pkg = Split-Path -Path $parts[0] -Parent
    $stmts = [int]$parts[3]
    $count = [int]$parts[4]
    if ($stmts -eq 0) {
        return
    }

    if (-not $rows.ContainsKey($pkg)) {
        $rows[$pkg] = [pscustomobject]@{
            Covered = 0
            Total   = 0
        }
    }

    $rows[$pkg].Total += $stmts
    if ($count -gt 0) {
        $rows[$pkg].Covered += $stmts
    }
}

if ($rows.Count -eq 0) {
    Write-Error "no executable coverage data found in $Profile"
}

Write-Host ("Per-package coverage thresholds (default {0}%):" -f $defaultThreshold)
$failures = 0

$rows.GetEnumerator() | Sort-Object Name | ForEach-Object {
    $pkg = $_.Name
    $covered = $_.Value.Covered
    $total = $_.Value.Total
    $pct = if ($total -eq 0) { 0.0 } else { ($covered / $total) * 100.0 }

    if ($pkg -eq "github.com/skaphos/repokeeper/scripts/perf") {
        "{0,-55} {1,6:N2}% ({2}/{3}) [skipped]" -f $pkg, $pct, $covered, $total | Write-Host
        return
    }

    "{0,-55} {1,6:N2}% ({2}/{3}) [min {4}%]" -f $pkg, $pct, $covered, $total, $defaultThreshold | Write-Host
    if ($pct -lt $defaultThreshold) {
        $failures++
    }
}

if ($failures -gt 0) {
    Write-Error "coverage threshold check failed: $failures package(s) below minimum"
}

Write-Host "coverage threshold check passed"
