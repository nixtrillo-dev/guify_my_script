#!/bin/bash
set -e
cd -- "$(dirname -- "${BASH_SOURCE[0]}")"

# Map the host CPU architecture to the standardized Linux build target.
case "$(uname -m)" in
    x86_64|amd64)
        TARGET_ARCH="linux_amd64"
        EXEC="./guify_my_script_linux_amd64"
        FALLBACK_EXEC="./guify_my_script_amd64"
        ;;
    aarch64|arm64)
        TARGET_ARCH="linux_aarch64"
        EXEC="./guify_my_script_linux_aarch64"
        FALLBACK_EXEC="./guify_my_script_arm64"
        ;;
    *)
        echo "Error: Unsupported architecture ($(uname -m))."
        exit 1
        ;;
esac

if [ ! -x "$EXEC" ]; then
    EXEC="$FALLBACK_EXEC"
fi
if [ ! -x "$EXEC" ]; then
    echo "Error: no executable for $TARGET_ARCH found in the current directory."
    exit 1
fi

# The compiler accepts architecture identifiers. UI packaging uses a package
# format plus architecture for each target, so request all formats for this host.
UI_TARGET_BUILDS=".pkg.tar.zst_${TARGET_ARCH},.deb_${TARGET_ARCH},.rpm_${TARGET_ARCH}"

"$EXEC" compile_script_with_gui_config ../ 'GO' ../guify_my_script_config.json '' ../main.go ../guify_my_script 1 0 "INFO" 0 0 "$TARGET_ARCH"
"$EXEC" generate_ui ../guify_my_script_config.json 'production' 'INFO' '0' "$UI_TARGET_BUILDS"
