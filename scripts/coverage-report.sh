#!/usr/bin/env bash
# SPDX-License-Identifier: MIT
set -euo pipefail

profile="${1:-coverage.out}"

if [[ ! -f "$profile" ]]; then
  echo "coverage profile not found: $profile" >&2
  exit 1
fi

echo "Package coverage (lowest first):"
awk -F'[: ,]+' '
  NR>1 {
    file=$1; stmts=$4; cnt=$5;
    if (stmts == 0) next;
    pkg=file;
    sub(/\/[^/]+$/, "", pkg);
    total[pkg]+=stmts;
    if (cnt > 0) covered[pkg]+=stmts;
  }
  END {
    for (pkg in total) {
      pct=(covered[pkg]/total[pkg])*100;
      printf "%7.2f%%  %s\n", pct, pkg;
    }
  }
' "$profile" | sort -n

echo
echo "Lowest-covered functions:"
go tool cover -func="$profile" | grep -vE "^total:" | sort -k3,3n | head -20
