# SPDX-License-Identifier: MIT

param(
    [string]$Profile = "coverage.out"
)

$ErrorActionPreference = "Stop"

$packages = @(go list ./... | Where-Object { $_ -ne "github.com/skaphos/repokeeper/scripts/perf" })
if ($packages.Count -eq 0) {
    Write-Error "no packages to test"
}

go test "-coverprofile=$Profile" $packages
