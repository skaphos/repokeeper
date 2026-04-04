#!/usr/bin/env bash
# SPDX-License-Identifier: MIT

set -euo pipefail

repo_root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
go_licenses_pkg="github.com/google/go-licenses/v2@v2.0.1"
output_root="${repo_root}/third_party_licenses"

runtime_output_dir="${output_root}/runtime"
runtime_report="${output_root}/runtime-report.csv"

mkdir -p "${output_root}"
rm -rf "${runtime_output_dir}"

run_generation() {
	local module_dir="$1"
	local package_arg="$2"
	local ignore_prefix="$3"
	local report_path="$4"
	local save_path="$5"

	(
		cd "${module_dir}"
		go run "${go_licenses_pkg}" report "${package_arg}" --ignore "${ignore_prefix}" >"${report_path}"
		go run "${go_licenses_pkg}" save "${package_arg}" --ignore "${ignore_prefix}" --save_path "${save_path}"
	)
}

run_generation "${repo_root}" "." "github.com/skaphos/repokeeper" "${runtime_report}" "${runtime_output_dir}"

printf 'Updated third-party notices in %s\n' "${output_root}"
