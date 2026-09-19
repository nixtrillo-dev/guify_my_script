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
	androidNDKVersion = "26.1.10909125"
	androidToolsURL   = "https://dl.google.com/android/repository/commandlinetools-linux-15859902_latest.zip"
)

// SetupAndroidNDKForRust installs the Android command-line tools, platform-tools,
// and the NDK version used by the Rust Android build. It is intended to be
// called by a root process; user-owned SDK files and shell profiles are written
// while temporarily de-elevated through sudoers.RunAsSudoUser.
func SetupAndroidNDKForRust(logLevel logger.LogLevel) error {
	if runtime.GOOS != "linux" {
		return fmt.Errorf("Android NDK setup is only supported on Linux")
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

	logger.Info(logLevel, "Android SDK and NDK setup completed at %s.", sdkRoot)
	if reloadCommand != "" {
		logger.Info(logLevel, "Run `%s` to apply the shell changes to the current terminal.", reloadCommand)
	}
	return nil
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

	manager, err := detectPackageManager()
	if err != nil {
		return err
	}
	logger.Info(logLevel, "Installing missing dependencies with %s: %s", manager, strings.Join(missing, ", "))
	args := []string{}
	switch manager {
	case "apt-get":
		args = append(args, "install", "-y", "default-jdk", "unzip")
	case "dnf":
		args = append(args, "install", "-y", "java-latest-openjdk", "unzip")
	case "pacman":
		args = append(args, "-S", "--noconfirm", "jre-openjdk", "unzip")
	default:
		return fmt.Errorf("unsupported package manager %q; install Java and unzip manually", manager)
	}
	command := exec.Command(manager, args...)
	command.Stdout, command.Stderr = os.Stdout, os.Stderr
	if err := command.Run(); err != nil {
		return fmt.Errorf("install dependencies with %s: %w", manager, err)
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
