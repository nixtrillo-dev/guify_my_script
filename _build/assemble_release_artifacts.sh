#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd -- "${SCRIPT_DIR}/.." && pwd)"
DIST_DIR="${PROJECT_ROOT}/dist"

cli_binaries=(
    "guify_my_script_android_aarch64"
    "guify_my_script_linux_aarch64"
    "guify_my_script_linux_amd64"
)
package_dirs=(
    "${DIST_DIR}/com.nixtrillo.guifymyscript_linux_aarch64"
    "${DIST_DIR}/com.nixtrillo.guifymyscript_linux_amd64"
)
package_subdirs=(arch debian fedora)

# Check all inputs before clearing the existing top-level release artifacts.
for binary in "${cli_binaries[@]}"; do
    if [[ ! -f "${PROJECT_ROOT}/${binary}" ]]; then
        printf 'Missing CLI binary: %s\n' "${PROJECT_ROOT}/${binary}" >&2
        exit 1
    fi
done

for package_dir in "${package_dirs[@]}"; do
    for package_subdir in "${package_subdirs[@]}"; do
        found=0
        for package in "${package_dir}/${package_subdir}"/*; do
            if [[ -f "${package}" && ( "${package}" == *.deb || "${package}" == *.rpm || "${package}" == *.pkg.tar.zst ) ]]; then
                found=1
                break
            fi
        done
        if (( ! found )); then
            printf 'Missing install package in %s\n' "${package_dir}/${package_subdir}" >&2
            exit 1
        fi
    done
done

mkdir -p -- "${DIST_DIR}"
# Remove files and symlinks immediately inside dist; preserve its directories.
find "${DIST_DIR}" -maxdepth 1 -type f -delete
find "${DIST_DIR}" -maxdepth 1 -type l -delete

for binary in "${cli_binaries[@]}"; do
    cp -- "${PROJECT_ROOT}/${binary}" "${DIST_DIR}/${binary}_cli"
done

for package_dir in "${package_dirs[@]}"; do
    for package_subdir in "${package_subdirs[@]}"; do
        for package in "${package_dir}/${package_subdir}"/*; do
            if [[ -f "${package}" && ( "${package}" == *.deb || "${package}" == *.rpm || "${package}" == *.pkg.tar.zst ) ]]; then
                cp -- "${package}" "${DIST_DIR}/"
            fi
        done
    done
done

printf 'Release artifacts in %s: ' "${DIST_DIR}"
find "${DIST_DIR}" -maxdepth 1 -type f | wc -l
