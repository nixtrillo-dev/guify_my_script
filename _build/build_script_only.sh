#!/bin/bash

# Detect the system architecture
ARCH=$(uname -m)

# Assign the correct executable name
if [ "$ARCH" = "x86_64" ] || [ "$ARCH" = "amd64" ]; then
    if [ -x "./guify_my_script_linux_amd64" ]; then
        EXEC="./guify_my_script_linux_amd64"
    else
        EXEC="./guify_my_script_amd64"
    fi
elif [ "$ARCH" = "aarch64" ] || [ "$ARCH" = "arm64" ]; then
    if [ -x "./guify_my_script_linux_aarch64" ]; then
        EXEC="./guify_my_script_linux_aarch64"
    else
        EXEC="./guify_my_script_arm64"
    fi
else
    echo "Error: Unsupported architecture ($ARCH)."
    exit 1
fi

# (Optional) Verify the executable exists and has run permissions
if [ ! -x "$EXEC" ]; then
    echo "Error: $EXEC not found or not executable in the current directory."
    echo "Run 'chmod +x $EXEC' to grant execution permissions."
    exit 1
fi

# Build the application with the correct architecture
"$EXEC" compile_script_with_gui_config ../ 'GO' ../guify_my_script_config.json '' ../main.go ../guify_my_script 1 0 "INFO" 0 0 ''
