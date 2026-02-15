#!/usr/bin/env bash
set -euo pipefail

profile="${1:-coverage.out}"
default_threshold="${COVERAGE_MIN_DEFAULT:-80}"

if [[ ! -f "$profile" ]]; then
  echo "coverage profile not found: $profile" >&2
  exit 1
fi

threshold_for_pkg() {
  local pkg="$1"
  case "$pkg" in
    # Temporary exception: command wiring is still under active refactor and
    # will be raised toward 80% as Milestone 6.x/7 testability work lands.
    github.com/skaphos/repokeeper/cmd/repokeeper) echo 65 ;;
    *) echo "$default_threshold" ;;
  esac
}

mapfile -t coverage_rows < <(
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
        printf "%s %.2f %d %d\n", pkg, pct, covered[pkg], total[pkg];
      }
    }
  ' "$profile" | sort
)

if [[ ${#coverage_rows[@]} -eq 0 ]]; then
  echo "no executable coverage data found in $profile" >&2
  exit 1
fi

echo "Per-package coverage thresholds (default ${default_threshold}%):"
failures=0
for row in "${coverage_rows[@]}"; do
  pkg="$(awk '{print $1}' <<<"$row")"
  pct="$(awk '{print $2}' <<<"$row")"
  covered="$(awk '{print $3}' <<<"$row")"
  total="$(awk '{print $4}' <<<"$row")"
  threshold="$(threshold_for_pkg "$pkg")"

  printf "  %-55s %6.2f%% (%s/%s) [min %s%%]\n" "$pkg" "$pct" "$covered" "$total" "$threshold"
  if ! awk -v p="$pct" -v t="$threshold" 'BEGIN { exit (p+0 >= t+0) ? 0 : 1 }'; then
    failures=$((failures + 1))
  fi
done

if [[ "$failures" -gt 0 ]]; then
  echo "coverage threshold check failed: ${failures} package(s) below minimum" >&2
  exit 1
fi

echo "coverage threshold check passed"
