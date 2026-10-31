// Copyright (c) 2026 Nick Rowley
// All rights reserved.
// Use of this source code is governed by an MIT-style license that can be
// found in the LICENSE file.

package gui_gen

import (
	"bytes"
	"encoding/binary"
	"encoding/csv"
	"fmt"
	"image"
	"image/png"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	"guify_my_script/logger"
	"guify_my_script/setup_utils"
)

const amd64Dockerfile = `FROM debian:bookworm-slim
ENV DEBIAN_FRONTEND=noninteractive
RUN apt-get update && \
    apt-get install -y build-essential qt6-base-dev pkg-config && \
    rm -rf /var/lib/apt/lists/*
`

const amd64CrossDockerfile = `FROM debian:bookworm-slim
ENV DEBIAN_FRONTEND=noninteractive
RUN dpkg --add-architecture amd64 && \
    apt-get update && \
    apt-get install -y crossbuild-essential-amd64 qt6-base-dev:amd64 pkg-config && \
    rm -rf /var/lib/apt/lists/*
RUN ln -sf /usr/bin/x86_64-linux-gnu-g++ /usr/bin/g++
ENV PKG_CONFIG_LIBDIR=/usr/lib/x86_64-linux-gnu/pkgconfig
`

const arm64Dockerfile = `FROM debian:bookworm-slim
ENV DEBIAN_FRONTEND=noninteractive
RUN dpkg --add-architecture arm64 && \
    apt-get update && \
    apt-get install -y crossbuild-essential-arm64 qt6-base-dev:arm64 pkg-config && \
    rm -rf /var/lib/apt/lists/*
RUN ln -sf /usr/bin/aarch64-linux-gnu-g++ /usr/bin/g++
ENV PKG_CONFIG_PATH=/usr/lib/aarch64-linux-gnu/pkgconfig
`

const arm64NativeDockerfile = `FROM debian:bookworm-slim
ENV DEBIAN_FRONTEND=noninteractive
RUN apt-get update && \
    apt-get install -y build-essential qt6-base-dev pkg-config && \
    rm -rf /var/lib/apt/lists/*
`

// BuildContext bundles all the necessary data required to compile and stage the application.
type BuildContext struct {
	CompileMode            string
	TargetArchs            []string
	BaseDir                string
	PackageName            string
	BundledSourceName      string
	SourcePathLinuxAMD64   string
	SourcePathLinuxAArch64 string
	SourcePathWindowsAMD64 string
	SourcePathAMD64        string // Deprecated: Kept for backwards compatibility
	SourcePathARM64        string // Deprecated: Kept for backwards compatibility
	FinalSourceCode        string
	DefaultsJSON           []byte
	RootConfig             RootConfig
	IconName               string
	IconPath               string
	LogLevel               logger.LogLevel
	ClearCache             bool
	TargetBuilds           string
}

// ensureBuilderImage checks if a container image exists locally. If not, it builds it from the Dockerfile string.
func ensureBuilderImage(engine, tag, platform, dockerfile string, logLevel logger.LogLevel) error {
	inspectCmd := exec.Command(engine, "image", "inspect", "--format={{.Os}}/{{.Architecture}}", tag)
	if output, err := inspectCmd.Output(); err == nil && strings.TrimSpace(string(output)) == platform {
		return nil
	}

	logger.Info(logLevel, "Builder image '%s' for %s not found locally. Building container...", tag, platform)

	buildCmd := exec.Command(engine, "build", "--platform", platform, "-t", tag, "-")
	buildCmd.Stdin = strings.NewReader(dockerfile)
	buildCmd.Stdout = os.Stdout
	buildCmd.Stderr = os.Stderr

	if err := buildCmd.Run(); err != nil {
		return fmt.Errorf("failed building container image '%s': %w", tag, err)
	}

	logger.Info(logLevel, "Successfully created builder image '%s'.", tag)
	return nil
}

// RunBuildLoop handles dynamic compiler selection, C++ compilation, and asset staging.
func runBuildLoop(ctx BuildContext) error {
	needsLinuxBuild := false
	for _, target := range ctx.TargetArchs {
		if strings.HasPrefix(target, "linux_") || target == "amd64" || target == "arm64" || target == "aarch64" {
			needsLinuxBuild = true
			break
		}
	}
	containerEngine := ""
	if needsLinuxBuild {
		// Preserve the existing container builder path for Linux targets.
		containerEngine = "docker"
		if _, err := exec.LookPath("docker"); err != nil {
			if _, err := exec.LookPath("podman"); err == nil {
				containerEngine = "podman"
			} else {
				return fmt.Errorf("neither 'docker' nor 'podman' was found in system PATH")
			}
		}
	}

	if ctx.ClearCache && needsLinuxBuild {
		logger.Info(ctx.LogLevel, "Clearing build caches and builder images to download dependencies from scratch...")
		// Untag images to force a rebuild from the Dockerfile
		_ = exec.Command(containerEngine, "rmi", "-f", "qt6-amd64-builder", "qt6-arm64-builder").Run()
		// Remove named volumes persisting DNF/Pacman databases and package cache
		_ = exec.Command(containerEngine, "volume", "rm", "-f", "arch_pacman_cache", "arch_pacman_db", "arch_yay_cache", "fedora_dnf_cache").Run()
	}
	if ctx.ClearCache {
		for _, target := range ctx.TargetArchs {
			if target != "windows_amd64" {
				continue
			}
			engine := containerEngine
			if engine == "" {
				engine = "docker"
				if _, err := exec.LookPath(engine); err != nil {
					if _, err := exec.LookPath("podman"); err == nil {
						engine = "podman"
					} else {
						break // SetupWindowsQtGUIContainer will report or install the missing engine.
					}
				}
			}
			if err := exec.Command(engine, "image", "rm", "-f", setup_utils.WindowsQtGUIImage()).Run(); err != nil {
				logger.Info(ctx.LogLevel, "Windows Qt builder image was not present in the local cache.")
			} else {
				logger.Info(ctx.LogLevel, "Removed cached Windows Qt builder image %s.", setup_utils.WindowsQtGUIImage())
			}
			break
		}
	}

	for _, targetArch := range ctx.TargetArchs {
		logger.Info(ctx.LogLevel, "\n--- Starting build for architecture: %s ---", targetArch)
		if targetArch == "windows_amd64" {
			if err := buildWindowsGUI(ctx); err != nil {
				return err
			}
			continue
		}

		archOutputDir := filepath.Join(ctx.BaseDir, "build_output")
		archOutputBinary := filepath.Join(archOutputDir, "custom_toolkit_app")
		archCppFile := filepath.Join(archOutputDir, "generated_app.cpp")

		archStageDir := filepath.Join(ctx.BaseDir, "dist", fmt.Sprintf("%s_%s", ctx.PackageName, targetArch))
		archPayloadDir := filepath.Join(archStageDir, "payload")
		archAppDir := filepath.Join(archPayloadDir, "opt", ctx.PackageName)
		archBuildDir := filepath.Join(archStageDir, "build")

		if ctx.CompileMode == "production" {
			logger.Info(ctx.LogLevel, "Cleaning staging directories for requested targets...")

			// Always clean the core payload and build artifacts to ensure the new C++ binary is fresh
			_ = os.RemoveAll(archPayloadDir)
			_ = os.RemoveAll(archBuildDir)

			// Selectively clean package manager directories if we are targeting them
			if ctx.TargetBuilds == "" {
				// Empty string means build all, so wipe all PM directories
				_ = os.RemoveAll(filepath.Join(archStageDir, "arch"))
				_ = os.RemoveAll(filepath.Join(archStageDir, "debian"))
				_ = os.RemoveAll(filepath.Join(archStageDir, "fedora"))
			} else {
				reader := csv.NewReader(strings.NewReader(ctx.TargetBuilds))
				reader.TrimLeadingSpace = true
				builds, err := reader.Read()

				if err == nil || err.Error() == "EOF" {
					for _, b := range builds {
						cleanBuild := strings.TrimSpace(b)
						if isBuildMatch(cleanBuild, ".pkg.tar.zst", targetArch) {
							_ = os.RemoveAll(filepath.Join(archStageDir, "arch"))
						} else if isBuildMatch(cleanBuild, ".deb", targetArch) {
							_ = os.RemoveAll(filepath.Join(archStageDir, "debian"))
						} else if isBuildMatch(cleanBuild, ".rpm", targetArch) {
							_ = os.RemoveAll(filepath.Join(archStageDir, "fedora"))
						}
					}
				}
			}

			_ = os.MkdirAll(archAppDir, 0755)
			_ = os.MkdirAll(archBuildDir, 0755)

			archOutputDir = archBuildDir
			archCppFile = filepath.Join(archOutputDir, "generated_app.cpp")
			archOutputBinary = filepath.Join(archAppDir, ctx.PackageName)
		} else {
			_ = os.MkdirAll(archOutputDir, 0755)
		}

		if err := os.WriteFile(archCppFile, []byte(ctx.FinalSourceCode), 0644); err != nil {
			return fmt.Errorf("failed creating source file for %s: %w", targetArch, err)
		}

		// --- CONTAINER SETUP & COMPILATION ---
		var builderTag, dockerfile, compilerBin, pkgConfigEnv, platform string

		if targetArch == "linux_aarch64" || targetArch == "arm64" || targetArch == "aarch64" {
			builderTag = "qt6-arm64-builder"
			if runtime.GOARCH == "arm64" {
				platform = "linux/arm64"
				dockerfile = arm64NativeDockerfile
				compilerBin = "g++"
			} else {
				platform = "linux/amd64"
				dockerfile = arm64Dockerfile
				compilerBin = "aarch64-linux-gnu-g++"
				pkgConfigEnv = "PKG_CONFIG_LIBDIR=/usr/lib/aarch64-linux-gnu/pkgconfig "
			}
		} else {
			if runtime.GOARCH == "arm64" {
				builderTag = "qt6-amd64-builder"
				platform = "linux/arm64"
				dockerfile = amd64CrossDockerfile
				compilerBin = "x86_64-linux-gnu-g++"
				pkgConfigEnv = "PKG_CONFIG_LIBDIR=/usr/lib/x86_64-linux-gnu/pkgconfig "
			} else {
				builderTag = "qt6-amd64-builder"
				platform = "linux/amd64"
				dockerfile = amd64Dockerfile
				compilerBin = "g++"
				pkgConfigEnv = ""
			}
		}

		// 1. Ensure the required builder image exists locally
		if err := ensureBuilderImage(containerEngine, builderTag, platform, dockerfile, ctx.LogLevel); err != nil {
			return err
		}

		// 2. Mount only the directory needed for this compilation. The :Z option
		// gives SELinux-confined Podman/Docker access on Fedora without relabeling
		// the entire project tree.
		workspaceDir := archOutputDir
		if ctx.CompileMode == "production" {
			workspaceDir = archStageDir
		}
		relCppFile, _ := filepath.Rel(workspaceDir, archCppFile)
		relOutputBinary, _ := filepath.Rel(workspaceDir, archOutputBinary)

		// 3. Resolve pkg-config flags inside the container
		pkgCmdStr := fmt.Sprintf("%s run --rm --platform %s %s sh -c \"%spkg-config --cflags --libs Qt6Widgets Qt6Network\"",
			containerEngine, platform, builderTag, pkgConfigEnv)
		pkgOut, err := exec.Command("sh", "-c", pkgCmdStr).CombinedOutput()
		if err != nil {
			return fmt.Errorf("failed resolving pkg-config inside %s container: %w: %s", builderTag, err, strings.TrimSpace(string(pkgOut)))
		}
		pkgFlags := strings.TrimSpace(string(pkgOut))

		// 4. Compile binary inside the container
		compileCmdStr := fmt.Sprintf(
			"%s run --rm --platform %s -v \"%s:/workspace:Z\" -w /workspace %s sh -c "+
				"\"%s -O3 -s -std=c++17 -fPIC -flto=auto -fvisibility=hidden -ffunction-sections -fdata-sections '%s' %s -o '%s' -Wl,--gc-sections\"",
			containerEngine, platform, workspaceDir, builderTag, compilerBin, relCppFile, pkgFlags, relOutputBinary,
		)

		cmd := exec.Command("sh", "-c", compileCmdStr)
		cmd.Env = []string{"PATH=" + os.Getenv("PATH")}
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr

		if err := cmd.Run(); err != nil {
			return fmt.Errorf("container compilation failed for %s (%s): %w", targetArch, builderTag, err)
		}

		if ctx.CompileMode == "production" {
			// --- PRODUCTION ASSET BUNDLING ---
			actualSourcePath := ""
			if targetArch == "linux_amd64" || targetArch == "amd64" {
				actualSourcePath = ctx.SourcePathLinuxAMD64
			} else if targetArch == "linux_aarch64" || targetArch == "arm64" || targetArch == "aarch64" {
				actualSourcePath = ctx.SourcePathLinuxAArch64
			}

			if actualSourcePath == "" {
				return fmt.Errorf("production mode could not find source script path for architecture %s", targetArch)
			}

			sourceInfo, err := os.Stat(actualSourcePath)
			if err != nil {
				return fmt.Errorf("production mode failed to inspect source script %q for %s: %w", actualSourcePath, targetArch, err)
			}
			sourceMode := sourceInfo.Mode().Perm()
			if sourceMode&0111 == 0 {
				sourceMode |= 0755
			}
			if err := copyFile(actualSourcePath, filepath.Join(archAppDir, ctx.BundledSourceName), sourceMode); err != nil {
				return fmt.Errorf("failed bundling source script for %s: %w", targetArch, err)
			}
			if err := os.WriteFile(filepath.Join(archAppDir, "runtime_defaults.json"), ctx.DefaultsJSON, 0644); err != nil {
				return fmt.Errorf("failed writing production runtime defaults: %w", err)
			}

			// Copy icon file to production staging area if it exists
			iconPath := ctx.IconPath
			if iconPath != "" {
				if iconInfo, err := os.Stat(iconPath); err == nil && !iconInfo.IsDir() {
					iconExt := strings.ToLower(filepath.Ext(iconPath))
					if iconExt == "" {
						iconExt = ".png"
					}
					iconDest := filepath.Join(archPayloadDir, "usr", "share", "icons", "hicolor", "256x256", "apps", ctx.PackageName+iconExt)
					if err := copyFile(iconPath, iconDest, 0644); err != nil {
						return fmt.Errorf("failed copying production icon: %w", err)
					}
				}
			}

			desktopDir := filepath.Join(archPayloadDir, "usr", "share", "applications")
			if err := os.MkdirAll(desktopDir, 0755); err != nil {
				return err
			}

			desktopPath := filepath.Join(desktopDir, ctx.PackageName+".desktop")
			installedBinaryPath := filepath.Join("/opt", ctx.PackageName, ctx.PackageName)

			summary := ctx.RootConfig.Deploy.Summary
			if summary == "" {
				summary = ctx.RootConfig.Config.Title
			}

			desktopContent := fmt.Sprintf("[Desktop Entry]\nVersion=1.0\nType=Application\nName=%s\nComment=%s\nIcon=%s\nExec=%s\nTerminal=false\nCategories=Utility;\nActions=Settings;\n\n[Desktop Action Settings]\nName=⚙️ Edit Runtime Defaults\nExec=%s --settings\n",
				ctx.RootConfig.Config.Title, summary, ctx.IconName, quoteDesktopExec(installedBinaryPath), quoteDesktopExec(installedBinaryPath))

			if err := os.WriteFile(desktopPath, []byte(desktopContent), 0644); err != nil {
				return err
			}

			binDir := filepath.Join(archPayloadDir, "usr", "bin")
			if err := os.MkdirAll(binDir, 0755); err != nil {
				return err
			}

			binLink := filepath.Join(binDir, ctx.PackageName)
			_ = os.Remove(binLink)
			relativeBinPath := filepath.Join("..", "..", "opt", ctx.PackageName, ctx.PackageName)
			if err := os.Symlink(relativeBinPath, binLink); err != nil {
				return err
			}

			localRunner := filepath.Join(archStageDir, "run-local.sh")
			localRunnerContent := fmt.Sprintf("#!/usr/bin/env sh\nDIR=$(CDPATH= cd -- \"$(dirname -- \"$0\")\" && pwd)\nexec \"$DIR/payload/opt/%s/%s\" \"$@\"\n", ctx.PackageName, ctx.PackageName)
			if err := os.WriteFile(localRunner, []byte(localRunnerContent), 0755); err != nil {
				return err
			}

			if err := writeProductionPackageFiles(archStageDir, ctx.PackageName, ctx.RootConfig.Config.Title, ctx.IconName, ctx.RootConfig.Deploy, targetArch, ctx.RootConfig.Config.SourceType == "python", ctx.RootConfig.Config.SourceType == "javascript", ctx.RootConfig.Config.SourceType == "powershell"); err != nil {
				return fmt.Errorf("failed writing package metadata for %s: %w", targetArch, err)
			}

			logger.Info(ctx.LogLevel, "Production %s package successfully staged at %s", targetArch, archStageDir)

			// Update this function call to pass TargetBuilds
			if err := executePackageBuilds(archStageDir, targetArch, ctx.TargetBuilds, ctx.LogLevel); err != nil {
				return err
			}

		} else {
			// --- DEBUG MODE DESKTOP SHORTCUT ---
			appsDir := filepath.Join(os.Getenv("HOME"), ".local", "share", "applications")
			_ = os.MkdirAll(appsDir, 0755)

			absBinaryPath, err := filepath.Abs(archOutputBinary)
			if err != nil {
				return fmt.Errorf("failed to determine absolute path for binary: %w", err)
			}

			desktopFilename := fmt.Sprintf("cpp_qt_%s_%s.desktop", ctx.RootConfig.Config.UUID, ctx.PackageName)
			debugDesktopPath := filepath.Join(appsDir, desktopFilename)

			summary := ctx.RootConfig.Deploy.Summary
			if summary == "" {
				summary = ctx.RootConfig.Config.Title
			}

			// For debug mode, use the absolute icon path if available, otherwise fall back to icon name
			iconPathForDesktop := ctx.IconPath
			if iconPathForDesktop == "" {
				iconPathForDesktop = ctx.IconName
			}
			desktopContent := fmt.Sprintf("[Desktop Entry]\nVersion=1.0\nType=Application\nName=%s\nComment=%s\nIcon=%s\nExec='%s'\nTerminal=false\nCategories=Utility;\nActions=Settings;\n\n[Desktop Action Settings]\nName=⚙️ Edit Runtime Defaults\nExec='%s' --settings\n",
				ctx.RootConfig.Config.Title, summary, iconPathForDesktop, absBinaryPath, absBinaryPath)

			_ = os.WriteFile(debugDesktopPath, []byte(desktopContent), 0644)
		}
	}

	return nil
}

func buildWindowsGUI(ctx BuildContext) error {
	engine, err := setup_utils.SetupWindowsQtGUIContainer(ctx.LogLevel)
	if err != nil {
		return err
	}

	outputDir := filepath.Join(ctx.BaseDir, "build_output", "windows_amd64")
	outputName := "custom_toolkit_app_windows_amd64.exe"
	if ctx.CompileMode == "production" {
		stageDir := filepath.Join(ctx.BaseDir, "dist", ctx.PackageName+"_windows_amd64")
		outputDir = filepath.Join(stageDir, "build")
		outputName = ctx.PackageName + ".exe"
		logger.Info(ctx.LogLevel, "Cleaning Windows staging directories for requested target...")
		for _, dir := range []string{"payload", "build"} {
			if err := os.RemoveAll(filepath.Join(stageDir, dir)); err != nil {
				return fmt.Errorf("clear previous Windows %s directory: %w", dir, err)
			}
		}
	}
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return fmt.Errorf("create Windows build directory: %w", err)
	}
	if err := copyWindowsIcon(ctx.IconPath, outputDir); err != nil {
		return err
	}
	sourceBundleName := ""
	if ctx.CompileMode == "production" {
		if ctx.SourcePathWindowsAMD64 == "" {
			return fmt.Errorf("production Windows build has no source_script_windows_amd64 configured")
		}
		sourceBundleName = windowsBundledSourceName(ctx.BundledSourceName, ctx.RootConfig.Config.SourceType)
		if err := copyFile(ctx.SourcePathWindowsAMD64, filepath.Join(outputDir, sourceBundleName), 0755); err != nil {
			return fmt.Errorf("stage Windows script executable next to the GUI build: %w", err)
		}
		if err := os.WriteFile(filepath.Join(outputDir, "runtime_defaults.json"), ctx.DefaultsJSON, 0644); err != nil {
			return fmt.Errorf("write Windows build runtime defaults: %w", err)
		}
	}
	sourcePath := filepath.Join(outputDir, "generated_app.cpp")
	if err := os.WriteFile(sourcePath, []byte(ctx.FinalSourceCode), 0644); err != nil {
		return fmt.Errorf("write generated Windows C++ source: %w", err)
	}

	workspaceDir, err := filepath.Abs(outputDir)
	if err != nil {
		return fmt.Errorf("resolve Windows build directory: %w", err)
	}
	uid, gid := os.Getuid(), os.Getgid()
	args := []string{"run", "--rm", "--platform", "linux/amd64", "--user", fmt.Sprintf("%d:%d", uid, gid)}
	if engine == "podman" {
		args = append(args, "--userns=keep-id")
	}
	args = append(args, "-v", workspaceDir+":/workspace:Z", "-w", "/workspace", setup_utils.WindowsQtGUIImage(), "sh", "-c", windowsQtContainerBuildScript, "sh", filepath.Base(sourcePath), outputName)
	compileCmd := exec.Command(engine, args...)
	compileCmd.Stdout, compileCmd.Stderr = os.Stdout, os.Stderr
	if err := compileCmd.Run(); err != nil {
		return fmt.Errorf("Windows Qt compilation in Fedora container failed: %w", err)
	}

	if ctx.CompileMode == "production" {
		payloadApp := filepath.Join(ctx.BaseDir, "dist", ctx.PackageName+"_windows_amd64", "payload", ctx.PackageName)
		if err := os.MkdirAll(payloadApp, 0755); err != nil {
			return fmt.Errorf("create Windows application payload: %w", err)
		}
		if err := copyFile(filepath.Join(outputDir, outputName), filepath.Join(payloadApp, outputName), 0755); err != nil {
			return fmt.Errorf("stage Windows GUI executable: %w", err)
		}
		if err := copyFile(filepath.Join(outputDir, sourceBundleName), filepath.Join(payloadApp, sourceBundleName), 0755); err != nil {
			return fmt.Errorf("stage Windows script executable: %w", err)
		}
		if err := copyFile(filepath.Join(outputDir, "runtime_defaults.json"), filepath.Join(payloadApp, "runtime_defaults.json"), 0644); err != nil {
			return fmt.Errorf("write Windows runtime defaults: %w", err)
		}
		if err := copyWindowsIcon(ctx.IconPath, payloadApp); err != nil {
			return err
		}
		if err := copyWindowsContainerRuntime(outputDir, payloadApp); err != nil {
			return err
		}
	} else {
		if err := copyWindowsContainerRuntime(outputDir, outputDir); err != nil {
			return err
		}
	}
	logger.Info(ctx.LogLevel, "Built Windows GUI executable: %s", filepath.Join(outputDir, outputName))
	return nil
}

const windowsQtContainerBuildScript = `set -eu
prefix=/usr/x86_64-w64-mingw32/sys-root/mingw
export PKG_CONFIG_PATH="$prefix/lib/pkgconfig"
export PKG_CONFIG_LIBDIR="$prefix/lib/pkgconfig"
flags=$(pkg-config --cflags --libs Qt6Widgets Qt6Network)
resource_object=
if [ -f app_icon.ico ]; then
  printf 'IDI_ICON1 ICON "app_icon.ico"\n' > app_icon.rc
  x86_64-w64-mingw32-windres app_icon.rc -O coff -o app_icon.o
  resource_object=app_icon.o
fi
x86_64-w64-mingw32-g++ -O3 -s -std=c++17 -fvisibility=hidden -ffunction-sections -fdata-sections \
  -static-libgcc -static-libstdc++ "$1" $resource_object $flags -o "$2" -Wl,--gc-sections -mwindows
for file in "$prefix/bin"/*.dll; do
  [ -f "$file" ] && cp "$file" /workspace/
done
mkdir -p /workspace/platforms /workspace/imageformats /workspace/styles
cp "$prefix/lib/qt6/plugins/platforms/qwindows.dll" /workspace/platforms/
for group in imageformats styles; do
  for file in "$prefix/lib/qt6/plugins/$group"/*.dll; do
    [ -f "$file" ] && cp "$file" "/workspace/$group/"
  done
done
for name in libgcc_s_seh-1.dll libstdc++-6.dll libwinpthread-1.dll; do
  file=$(find /usr -path '*mingw*' -name "$name" -print -quit)
  [ -z "$file" ] || cp "$file" /workspace/
done
`

func copyWindowsContainerRuntime(sourceDir, destination string) error {
	dllFiles, err := filepath.Glob(filepath.Join(sourceDir, "*.dll"))
	if err != nil {
		return fmt.Errorf("list Windows runtime DLLs: %w", err)
	}
	for _, source := range dllFiles {
		if err := copyFile(source, filepath.Join(destination, filepath.Base(source)), 0644); err != nil {
			return fmt.Errorf("stage Windows runtime %s: %w", source, err)
		}
	}
	for _, group := range []string{"platforms", "imageformats", "styles"} {
		entries, err := os.ReadDir(filepath.Join(sourceDir, group))
		if err != nil {
			if os.IsNotExist(err) && group != "platforms" {
				continue
			}
			return err
		}
		targetDir := filepath.Join(destination, group)
		if err := os.MkdirAll(targetDir, 0755); err != nil {
			return err
		}
		for _, entry := range entries {
			if entry.IsDir() || strings.ToLower(filepath.Ext(entry.Name())) != ".dll" {
				continue
			}
			if err := copyFile(filepath.Join(sourceDir, group, entry.Name()), filepath.Join(targetDir, entry.Name()), 0644); err != nil {
				return fmt.Errorf("stage Windows Qt %s plugin %s: %w", group, entry.Name(), err)
			}
		}
	}
	for _, name := range []string{"Qt6Core.dll", "Qt6Gui.dll", "Qt6Widgets.dll", "Qt6Network.dll"} {
		if _, err := os.Stat(filepath.Join(destination, name)); err != nil {
			return fmt.Errorf("required Windows Qt runtime %s was not staged: %w", name, err)
		}
	}
	if _, err := os.Stat(filepath.Join(destination, "platforms", "qwindows.dll")); err != nil {
		return fmt.Errorf("required Windows Qt platform plugin was not staged: %w", err)
	}
	return nil
}

func copyWindowsIcon(iconPath, destination string) error {
	for _, ext := range []string{".png", ".ico", ".svg", ".jpg", ".jpeg"} {
		path := filepath.Join(destination, "app_icon"+ext)
		if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
			return fmt.Errorf("remove stale Windows app icon %s: %w", path, err)
		}
	}
	if iconPath == "" {
		return nil
	}
	info, err := os.Stat(iconPath)
	if err != nil || info.IsDir() {
		return nil
	}
	ext := strings.ToLower(filepath.Ext(iconPath))
	if err := copyFile(iconPath, filepath.Join(destination, "app_icon"+ext), 0644); err != nil {
		return err
	}
	if ext == ".png" {
		return writeWindowsICOFromPNG(filepath.Join(destination, "app_icon.png"), filepath.Join(destination, "app_icon.ico"))
	}
	return nil
}

func writeWindowsICOFromPNG(sourcePath, destination string) error {
	source, err := os.Open(sourcePath)
	if err != nil {
		return fmt.Errorf("open Windows PNG icon: %w", err)
	}
	decoded, err := png.Decode(source)
	closeErr := source.Close()
	if err != nil {
		return fmt.Errorf("decode Windows PNG icon: %w", err)
	}
	if closeErr != nil {
		return fmt.Errorf("close Windows PNG icon: %w", closeErr)
	}
	bounds := decoded.Bounds()
	sourceWidth, sourceHeight := bounds.Dx(), bounds.Dy()
	if sourceWidth < 1 || sourceHeight < 1 {
		return fmt.Errorf("Windows PNG icon has invalid dimensions")
	}
	width, height := sourceWidth, sourceHeight
	longest := width
	if height > longest {
		longest = height
	}
	if longest > 256 {
		width = width * 256 / longest
		height = height * 256 / longest
		if width == 0 {
			width = 1
		}
		if height == 0 {
			height = 1
		}
	}
	if width > 256 || height > 256 {
		return fmt.Errorf("Windows icon dimensions exceed the ICO format limit")
	}
	resized := image.NewNRGBA(image.Rect(0, 0, width, height))
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			sourceX := bounds.Min.X + x*sourceWidth/width
			sourceY := bounds.Min.Y + y*sourceHeight/height
			resized.Set(x, y, decoded.At(sourceX, sourceY))
		}
	}
	var pngData bytes.Buffer
	if err := png.Encode(&pngData, resized); err != nil {
		return fmt.Errorf("encode Windows ICO image: %w", err)
	}
	if pngData.Len() > int(^uint32(0))-22 {
		return fmt.Errorf("Windows ICO image is too large")
	}
	icoData := make([]byte, 22+pngData.Len())
	icoData[2] = 1 // ICO resource type.
	icoData[4] = 1 // One image entry.
	if width < 256 {
		icoData[6] = byte(width)
	}
	if height < 256 {
		icoData[7] = byte(height)
	}
	binary.LittleEndian.PutUint16(icoData[10:12], 1)
	binary.LittleEndian.PutUint16(icoData[12:14], 32)
	binary.LittleEndian.PutUint32(icoData[14:18], uint32(pngData.Len()))
	binary.LittleEndian.PutUint32(icoData[18:22], 22)
	copy(icoData[22:], pngData.Bytes())
	if err := os.WriteFile(destination, icoData, 0644); err != nil {
		return fmt.Errorf("write Windows ICO icon: %w", err)
	}
	return nil
}

func isWindowsCompiledSourceType(sourceType string) bool {
	switch strings.ToLower(strings.TrimSpace(sourceType)) {
	case "go", "rust", "binary":
		return true
	default:
		return false
	}
}

func windowsBundledSourceName(sourceName, sourceType string) string {
	if isWindowsCompiledSourceType(sourceType) {
		return strings.TrimSuffix(sourceName, ".exe") + ".exe"
	}
	return sourceName
}
