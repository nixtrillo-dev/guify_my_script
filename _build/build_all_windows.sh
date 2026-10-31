#!/usr/bin/env bash
set -euo pipefail
cd -- "$(dirname -- "${BASH_SOURCE[0]}")"

if [[ "$(uname -m)" != "x86_64" && "$(uname -m)" != "amd64" ]]; then
    printf 'Error: Windows cross-builds currently require an x86_64 Linux host (found %s).\n' "$(uname -m)" >&2
    exit 1
fi

WINDOWS_CC="${WINDOWS_AMD64_CC:-x86_64-w64-mingw32-gcc}"
if ! command -v "$WINDOWS_CC" >/dev/null 2>&1; then
    printf 'Error: MinGW C cross-compiler is unavailable: %s. Install MinGW-w64 or set WINDOWS_AMD64_CC.\n' "$WINDOWS_CC" >&2
    exit 1
fi

# The Linux executable is the build host tool; the target argument selects the
# Windows Go CLI. It must remain a Linux binary to run on CachyOS.
EXEC="./guify_my_script_linux_amd64"
FALLBACK_EXEC="./guify_my_script_amd64"
if [[ ! -x "$EXEC" ]]; then
    EXEC="$FALLBACK_EXEC"
fi
if [[ ! -x "$EXEC" ]]; then
    printf 'Error: no executable Linux AMD64 build tool found in %s.\n' "$PWD" >&2
    exit 1
fi

"$EXEC" compile_script_with_gui_config ../ 'GO' ../guify_my_script_config.json '' ../main.go ../guify_my_script 1 0 "INFO" 0 0 "windows_amd64"
"$EXEC" generate_ui ../guify_my_script_config.json 'production' 'INFO' '0' ".exe_windows_amd64"
