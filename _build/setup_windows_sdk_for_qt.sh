#!/usr/bin/env bash
set -euo pipefail

# Install a MinGW-w64 Qt 6 SDK and configure the user-local prefix expected by
# gui_gen/compiler.go. Arch/CachyOS and Fedora use packaged target Qt; other
# supported Linux systems use MXE to build the target SDK locally.

readonly REPO_NAME="ownstuff"
readonly REPO_URL='https://martchus.dyn.f3l.de/repo/arch/ownstuff/os/$arch'
readonly FALLBACK_REPO_URL='https://ftp.f3l.de/~martchus/ownstuff/os/$arch'
readonly PACMAN_CONF="/etc/pacman.conf"
readonly QT_PACKAGE_PREFIX="${HOME}/.local/opt/guify-windows-qt"
readonly ENV_FILE="${HOME}/.config/guify/windows-qt-env.sh"
readonly MXE_ROOT="${HOME}/.local/opt/mxe"

fail() {
    printf 'Error: %s\n' "$*" >&2
    exit 1
}

install_with_pacman() {
    local host_arch="$1"
    if [[ "$host_arch" != "x86_64" ]]; then
        install_mxe_from_source "$host_arch"
        return
    fi

    cat <<'NOTICE'
This setup uses the third-party ownstuff Arch repository maintained by
Martchus to obtain MinGW-w64 Qt 6 packages. It is not an official Qt or
CachyOS repository. Its published configuration uses "SigLevel = Optional
TrustAll" (packages are not authenticated by pacman).
NOTICE
    read -r -p 'Continue and allow pacman to use that repository? [y/N] ' answer
    [[ "$answer" =~ ^[Yy]$ ]] || fail "Cancelled. No system changes were made."

    if ! grep -Eq '^\[ownstuff\][[:space:]]*$' "$PACMAN_CONF"; then
        printf '\nAdding the ownstuff repository to %s...\n' "$PACMAN_CONF"
        sudo tee -a "$PACMAN_CONF" >/dev/null <<EOF

# MinGW-w64 Qt 6 packages for Windows cross-compilation (third-party).
[$REPO_NAME]
SigLevel = Optional TrustAll
Server = $REPO_URL
Server = $FALLBACK_REPO_URL
EOF
    else
        printf 'The [%s] repository is already configured in %s.\n' "$REPO_NAME" "$PACMAN_CONF"
    fi

    sudo pacman -Syu --needed \
        mingw-w64-gcc \
        qt6-base \
        ownstuff/mingw-w64-qt6-base \
        ownstuff/mingw-w64-qt6-imageformats

    local target_prefix="/usr/x86_64-w64-mingw32"
    configure_qt_prefix "$target_prefix" "$target_prefix/bin" "" \
        "$target_prefix/lib/qt6/plugins" \
        "x86_64-w64-mingw32-g++" \
        "x86_64-w64-mingw32-gcc" \
        "$target_prefix/lib/pkgconfig"
}

install_with_dnf() {
    local host_arch="$1"
    local target_prefix="/usr/x86_64-w64-mingw32/sys-root/mingw"

    # Fedora ships Windows-target Qt packages built for both x86_64 and
    # aarch64 Linux hosts, so this path works without emulation on either.
    sudo dnf install -y \
        mingw64-gcc-c++ \
        mingw64-qt6-qtbase \
        mingw64-qt6-qtimageformats \
        qt6-qtbase-devel \
        pkgconf-pkg-config

    [[ -d "$target_prefix" ]] || fail "Fedora MinGW Qt sysroot not found: $target_prefix"
    configure_qt_prefix "$target_prefix" "$target_prefix/bin" "" \
        "$target_prefix/lib/qt6/plugins" \
        "x86_64-w64-mingw32-g++" \
        "x86_64-w64-mingw32-gcc" \
        "$target_prefix/lib/pkgconfig"
}

install_mxe_from_source() {
    local host_arch="$1"
    local jobs="${MXE_BUILD_JOBS:-2}"

    if [[ "$MXE_ROOT" =~ [[:space:]] ]]; then
        fail "MXE requires a checkout path without spaces; set HOME to a path without spaces or edit MXE_ROOT in this script."
    fi
    command -v git >/dev/null 2>&1 || fail "git is required to download MXE."
    command -v make >/dev/null 2>&1 || fail "make is required to build MXE."

    case "$host_arch" in
        x86_64|aarch64) ;;
        *) fail "MXE setup supports x86_64 and aarch64 Linux hosts; found $host_arch." ;;
    esac

    case "$PKG_MANAGER" in
        apt-get)
            sudo apt-get update
            sudo apt-get install -y \
                autoconf automake autopoint bash bison bzip2 clang flex g++ gettext git gperf \
                intltool libgdk-pixbuf2.0-dev libltdl-dev libgl-dev libpcre2-dev libssl-dev \
                libtool-bin libxml-parser-perl lzip make openssl p7zip-full patch perl \
                python3 python3-mako python3-packaging python3-pkg-resources \
                python3-setuptools python-is-python3 ruby sed sqlite3 unzip wget xz-utils
            if [[ "$host_arch" == "x86_64" ]]; then
                sudo apt-get install -y g++-multilib libc6-dev-i386
            fi
            ;;
        dnf)
            sudo dnf install -y \
                autoconf automake bash bison bzip2 clang flex gcc-c++ gdk-pixbuf2-devel \
                gettext git gperf intltool libtool lzip make mesa-libGL-devel openssl \
                openssl-devel p7zip patch perl python3 python3-mako ruby sed unzip wget which xz
            ;;
        pacman)
            sudo pacman -Syu --needed \
                autoconf automake bash bison bzip2 clang flex gcc gettext git gperf intltool \
                libtool lzip make mesa openssl p7zip patch perl python python-mako ruby sed unzip wget xz
            ;;
        *) fail "No supported package manager was found to install MXE build requirements." ;;
    esac

    if [[ ! -d "$MXE_ROOT/.git" ]]; then
        mkdir -p "$(dirname "$MXE_ROOT")"
        git clone https://github.com/mxe/mxe.git "$MXE_ROOT"
    fi

    printf '\nBuilding the Windows Qt 6 SDK with MXE. This can take a long time and use several gigabytes of disk space.\n'
    make -C "$MXE_ROOT" -j"$jobs" \
        MXE_TARGETS=x86_64-w64-mingw32.shared \
        qt6-qtbase qt6-qtimageformats

    local target_prefix="$MXE_ROOT/usr/x86_64-w64-mingw32.shared"
    local target_compiler="$MXE_ROOT/usr/bin/x86_64-w64-mingw32.shared-g++"
    [[ -x "$target_compiler" ]] || fail "MXE compiler was not produced at $target_compiler"
    configure_qt_prefix "$target_prefix" \
        "$target_prefix/qt6/bin" "$target_prefix/bin" \
        "$target_prefix/qt6/plugins" \
        "$target_compiler" \
        "$MXE_ROOT/usr/bin/x86_64-w64-mingw32.shared-gcc" \
        "$target_prefix/lib/pkgconfig"
}

configure_qt_prefix() {
    local target_prefix="$1"
    local runtime_bin="$2"
    local additional_runtime_bin="$3"
    local plugin_dir="$4"
    local compiler="$5"
    local linker="$6"
    local pkgconfig_dir="$7"

    [[ -d "$runtime_bin" ]] || fail "Qt runtime bin directory not found: $runtime_bin"
    [[ -f "$pkgconfig_dir/Qt6Widgets.pc" ]] || fail "Qt6Widgets.pc not found in $pkgconfig_dir"
    [[ -f "$pkgconfig_dir/Qt6Network.pc" ]] || fail "Qt6Network.pc not found in $pkgconfig_dir"
    [[ -f "$plugin_dir/platforms/qwindows.dll" ]] || fail "Qt's qwindows.dll plugin not found under $plugin_dir"
    local compiler_path linker_path
    compiler_path="$(command -v "$compiler" 2>/dev/null || true)"
    linker_path="$(command -v "$linker" 2>/dev/null || true)"
    [[ -x "$compiler_path" ]] || fail "Windows C++ cross-compiler not found: $compiler"
    [[ -x "$linker_path" ]] || fail "Windows C cross-compiler not found: $linker"

    # Make each provider's install tree look like the layout gui_gen expects,
    # without copying files out of the package manager/MXE-owned tree.
    mkdir -p "$QT_PACKAGE_PREFIX/lib"
    mkdir -p "$QT_PACKAGE_PREFIX/bin"
    find "$QT_PACKAGE_PREFIX/bin" -maxdepth 1 -type l -delete
    for runtime_dir in "$runtime_bin" "$additional_runtime_bin"; do
        [[ -n "$runtime_dir" && -d "$runtime_dir" ]] || continue
        for dll in "$runtime_dir"/*.dll; do
            [[ -f "$dll" ]] || continue
            ln -sfn "$dll" "$QT_PACKAGE_PREFIX/bin/$(basename "$dll")"
        done
    done
    ln -sfn "$pkgconfig_dir" "$QT_PACKAGE_PREFIX/lib/pkgconfig"
    ln -sfn "$plugin_dir" "$QT_PACKAGE_PREFIX/plugins"
    mkdir -p "$QT_PACKAGE_PREFIX/host-bin"
    ln -sfn "$compiler_path" "$QT_PACKAGE_PREFIX/host-bin/x86_64-w64-mingw32-g++"
    ln -sfn "$linker_path" "$QT_PACKAGE_PREFIX/host-bin/x86_64-w64-mingw32-gcc"

    mkdir -p "$(dirname "$ENV_FILE")"
    cat >"$ENV_FILE" <<EOF
# Generated by guify_my_script/_build/setup_windows_sdk_for_qt.sh
export WINDOWS_QT6_PREFIX="$QT_PACKAGE_PREFIX"
export WINDOWS_QT6_PKGCONFIG="$QT_PACKAGE_PREFIX/lib/pkgconfig"
export WINDOWS_CXX="$compiler_path"
export WINDOWS_AMD64_CC="$linker_path"
export PATH="$QT_PACKAGE_PREFIX/host-bin:\$PATH"
EOF

    configure_shell_startup "$compiler_path" "$linker_path"
    printf '\nWindows Qt cross-build setup is installed.\n'
    printf 'WINDOWS_QT6_PREFIX=%s\n' "$QT_PACKAGE_PREFIX"
    printf 'C++ cross compiler=%s\n' "$compiler_path"
    printf 'C cross compiler=%s\n' "$linker_path"
    printf 'Use the SDK from any shell; the application also detects this default prefix when GUI-launched processes do not inherit shell variables.\n'
    case "$(basename "${SHELL:-sh}")" in
        fish) printf 'Reload Fish settings with: source "%s/.config/fish/config.fish"\n' "$HOME" ;;
        csh|tcsh) printf 'Reload C shell settings with: source "%s/.cshrc"\n' "$HOME" ;;
        bash|zsh) printf 'Load the variables now with: source "%s"\n' "$ENV_FILE" ;;
        *) printf 'Load the variables now with: . "%s/.profile"\n' "$HOME" ;;
    esac
}

configure_shell_startup() {
    local compiler_path="$1"
    local linker_path="$2"
    local shell_name
    shell_name="$(basename "${SHELL:-sh}")"
    local marker_begin="# BEGIN GUIFY WINDOWS QT"
    local marker_end="# END GUIFY WINDOWS QT"

    write_block() {
        local rc_file="$1"
        local body="$2"
        mkdir -p "$(dirname "$rc_file")"
        [[ -f "$rc_file" ]] || touch "$rc_file"
        sed -i "/$marker_begin/,/$marker_end/d" "$rc_file"
        {
            printf '\n%s\n' "$marker_begin"
            printf '%b\n' "$body"
            printf '%s\n' "$marker_end"
        } >>"$rc_file"
        printf 'Configured %s\n' "$rc_file"
    }

    case "$shell_name" in
        fish)
            write_block "$HOME/.config/fish/config.fish" \
                "set -gx WINDOWS_QT6_PREFIX \"$QT_PACKAGE_PREFIX\"\nset -gx WINDOWS_QT6_PKGCONFIG \"$QT_PACKAGE_PREFIX/lib/pkgconfig\"\nset -gx WINDOWS_CXX \"$compiler_path\"\nset -gx WINDOWS_AMD64_CC \"$linker_path\"\nfish_add_path \"$QT_PACKAGE_PREFIX/host-bin\""
            ;;
        bash) write_block "$HOME/.bashrc" "source \"$ENV_FILE\"" ;;
        zsh) write_block "$HOME/.zshrc" "source \"$ENV_FILE\"" ;;
        csh|tcsh)
            write_block "$HOME/.cshrc" \
                "setenv WINDOWS_QT6_PREFIX \"$QT_PACKAGE_PREFIX\"\nsetenv WINDOWS_QT6_PKGCONFIG \"$QT_PACKAGE_PREFIX/lib/pkgconfig\"\nsetenv WINDOWS_CXX \"$compiler_path\"\nsetenv WINDOWS_AMD64_CC \"$linker_path\"\nsetenv PATH \"$QT_PACKAGE_PREFIX/host-bin:\$PATH\""
            ;;
        *) write_block "$HOME/.profile" "[ -r \"$ENV_FILE\" ] && . \"$ENV_FILE\"" ;;
    esac
}

command -v sudo >/dev/null 2>&1 || fail "Please install sudo or run the required package commands manually as root."
[[ -r /etc/os-release ]] || fail "Cannot identify the Linux distribution (/etc/os-release missing)."
# shellcheck disable=SC1091
source /etc/os-release
readonly DISTRO_ID="${ID:-unknown}"
readonly HOST_ARCH="$(uname -m)"

PKG_MANAGER=""
for candidate in pacman dnf apt-get; do
    if command -v "$candidate" >/dev/null 2>&1; then
        PKG_MANAGER="$candidate"
        break
    fi
done
[[ -n "$PKG_MANAGER" ]] || fail "Supported package managers are pacman, dnf, and apt-get."

case "$DISTRO_ID:$PKG_MANAGER:$HOST_ARCH" in
    cachyos:pacman:x86_64|arch:pacman:x86_64)
        install_with_pacman "$HOST_ARCH"
        ;;
    fedora:dnf:x86_64|fedora:dnf:aarch64)
        install_with_dnf "$HOST_ARCH"
        ;;
    *:apt-get:*)
        PKG_MANAGER="apt-get"
        install_mxe_from_source "$HOST_ARCH"
        ;;
    *:dnf:*)
        install_mxe_from_source "$HOST_ARCH"
        ;;
    *:pacman:*)
        install_mxe_from_source "$HOST_ARCH"
        ;;
    *) fail "No Windows Qt SDK setup route for $DISTRO_ID on $HOST_ARCH." ;;
esac
