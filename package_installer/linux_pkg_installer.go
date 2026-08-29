package package_installer

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"guify_my_script/logger"
	"guify_my_script/sudoers"
)

// InstallLocalPackage attempts to install a local package file, determining the correct
// package manager while actively preventing the installation of foreign package types.
func InstallLocalPackage(pkgPath string, updateFirst bool, logLevel logger.LogLevel) error {
	logger.Info(logLevel, "Starting installation process for local package: %s", pkgPath)

	var pkgExt string
	var pkgManager string

	// Temporarily drop root privileges to perform environment checks safely
	err := sudoers.RunAsSudoUser(func() error {
		logger.Debug(logLevel, "Successfully dropped to standard user privileges for safety checks.")

		// 1. Verify the package file actually exists
		if _, err := os.Stat(pkgPath); err != nil {
			return fmt.Errorf("package file not found or inaccessible: %w", err)
		}

		// 2. Identify the file extension (handling multi-part Arch extensions)
		pkgExt = getPackageExtension(pkgPath)

		// 3. Detect the host OS package manager
		pkgManager = detectPackageManager()
		if pkgManager == "" {
			return fmt.Errorf("could not detect a supported package manager on this system")
		}

		return nil
	}) // Regains root privileges upon exiting the closure

	if err != nil {
		logger.Error(logLevel, "Pre-installation validation failed: %v", err)
		return err
	}

	// 4. Fail-Safe: Block foreign packages from damaging the system
	if !isValidForManager(pkgExt, pkgManager) {
		blockErr := fmt.Errorf("foreign package blocked: cannot install a %s file on a %s-based system", pkgExt, pkgManager)
		logger.Fatal(logLevel, "%v", blockErr)
		return blockErr
	}

	// 5. Perform System Update if requested
	if updateFirst {
		logger.Info(logLevel, "Update flag is true. Performing system update via %s before installation...", pkgManager)
		if err := performSystemUpdate(pkgManager, logLevel); err != nil {
			logger.Error(logLevel, "System update failed: %v", err)
			return fmt.Errorf("system update failed: %w", err)
		}
		logger.Info(logLevel, "System update completed successfully.")
	}

	logger.Info(logLevel, "System verified as %s. Preparing to install %s package as root.", pkgManager, pkgExt)

	// 6. Build and execute the root-level installation command
	cmd := buildInstallCommand(pkgManager, pkgPath)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	logger.Debug(logLevel, "Executing command: %s", cmd.String())

	if err := cmd.Run(); err != nil {
		logger.Error(logLevel, "Package installation failed: %v", err)
		return err
	}

	logger.Info(logLevel, "Successfully installed %s!", filepath.Base(pkgPath))
	return nil
}

// performSystemUpdate maps and executes the correct update commands for the host OS.
func performSystemUpdate(manager string, logLevel logger.LogLevel) error {
	var cmds []*exec.Cmd

	switch manager {
	case "pacman":
		cmds = append(cmds, exec.Command("pacman", "-Syu", "--noconfirm"))
	case "apt":
		cmds = append(cmds, exec.Command("apt-get", "update"))
		cmds = append(cmds, exec.Command("apt-get", "upgrade", "-y"))
	case "dnf":
		cmds = append(cmds, exec.Command("dnf", "upgrade", "-y"))
	case "zypper":
		cmds = append(cmds, exec.Command("zypper", "update", "-y"))
	case "apk":
		cmds = append(cmds, exec.Command("apk", "update"))
		cmds = append(cmds, exec.Command("apk", "upgrade"))
	default:
		return fmt.Errorf("unsupported package manager for automated updates: %s", manager)
	}

	for _, cmd := range cmds {
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr

		logger.Debug(logLevel, "Executing update command: %s", cmd.String())

		if err := cmd.Run(); err != nil {
			return err
		}
	}

	return nil
}

// getPackageExtension extracts the file extension, correctly identifying Arch's composite extensions.
func getPackageExtension(path string) string {
	lowerPath := strings.ToLower(path)
	if strings.HasSuffix(lowerPath, ".pkg.tar.zst") {
		return ".pkg.tar.zst"
	}
	if strings.HasSuffix(lowerPath, ".pkg.tar.xz") {
		return ".pkg.tar.xz"
	}
	return filepath.Ext(lowerPath)
}

// detectPackageManager parses /etc/os-release to accurately identify the host OS.
func detectPackageManager() string {
	file, err := os.Open("/etc/os-release")
	if err != nil {
		return "" // Fallback if the file somehow doesn't exist
	}
	defer file.Close()

	var id string
	var idLike string

	// Read /etc/os-release line by line
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, "ID=") {
			id = strings.Trim(strings.TrimPrefix(line, "ID="), `"'`)
		} else if strings.HasPrefix(line, "ID_LIKE=") {
			idLike = strings.Trim(strings.TrimPrefix(line, "ID_LIKE="), `"'`)
		}
	}

	// Combine ID and ID_LIKE to check for distro families
	osIdentifiers := id + " " + idLike

	switch {
	case strings.Contains(osIdentifiers, "arch"):
		return "pacman"
	case strings.Contains(osIdentifiers, "debian") || strings.Contains(osIdentifiers, "ubuntu"):
		return "apt"
	case strings.Contains(osIdentifiers, "fedora") || strings.Contains(osIdentifiers, "rhel") || strings.Contains(osIdentifiers, "centos"):
		return "dnf"
	case strings.Contains(osIdentifiers, "suse"):
		return "zypper"
	case strings.Contains(osIdentifiers, "alpine"):
		return "apk"
	default:
		return ""
	}
}

// isValidForManager maps file extensions to their strictly allowed package manager.
func isValidForManager(ext, manager string) bool {
	switch manager {
	case "pacman":
		return ext == ".pkg.tar.zst" || ext == ".pkg.tar.xz"
	case "apt":
		return ext == ".deb"
	case "dnf", "zypper":
		return ext == ".rpm"
	case "apk":
		return ext == ".apk"
	default:
		return false
	}
}

// buildInstallCommand generates the correct command-line invocation for the local file.
func buildInstallCommand(manager, pkgPath string) *exec.Cmd {
	switch manager {
	case "pacman":
		return exec.Command("pacman", "-U", "--noconfirm", pkgPath)
	case "apt":
		return exec.Command("apt-get", "install", "-y", pkgPath) // apt-get handles local files and their deps elegantly
	case "dnf":
		return exec.Command("dnf", "install", "-y", pkgPath)
	case "zypper":
		return exec.Command("zypper", "install", "-y", "--allow-unsigned-rpm", pkgPath)
	case "apk":
		return exec.Command("apk", "add", "--allow-untrusted", pkgPath)
	default:
		return nil
	}
}
