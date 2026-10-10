# SPDX-License-Identifier: MIT

param(
    [string]$Profile = "coverage.out"
)

$ErrorActionPreference = "Stop"

# Windows PowerShell -File neither applies ErrorActionPreference to native
# commands nor propagates their exit codes, so each native call is checked and
# its status returned explicitly; otherwise a failing go test exits 0.
$packages = @(go list ./... | Where-Object { $_ -ne "github.com/skaphos/repokeeper/v2/scripts/perf" })
if ($LASTEXITCODE -ne 0) {
    exit $LASTEXITCODE
}
if ($packages.Count -eq 0) {
    Write-Error "no packages to test"
}

go test "-coverprofile=$Profile" $packages
exit $LASTEXITCODE
