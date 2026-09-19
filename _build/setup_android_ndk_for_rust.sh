#!/usr/bin/env bash
set -e

# Configuration
NDK_VERSION="26.1.10909125"
ANDROID_SDK_ROOT="$HOME/Android/Sdk"
NDK_HOME_PATH="$ANDROID_SDK_ROOT/ndk/$NDK_VERSION"

# 0. Check and install required dependencies across distros
echo "Checking for required dependencies (Java and unzip)..."
if ! command -v java &> /dev/null || ! command -v unzip &> /dev/null; then
    echo "Dependencies missing. Prompting for sudo access..."
    if command -v apt-get &> /dev/null; then
        sudo apt-get update && sudo apt-get install -y default-jdk unzip
    elif command -v dnf &> /dev/null; then
        sudo dnf install -y java-latest-openjdk unzip
    elif command -v pacman &> /dev/null; then
        sudo pacman -Sy --noconfirm jre-openjdk unzip
    else
        echo "Error: Unsupported package manager. Please install a JRE/JDK and unzip manually."
        exit 1
    fi
else
    echo "Dependencies are already installed."
fi

echo "Setting up Android SDK and NDK in: $ANDROID_SDK_ROOT"

# 1. Download and extract Android Command-line Tools
mkdir -p "$ANDROID_SDK_ROOT/cmdline-tools"
wget -q https://dl.google.com/android/repository/commandlinetools-linux-15859902_latest.zip -O /tmp/cmdline-tools.zip
unzip -q /tmp/cmdline-tools.zip -d /tmp/cmdline-temp
mkdir -p "$ANDROID_SDK_ROOT/cmdline-tools/latest"

if [ -d "/tmp/cmdline-temp/cmdline-tools" ]; then
    cp -r /tmp/cmdline-temp/cmdline-tools/* "$ANDROID_SDK_ROOT/cmdline-tools/latest/"
else
    cp -r /tmp/cmdline-temp/* "$ANDROID_SDK_ROOT/cmdline-temp/latest/"
fi
rm -rf /tmp/cmdline-tools.zip /tmp/cmdline-temp

export PATH="$ANDROID_SDK_ROOT/cmdline-tools/latest/bin:$PATH"

# 2. Purge existing NDK installation if present to force a clean reinstall
if [ -d "$NDK_HOME_PATH" ]; then
    echo "Purging existing NDK at $NDK_HOME_PATH..."
    rm -rf "$NDK_HOME_PATH"
fi

# 3. Install platform-tools and NDK using the Android CLI
echo "Installing platform-tools and NDK $NDK_VERSION via Android CLI..."
android --sdk="$ANDROID_SDK_ROOT" sdk install "platform-tools" "ndk;$NDK_VERSION"

# 4. Configure all available shell profiles (Purging existing blocks cleanly)
echo "Configuring environment variables for available shells..."

# Helper function to remove old variable blocks and orphan comment lines
clean_shell_config() {
    local config_file="$1"
    if [ -f "$config_file" ]; then
        # Delete block delimited by markers
        sed -i '/# BEGIN ANDROID SDK/,/# END ANDROID SDK/d' "$config_file"
        # Delete orphan legacy lines
        sed -i '/# Android SDK & NDK Environment Variables/d' "$config_file"
        sed -i '/ANDROID_HOME/d' "$config_file"
        sed -i '/ANDROID_NDK_HOME/d' "$config_file"
    fi
}

# Bash Configuration
if [ -f "$HOME/.bashrc" ]; then
    clean_shell_config "$HOME/.bashrc"
    cat << EOF >> "$HOME/.bashrc"

# BEGIN ANDROID SDK
# Android SDK & NDK Environment Variables
export ANDROID_HOME="$ANDROID_SDK_ROOT"
export ANDROID_NDK_HOME="$NDK_HOME_PATH"
export PATH="\$ANDROID_HOME/platform-tools:\$PATH"
# END ANDROID SDK
EOF
    echo " -> Updated ~/.bashrc"
fi

# Zsh Configuration
if [ -f "$HOME/.zshrc" ]; then
    clean_shell_config "$HOME/.zshrc"
    cat << EOF >> "$HOME/.zshrc"

# BEGIN ANDROID SDK
# Android SDK & NDK Environment Variables
export ANDROID_HOME="$ANDROID_SDK_ROOT"
export ANDROID_NDK_HOME="$NDK_HOME_PATH"
export PATH="\$ANDROID_HOME/platform-tools:\$PATH"
# END ANDROID SDK
EOF
    echo " -> Updated ~/.zshrc"
fi

# Fish Configuration
FISH_CONFIG="$HOME/.config/fish/config.fish"
if [ -d "$HOME/.config/fish" ] || [ -f "$FISH_CONFIG" ]; then
    mkdir -p "$(dirname "$FISH_CONFIG")"
    clean_shell_config "$FISH_CONFIG"
    cat << EOF >> "$FISH_CONFIG"

# BEGIN ANDROID SDK
# Android SDK & NDK Environment Variables
set -gx ANDROID_HOME "$ANDROID_SDK_ROOT"
set -gx ANDROID_NDK_HOME "$NDK_HOME_PATH"
fish_add_path "$ANDROID_SDK_ROOT/platform-tools"
# END ANDROID SDK
EOF
    echo " -> Updated $FISH_CONFIG"
fi

# 5. Detect current running shell and print exact reload command
PARENT_PROC=$(ps -p $PPID -o comm= 2>/dev/null || true)
USER_SHELL=$(basename "${PARENT_PROC:-$SHELL}")

echo ""
echo "Setup complete! Run this exact command to apply the changes to your current terminal:"
echo ""

case "$USER_SHELL" in
    fish)
        echo "    source ~/.config/fish/config.fish"
        ;;
    zsh)
        echo "    source ~/.zshrc"
        ;;
    *)
        echo "    source ~/.bashrc"
        ;;
esac
echo ""
