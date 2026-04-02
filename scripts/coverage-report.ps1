# SPDX-License-Identifier: MIT

param(
    [string]$Profile = "coverage.out"
)

$ErrorActionPreference = "Stop"

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

Write-Host "Package coverage (lowest first):"
$rows.GetEnumerator() |
    ForEach-Object {
        $pct = if ($_.Value.Total -eq 0) { 0.0 } else { ($_.Value.Covered / $_.Value.Total) * 100.0 }
        [pscustomobject]@{
            Pct = $pct
            Pkg = $_.Name
        }
    } |
    Sort-Object Pct, Pkg |
    ForEach-Object {
        "{0,7:N2}%  {1}" -f $_.Pct, $_.Pkg | Write-Host
    }

Write-Host ""
Write-Host "Lowest-covered functions:"
go tool cover "-func=$Profile" |
    Select-String -NotMatch '^total:' |
    ForEach-Object {
        $parts = ($_ -split '\s+') | Where-Object { $_ -ne "" }
        if ($parts.Count -ge 3) {
            [pscustomobject]@{
                Name = $parts[0]
                Pct  = [double]$parts[2].TrimEnd('%')
                Line = $_.ToString()
            }
        }
    } |
    Sort-Object Pct, Name |
    Select-Object -First 20 |
    ForEach-Object { $_.Line | Write-Host }
