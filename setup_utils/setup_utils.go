// Copyright (c) 2026 Nick Rowley
// All rights reserved.
// Use of this source code is governed by an MIT-style license that can be
// found in the LICENSE file.

package setup_utils

import (
	"archive/zip"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"os/user"
	"path/filepath"
	"runtime"
	"strings"

	"guify_my_script/logger"
	"guify_my_script/sudoers"
)

const (
	androidNDKVersion    = "26.1.10909125"
	androidToolsURL      = "https://dl.google.com/android/repository/commandlinetools-linux-15859902_latest.zip"
	windowsQtImage       = "guify-windows-qt-fedora-v2"
	legacyWindowsQtImage = "guify-windows-qt-fedora-v1"
)

const windowsQtDockerfile = `FROM fedora:latest
RUN dnf install -y mingw64-binutils mingw64-gcc-c++ mingw64-qt6-qtbase mingw64-qt6-qtimageformats \
    qt6-qtbase-devel pkgconf-pkg-config && dnf clean all
`

// WindowsQtGUIImage is the image tag prepared by SetupWindowsQtGUIContainer.
func WindowsQtGUIImage() string { return windowsQtImage }

// SetupWindowsQtGUIContainer prepares a Fedora cross-compilation image for
// Windows Qt GUI builds and returns the available Docker or Podman executable.
func SetupWindowsQtGUIContainer(logLevel logger.LogLevel) (string, error) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		return "", fmt.Errorf("Windows Qt GUI container builds require Linux amd64")
	}
	engine := "docker"
	if _, err := exec.LookPath(engine); err != nil {
		if _, err := exec.LookPath("podman"); err == nil {
			engine = "podman"
		} else {
			if err := installDevelopmentDependencies(logLevel, "Windows Qt GUI container build", map[string][]string{
				"apt-get": {"podman"},
				"dnf":     {"podman"},
				"pacman":  {"podman"},
			}); err != nil {
				return "", err
			}
			engine = "podman"
		}
	}
	if err := exec.Command(engine, "image", "inspect", legacyWindowsQtImage).Run(); err == nil {
		if err := exec.Command(engine, "image", "rm", "-f", legacyWindowsQtImage).Run(); err == nil {
			logger.Info(logLevel, "Removed obsolete Windows Qt builder image %s.", legacyWindowsQtImage)
		}
	}
	const platform = "linux/amd64"
	inspect := exec.Command(engine, "image", "inspect", "--format={{.Os}}/{{.Architecture}}", windowsQtImage)
	if output, err := inspect.Output(); err == nil && strings.TrimSpace(string(output)) == platform {
		return engine, nil
	}
	logger.Info(logLevel, "Building Fedora Windows Qt GUI builder image %s...", windowsQtImage)
	build := exec.Command(engine, "build", "--platform", platform, "-t", windowsQtImage, "-")
	build.Stdin = strings.NewReader(windowsQtDockerfile)
	build.Stdout, build.Stderr = os.Stdout, os.Stderr
	if err := build.Run(); err != nil {
		return "", fmt.Errorf("build Fedora Windows Qt GUI image: %w", err)
	}
	return engine, nil
}

// SetupAndroidNDKForRust prepares the Android Rust target. On amd64 it installs
// the Android command-line tools and NDK for native builds; on arm64 the
// containerized builder supplies the NDK sysroot. It is intended to be called
// by a root process; user-owned SDK files and shell profiles are written while
// temporarily de-elevated through sudoers.RunAsSudoUser.
func SetupAndroidNDKForRust(logLevel logger.LogLevel) error {
	if runtime.GOOS != "linux" {
		return fmt.Errorf("Android NDK setup is only supported on Linux")
	}
	if runtime.GOARCH == "arm64" {
		// Run the Android build in an ARM64 Debian container. Its native Clang
		// uses the Android sysroot from Google's NDK archive, so no amd64 tools
		// or CPU emulation are needed.
		engine := "docker"
		if _, dockerErr := exec.LookPath(engine); dockerErr != nil {
			if _, podmanErr := exec.LookPath("podman"); podmanErr == nil {
				engine = "podman"
			} else {
				if err := installDevelopmentDependencies(logLevel, "Android Rust container build", map[string][]string{
					"apt-get": {"podman"},
					"dnf":     {"podman"},
					"pacman":  {"podman"},
				}); err != nil {
					return err
				}
				engine = "podman"
			}
		}
		if err := installRustupToolchain(logLevel, "aarch64-linux-android"); err != nil {
			return fmt.Errorf("install Android Rust target: %w", err)
		}
		logger.Info(logLevel, "Android Rust setup completed; the ARM64 Debian builder image will provide Clang and the Android NDK sysroot.")
		return nil
	}
	// The native SDK setup below uses Google's x86_64 Linux command-line tools.
	if runtime.GOARCH != "amd64" {
		return fmt.Errorf("Android NDK setup is currently supported only on Linux amd64; Google does not provide Linux %s host tools", runtime.GOARCH)
	}

	logger.Info(logLevel, "Checking dependencies for Android SDK and NDK setup...")
	if err := installMissingDependencies(logLevel); err != nil {
		return err
	}

	var sdkRoot, ndkRoot, reloadCommand string
	err := sudoers.RunAsSudoUser(func() error {
		usr, err := currentSetupUser()
		if err != nil {
			return err
		}

		sdkRoot = filepath.Join(usr.HomeDir, "Android", "Sdk")
		ndkRoot = filepath.Join(sdkRoot, "ndk", androidNDKVersion)
		if err := os.MkdirAll(filepath.Join(sdkRoot, "cmdline-tools", "latest"), 0o755); err != nil {
			return fmt.Errorf("create Android SDK directories: %w", err)
		}

		if err := installCommandLineTools(sdkRoot, logLevel); err != nil {
			return err
		}
		if err := installSDKPackages(sdkRoot, logLevel); err != nil {
			return err
		}
		reloadCommand, err = configureShellProfiles(usr.HomeDir, sdkRoot, ndkRoot, logLevel)
		return err
	})
	if err != nil {
		logger.Error(logLevel, "Android SDK and NDK setup failed: %v", err)
		return err
	}
	if err := installRustupToolchain(logLevel, "aarch64-linux-android"); err != nil {
		return fmt.Errorf("install Android Rust target: %w", err)
	}

	logger.Info(logLevel, "Android SDK and NDK setup completed at %s.", sdkRoot)
	if reloadCommand != "" {
		logger.Info(logLevel, "Run `%s` to apply the shell changes to the current terminal.", reloadCommand)
	}
	return nil
}

// SetupGoDependencies installs the Go compiler for source compilation.
func SetupGoDependencies(logLevel logger.LogLevel, crossCompile, setupWindows bool) error {
	if err := installDevelopmentDependencies(logLevel, "Go", map[string][]string{
		"apt-get": {"golang-go"},
		"dnf":     {"golang"},
		"pacman":  {"go"},
	}); err != nil {
		return err
	}
	if crossCompile {
		if err := SetupCrossCompileDependencies(logLevel); err != nil {
			return err
		}
	}
	if setupWindows {
		return installWindowsCrossCompileDependencies(logLevel)
	}
	return nil
}

// SetupRustDependencies installs Rust and Cargo for source compilation. When
// setupAndroid is true, it also prepares the Android NDK/toolchain.
func SetupRustDependencies(logLevel logger.LogLevel, crossCompile, setupAndroid, setupWindows bool) error {
	var targets []string
	if crossCompile {
		targets = append(targets, linuxRustCrossTarget())
	}
	if setupWindows {
		targets = append(targets, "x86_64-pc-windows-gnu")
	}
	if err := installRustupToolchain(logLevel, targets...); err != nil {
		return err
	}
	if crossCompile {
		if err := SetupCrossCompileDependencies(logLevel); err != nil {
			return err
		}
	}
	if setupAndroid {
		if err := SetupAndroidNDKForRust(logLevel); err != nil {
			return fmt.Errorf("Android Rust setup: %w", err)
		}
	}
	if setupWindows {
		if err := installWindowsCrossCompileDependencies(logLevel); err != nil {
			return err
		}
	}
	return nil
}

// installWindowsCrossCompileDependencies installs the MinGW-w64 linker used
// by Go cgo and Rust's x86_64-pc-windows-gnu target.
func installWindowsCrossCompileDependencies(logLevel logger.LogLevel) error {
	if _, err := exec.LookPath("x86_64-w64-mingw32-gcc"); err == nil {
		logger.Info(logLevel, "MinGW-w64 cross-compiler is already installed.")
		return nil
	}
	if err := installDevelopmentDependencies(logLevel, "Windows cross-compilation", map[string][]string{
		"apt-get": {"mingw-w64"},
		"dnf":     {"mingw64-gcc"},
		"pacman":  {"mingw-w64-gcc"},
	}); err != nil {
		return err
	}
	if _, err := exec.LookPath("x86_64-w64-mingw32-gcc"); err != nil {
		return fmt.Errorf("Windows cross-compiler was installed but x86_64-w64-mingw32-gcc is still not on PATH: %w", err)
	}
	return nil
}

// SetupCrossCompileDependencies installs tools needed to cross-compile for
// the other common Linux architecture.
func SetupCrossCompileDependencies(logLevel logger.LogLevel) error {
	var packagesByManager map[string][]string
	switch runtime.GOARCH {
	case "amd64":
		packagesByManager = map[string][]string{
			"apt-get": {"gcc-aarch64-linux-gnu", "libc6-dev-arm64-cross", "build-essential", "podman"},
			"dnf":     {"gcc-aarch64-linux-gnu", "sysroot-aarch64-glibc", "gcc", "make", "podman"},
			"pacman":  {"aarch64-linux-gnu-gcc", "aarch64-linux-gnu-glibc", "base-devel", "podman"},
		}
	case "arm64":
		packagesByManager = map[string][]string{
			"apt-get": {"gcc-x86-64-linux-gnu", "libc6-dev-amd64-cross", "build-essential", "podman"},
			"dnf":     {"gcc-x86_64-linux-gnu", "sysroot-x86_64-glibc", "gcc", "make", "podman"},
			"pacman":  {"x86_64-linux-gnu-gcc", "x86_64-linux-gnu-glibc", "base-devel", "podman"},
		}
	default:
		return fmt.Errorf("cross-compilation dependencies are unsupported on %s; supported host architectures are amd64 and arm64", runtime.GOARCH)
	}
	return installDevelopmentDependencies(logLevel, "cross-compilation", packagesByManager)
}

// linuxRustCrossTarget returns the target opposite the current supported Linux host.
func linuxRustCrossTarget() string {
	if runtime.GOARCH == "arm64" {
		return "x86_64-unknown-linux-gnu"
	}
	return "aarch64-unknown-linux-gnu"
}

// installRustupCrossTarget installs rustup for the invoking user when needed,
// then adds the standard library for the opposite Linux architecture.
func installRustupToolchain(logLevel logger.LogLevel, targets ...string) error {
	if runtime.GOOS != "linux" || (runtime.GOARCH != "amd64" && runtime.GOARCH != "arm64") {
		return fmt.Errorf("Rustup setup is supported only on Linux amd64 and arm64")
	}
	host := "x86_64-unknown-linux-gnu"
	if runtime.GOARCH == "arm64" {
		host = "aarch64-unknown-linux-gnu"
	}
	tempFile, err := os.CreateTemp("", "rustup-init-*")
	if err != nil {
		return fmt.Errorf("create rustup installer file: %w", err)
	}
	tempPath := tempFile.Name()
	defer os.Remove(tempPath)
	if err := tempFile.Close(); err != nil {
		return fmt.Errorf("close rustup installer file: %w", err)
	}
	logger.Info(logLevel, "Downloading rustup installer for %s...", runtime.GOARCH)
	response, err := http.Get("https://static.rust-lang.org/rustup/dist/" + host + "/rustup-init")
	if err != nil {
		return fmt.Errorf("download rustup installer: %w", err)
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		return fmt.Errorf("download rustup installer: HTTP %s", response.Status)
	}
	installer, err := os.OpenFile(tempPath, os.O_WRONLY|os.O_TRUNC, 0o700)
	if err != nil {
		return fmt.Errorf("open rustup installer: %w", err)
	}
	_, copyErr := io.Copy(installer, response.Body)
	closeErr := installer.Close()
	if copyErr != nil {
		return fmt.Errorf("save rustup installer: %w", copyErr)
	}
	if closeErr != nil {
		return fmt.Errorf("close rustup installer: %w", closeErr)
	}
	if err := os.Chmod(tempPath, 0o755); err != nil {
		return fmt.Errorf("make rustup installer executable: %w", err)
	}

	return sudoers.RunAsSudoUser(func() error {
		usr, err := currentSetupUser()
		if err != nil {
			return err
		}
		rustup := filepath.Join(usr.HomeDir, ".cargo", "bin", "rustup")
		if _, err := os.Stat(rustup); err != nil {
			installerArgs := []string{"-y", "--profile", "minimal", "--default-toolchain", "stable"}
			for _, target := range targets {
				installerArgs = append(installerArgs, "--target", target)
			}
			command := exec.Command(tempPath, installerArgs...)
			command.Env = rustupUserEnvironment(usr)
			command.Stdout, command.Stderr = os.Stdout, os.Stderr
			if err := command.Run(); err != nil {
				return fmt.Errorf("install rustup for %s: %w", usr.Username, err)
			}
		}
		toolchainArgs := []string{"toolchain", "install", "stable", "--profile", "minimal"}
		for _, target := range targets {
			toolchainArgs = append(toolchainArgs, "--target", target)
		}
		command := exec.Command(rustup, toolchainArgs...)
		command.Env = rustupUserEnvironment(usr)
		command.Stdout, command.Stderr = os.Stdout, os.Stderr
		if err := command.Run(); err != nil {
			return fmt.Errorf("install Rust stable toolchain for %s: %w", usr.Username, err)
		}
		for _, target := range targets {
			command = exec.Command(rustup, "target", "add", "--toolchain", "stable", target)
			command.Env = rustupUserEnvironment(usr)
			command.Stdout, command.Stderr = os.Stdout, os.Stderr
			if err := command.Run(); err != nil {
				return fmt.Errorf("install Rust target %s for %s: %w", target, usr.Username, err)
			}
			rustc := filepath.Join(usr.HomeDir, ".cargo", "bin", "rustc")
			command = exec.Command(rustc, "--print", "target-libdir", "--target", target)
			command.Env = rustupUserEnvironment(usr)
			targetLibDir, err := command.Output()
			if err != nil {
				return fmt.Errorf("verify Rust target %s for %s: %w", target, usr.Username, err)
			}
			if info, err := os.Stat(strings.TrimSpace(string(targetLibDir))); err != nil || !info.IsDir() {
				return fmt.Errorf("Rust target %s was added for %s, but its standard library directory is missing", target, usr.Username)
			}
		}
		command = exec.Command(rustup, "default", "stable")
		command.Env = rustupUserEnvironment(usr)
		command.Stdout, command.Stderr = os.Stdout, os.Stderr
		if err := command.Run(); err != nil {
			return fmt.Errorf("select Rust stable toolchain for %s: %w", usr.Username, err)
		}
		for _, target := range targets {
			logger.Info(logLevel, "Installed Rust target %s for %s.", target, usr.Username)
		}
		return nil
	})
}

func rustupUserEnvironment(usr *user.User) []string {
	path := filepath.Join(usr.HomeDir, ".cargo", "bin") + string(os.PathListSeparator) + os.Getenv("PATH")
	env := replaceSetupEnv(os.Environ(), "HOME", usr.HomeDir)
	env = replaceSetupEnv(env, "USER", usr.Username)
	// GUI actions may elevate setup through sudo. Do not let root's exported
	// Cargo or rustup directories redirect installation away from this user.
	env = replaceSetupEnv(env, "CARGO_HOME", filepath.Join(usr.HomeDir, ".cargo"))
	env = replaceSetupEnv(env, "RUSTUP_HOME", filepath.Join(usr.HomeDir, ".rustup"))
	return replaceSetupEnv(env, "PATH", path)
}

func replaceSetupEnv(env []string, key, value string) []string {
	prefix := key + "="
	updated := make([]string, 0, len(env)+1)
	for _, entry := range env {
		if !strings.HasPrefix(entry, prefix) {
			updated = append(updated, entry)
		}
	}
	return append(updated, prefix+value)
}

func installMissingDependencies(logLevel logger.LogLevel) error {
	missing := make([]string, 0, 2)
	for _, command := range []string{"java", "unzip"} {
		if _, err := exec.LookPath(command); err != nil {
			missing = append(missing, command)
		}
	}
	if len(missing) == 0 {
		return nil
	}

	return installPackages(logLevel, "Android SDK", map[string][]string{
		"apt-get": {"default-jdk", "unzip"},
		"dnf":     {"java-latest-openjdk", "unzip"},
		"pacman":  {"jre-openjdk", "unzip"},
	})
}

func installDevelopmentDependencies(logLevel logger.LogLevel, name string, packagesByManager map[string][]string) error {
	return installPackages(logLevel, name, packagesByManager)
}

func installPackages(logLevel logger.LogLevel, name string, packagesByManager map[string][]string) error {
	manager, err := detectPackageManager()
	if err != nil {
		return err
	}
	packages, ok := packagesByManager[manager]
	if !ok {
		return fmt.Errorf("unsupported package manager %q; install %s dependencies manually", manager, name)
	}
	logger.Info(logLevel, "Installing %s dependencies with %s: %s", name, manager, strings.Join(packages, ", "))
	var args []string
	switch manager {
	case "apt-get", "dnf":
		args = append(args, "install", "-y")
	case "pacman":
		args = append(args, "-S", "--noconfirm")
	}
	args = append(args, packages...)
	command := exec.Command(manager, args...)
	command.Stdout, command.Stderr = os.Stdout, os.Stderr
	if err := command.Run(); err != nil {
		return fmt.Errorf("install %s dependencies with %s: %w", name, manager, err)
	}
	return nil
}

func detectPackageManager() (string, error) {
	data, err := os.ReadFile("/etc/os-release")
	if err != nil {
		return "", fmt.Errorf("read /etc/os-release: %w", err)
	}
	identifiers := strings.ToLower(string(data))
	switch {
	case strings.Contains(identifiers, "debian") || strings.Contains(identifiers, "ubuntu"):
		return "apt-get", nil
	case strings.Contains(identifiers, "fedora") || strings.Contains(identifiers, "rhel") || strings.Contains(identifiers, "centos"):
		return "dnf", nil
	case strings.Contains(identifiers, "arch"):
		return "pacman", nil
	default:
		return "", fmt.Errorf("could not detect a supported package manager")
	}
}

func currentSetupUser() (*user.User, error) {
	username := os.Getenv("SUDO_USER")
	if username == "" {
		username = os.Getenv("USER")
	}
	if username == "" {
		current, err := user.Current()
		if err != nil {
			return nil, fmt.Errorf("determine setup user: %w", err)
		}
		return current, nil
	}
	usr, err := user.Lookup(username)
	if err != nil {
		return nil, fmt.Errorf("lookup setup user %q: %w", username, err)
	}
	return usr, nil
}

func installCommandLineTools(sdkRoot string, logLevel logger.LogLevel) error {
	tempFile, err := os.CreateTemp("", "android-cmdline-tools-*.zip")
	if err != nil {
		return fmt.Errorf("create command-line tools download: %w", err)
	}
	tempPath := tempFile.Name()
	defer os.Remove(tempPath)
	if err := tempFile.Close(); err != nil {
		return fmt.Errorf("close command-line tools download: %w", err)
	}

	logger.Info(logLevel, "Downloading Android command-line tools...")
	response, err := http.Get(androidToolsURL)
	if err != nil {
		return fmt.Errorf("download Android command-line tools: %w", err)
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		return fmt.Errorf("download Android command-line tools: HTTP %s", response.Status)
	}
	archive, err := os.OpenFile(tempPath, os.O_WRONLY|os.O_TRUNC, 0o600)
	if err != nil {
		return fmt.Errorf("open command-line tools download: %w", err)
	}
	_, copyErr := io.Copy(archive, response.Body)
	closeErr := archive.Close()
	if copyErr != nil {
		return fmt.Errorf("save command-line tools download: %w", copyErr)
	}
	if closeErr != nil {
		return fmt.Errorf("close command-line tools archive: %w", closeErr)
	}

	toolsRoot := filepath.Join(sdkRoot, "cmdline-tools", "latest")
	if err := os.RemoveAll(toolsRoot); err != nil {
		return fmt.Errorf("replace Android command-line tools: %w", err)
	}
	if err := os.MkdirAll(toolsRoot, 0o755); err != nil {
		return fmt.Errorf("recreate Android command-line tools directory: %w", err)
	}
	if err := extractZip(tempPath, toolsRoot); err != nil {
		return fmt.Errorf("extract Android command-line tools: %w", err)
	}
	nestedToolsRoot := filepath.Join(toolsRoot, "cmdline-tools")
	if entries, err := os.ReadDir(nestedToolsRoot); err == nil {
		for _, entry := range entries {
			if err := os.Rename(filepath.Join(nestedToolsRoot, entry.Name()), filepath.Join(toolsRoot, entry.Name())); err != nil {
				return fmt.Errorf("flatten Android command-line tools: %w", err)
			}
		}
		if err := os.Remove(nestedToolsRoot); err != nil {
			return fmt.Errorf("remove temporary command-line tools directory: %w", err)
		}
	}
	return nil
}

func extractZip(archivePath, destination string) error {
	archive, err := zip.OpenReader(archivePath)
	if err != nil {
		return err
	}
	defer archive.Close()
	for _, file := range archive.File {
		target, err := safeZipPath(destination, file.Name)
		if err != nil {
			return err
		}
		if file.FileInfo().IsDir() {
			if err := os.MkdirAll(target, 0o755); err != nil {
				return err
			}
			continue
		}
		if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
			return err
		}
		input, err := file.Open()
		if err != nil {
			return err
		}
		output, err := os.OpenFile(target, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, file.Mode().Perm())
		if err != nil {
			input.Close()
			return err
		}
		_, err = io.Copy(output, input)
		input.Close()
		if closeErr := output.Close(); err == nil {
			err = closeErr
		}
		if err != nil {
			return err
		}
	}
	return nil
}

func safeZipPath(destination, name string) (string, error) {
	clean := filepath.Clean(filepath.FromSlash(name))
	target := filepath.Join(destination, clean)
	root, err := filepath.Abs(destination)
	if err != nil {
		return "", err
	}
	absTarget, err := filepath.Abs(target)
	if err != nil {
		return "", err
	}
	if absTarget != root && !strings.HasPrefix(absTarget, root+string(os.PathSeparator)) {
		return "", fmt.Errorf("archive entry escapes destination: %q", name)
	}
	return target, nil
}

func installSDKPackages(sdkRoot string, logLevel logger.LogLevel) error {
	androidCLI := filepath.Join(sdkRoot, "cmdline-tools", "latest", "bin", "android")
	if _, err := os.Stat(androidCLI); err != nil {
		return fmt.Errorf("Android CLI was not installed: %w", err)
	}
	logger.Info(logLevel, "Installing platform-tools and Android NDK %s...", androidNDKVersion)
	command := exec.Command(androidCLI, "--sdk="+sdkRoot, "sdk", "install", "platform-tools", "ndk;"+androidNDKVersion)
	command.Env = append(os.Environ(), "ANDROID_HOME="+sdkRoot, "ANDROID_SDK_ROOT="+sdkRoot)
	command.Stdin = strings.NewReader(strings.Repeat("y\n", 100))
	command.Stdout, command.Stderr = os.Stdout, os.Stderr
	if err := command.Run(); err != nil {
		return fmt.Errorf("install Android SDK packages: %w", err)
	}
	return nil
}

func configureShellProfiles(home, sdkRoot, ndkRoot string, logLevel logger.LogLevel) (string, error) {
	block := "# BEGIN ANDROID SDK\n# Android SDK & NDK Environment Variables\nexport ANDROID_HOME=\"" + sdkRoot + "\"\nexport ANDROID_NDK_HOME=\"" + ndkRoot + "\"\nexport PATH=\"$ANDROID_HOME/platform-tools:$PATH\"\n# END ANDROID SDK\n"
	for _, path := range []string{filepath.Join(home, ".bashrc"), filepath.Join(home, ".zshrc")} {
		if _, err := os.Stat(path); err == nil {
			if err := updateProfile(path, block); err != nil {
				return "", err
			}
		}
	}
	fishConfig := filepath.Join(home, ".config", "fish", "config.fish")
	if _, err := os.Stat(filepath.Dir(fishConfig)); err == nil {
		fishBlock := "# BEGIN ANDROID SDK\n# Android SDK & NDK Environment Variables\nset -gx ANDROID_HOME \"" + sdkRoot + "\"\nset -gx ANDROID_NDK_HOME \"" + ndkRoot + "\"\nfish_add_path \"" + filepath.Join(sdkRoot, "platform-tools") + "\"\n# END ANDROID SDK\n"
		if err := updateProfile(fishConfig, fishBlock); err != nil {
			return "", err
		}
	}

	shell := filepath.Base(os.Getenv("SHELL"))
	switch shell {
	case "fish":
		return "source ~/.config/fish/config.fish", nil
	case "zsh":
		return "source ~/.zshrc", nil
	default:
		return "source ~/.bashrc", nil
	}
}

func updateProfile(path, block string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("read shell profile %s: %w", path, err)
	}
	content := string(data)
	for {
		start := strings.Index(content, "# BEGIN ANDROID SDK")
		end := strings.Index(content, "# END ANDROID SDK")
		if start < 0 || end < start {
			break
		}
		content = content[:start] + content[end+len("# END ANDROID SDK"):]
	}
	lines := strings.Split(content, "\n")
	filtered := lines[:0]
	for _, line := range lines {
		if strings.Contains(line, "# Android SDK & NDK Environment Variables") || strings.Contains(line, "ANDROID_HOME") || strings.Contains(line, "ANDROID_NDK_HOME") {
			continue
		}
		filtered = append(filtered, line)
	}
	content = strings.TrimRight(strings.Join(filtered, "\n"), "\n") + "\n\n" + block
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		return fmt.Errorf("update shell profile %s: %w", path, err)
	}
	return nil
}
