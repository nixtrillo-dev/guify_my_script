// Copyright (c) 2026 Nick Rowley
// All rights reserved.
// Use of this source code is governed by an MIT-style license that can be
// found in the LICENSE file.

package gui_gen

import (
	"encoding/csv"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	"guify_my_script/logger"
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
	// Detect available container engine
	containerEngine := "docker"
	if _, err := exec.LookPath("docker"); err != nil {
		if _, err := exec.LookPath("podman"); err == nil {
			containerEngine = "podman"
		} else {
			return fmt.Errorf("neither 'docker' nor 'podman' was found in system PATH")
		}
	}

	if ctx.ClearCache {
		logger.Info(ctx.LogLevel, "Clearing build caches and builder images to download dependencies from scratch...")
		// Untag images to force a rebuild from the Dockerfile
		_ = exec.Command(containerEngine, "rmi", "-f", "qt6-amd64-builder", "qt6-arm64-builder").Run()
		// Remove named volumes persisting DNF/Pacman databases and package cache
		_ = exec.Command(containerEngine, "volume", "rm", "-f", "arch_pacman_cache", "arch_pacman_db", "fedora_dnf_cache").Run()
	}

	for _, targetArch := range ctx.TargetArchs {
		logger.Info(ctx.LogLevel, "\n--- Starting build for architecture: %s ---", targetArch)

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

		// 2. Translate paths relative to /workspace mount
		relCppFile, _ := filepath.Rel(ctx.BaseDir, archCppFile)
		relOutputBinary, _ := filepath.Rel(ctx.BaseDir, archOutputBinary)

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
			"%s run --rm --platform %s -v \"%s:/workspace\" -w /workspace %s sh -c "+
				"\"%s -O3 -s -std=c++17 -fPIC -flto=auto -fvisibility=hidden -ffunction-sections -fdata-sections '%s' %s -o '%s' -Wl,--gc-sections\"",
			containerEngine, platform, ctx.BaseDir, builderTag, compilerBin, relCppFile, pkgFlags, relOutputBinary,
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
