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
	"strings"

	"guify_my_script/logger"
)

// helper to check for slice inclusion and avoid duplicates
func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}

// writeProductionPackageFiles generates the necessary staging files for Arch, Debian, and Fedora.
func writeProductionPackageFiles(stageDir, packageName, title, iconName string, deploy DeployConfig, targetArch string, requiresPython bool, requiresJavascript bool) error {
	debDir := filepath.Join(stageDir, "debian")
	fedDir := filepath.Join(stageDir, "fedora")
	archDir := filepath.Join(stageDir, "arch")

	// --- ARCHITECTURE MAPPINGS ---
	debArch := "amd64"
	archLinuxArch := "x86_64"
	rpmArch := "x86_64"

	if targetArch == "arm64" {
		debArch = "arm64"
		archLinuxArch = "aarch64"
		rpmArch = "aarch64"
	}

	// --- DEFAULTS & FALLBACKS ---
	version := deploy.Version
	if version == "" {
		version = "1.0.0"
	}

	release := deploy.ReleaseNumber
	if release == "" {
		release = "1"
	}

	desc := deploy.Description
	if desc == "" {
		desc = title
	}
	descEscaped := strings.ReplaceAll(desc, `"`, `\"`)

	summary := deploy.Summary
	if summary == "" {
		summary = title
	}
	summaryEscaped := strings.ReplaceAll(summary, `"`, `\"`)

	license := deploy.License
	if license == "" {
		license = "custom"
	}

	maintainerName := deploy.Maintainer
	if maintainerName == "" {
		maintainerName = "Generated Package"
	}

	maintainerEmail := deploy.SupportEmail
	if maintainerEmail == "" {
		maintainerEmail = "maintainer@example.invalid"
	}

	maintainerFull := fmt.Sprintf("%s <%s>", maintainerName, maintainerEmail)

	// --- BASE DEPENDENCIES ---
	depsArchList := []string{"qt6-base", "glibc", "gcc-libs", "hicolor-icon-theme", "noto-fonts"}
	depsDebianList := []string{"libqt6widgets6", "libqt6network6", "libc6", "libstdc++6", "hicolor-icon-theme", "fonts-noto"}
	depsFedoraList := []string{"qt6-qtbase-gui", "qt6-qtbase", "hicolor-icon-theme", "google-noto-fonts-all"}
	if requiresPython {
		// Python modules are executed by the generated launcher at runtime.
		depsArchList = append(depsArchList, "python")
		depsDebianList = append(depsDebianList, "python3")
		depsFedoraList = append(depsFedoraList, "python3")
	}
	if requiresJavascript {
		// JavaScript modules are executed by the generated launcher at runtime via Node.js.
		depsArchList = append(depsArchList, "nodejs")
		depsDebianList = append(depsDebianList, "nodejs")
		depsFedoraList = append(depsFedoraList, "nodejs")
	}

	// --- MERGE CUSTOM DEPENDENCIES FROM JSON CONFIG ---
	for _, depGroup := range deploy.Dependencies {
		// Arch / Pacman
		for _, pkg := range append(depGroup.Pacman, depGroup.Arch...) {
			pkg = strings.TrimSpace(pkg)
			if pkg != "" && !contains(depsArchList, pkg) {
				depsArchList = append(depsArchList, pkg)
			}
		}
		// Debian / Apt
		for _, pkg := range append(depGroup.Apt, depGroup.Debian...) {
			pkg = strings.TrimSpace(pkg)
			if pkg != "" && !contains(depsDebianList, pkg) {
				depsDebianList = append(depsDebianList, pkg)
			}
		}
		// Fedora / Dnf
		for _, pkg := range append(depGroup.Dnf, depGroup.Fedora...) {
			pkg = strings.TrimSpace(pkg)
			if pkg != "" && !contains(depsFedoraList, pkg) {
				depsFedoraList = append(depsFedoraList, pkg)
			}
		}
	}

	// --- FORMATTING FOR MANIFEST FILES AND SPEC FILES ---
	depsArchTxt := strings.Join(depsArchList, "\n") + "\n"
	depsDebianTxt := strings.Join(depsDebianList, "\n") + "\n"
	depsFedoraTxt := strings.Join(depsFedoraList, "\n") + "\n"

	// Format PKGBUILD depends array: 'pkg1' 'pkg2' 'pkg3'
	var archQuoted []string
	for _, pkg := range depsArchList {
		archQuoted = append(archQuoted, fmt.Sprintf("'%s'", pkg))
	}
	archPkgbuildDepends := strings.Join(archQuoted, " ")

	// Format Debian control Depends: pkg1, pkg2, pkg3
	debianControlDepends := strings.Join(depsDebianList, ", ")

	// Format Fedora spec Requires entries:
	// Requires: pkg1
	// Requires: pkg2
	var fedoraRequires []string
	for _, pkg := range depsFedoraList {
		fedoraRequires = append(fedoraRequires, fmt.Sprintf("Requires:       %s", pkg))
	}
	fedoraSpecRequiresStr := strings.Join(fedoraRequires, "\n")

	files := map[string]string{
		filepath.Join(archDir, "dependencies.arch.txt"):  depsArchTxt,
		filepath.Join(debDir, "dependencies.debian.txt"): depsDebianTxt,
		filepath.Join(fedDir, "dependencies.fedora.txt"): depsFedoraTxt,

		// ARCH LINUX
		filepath.Join(archDir, "PKGBUILD"): fmt.Sprintf(`# Generated PKGBUILD. Review dependencies before publishing.
pkgname=%[1]s
pkgver=%[2]s
pkgrel=%[3]s
pkgdesc="%[4]s"
arch=('%[6]s')
url="https://example.invalid/%[1]s"
license=('%[5]s')
depends=(%[7]s)
options=('!strip')
source=()
sha256sums=()

package() {
  cp -a "$srcdir/../../payload/opt" "$pkgdir/"
  cp -a "$srcdir/../../payload/usr" "$pkgdir/"
}
`, packageName, version, release, descEscaped, license, archLinuxArch, archPkgbuildDepends),
		filepath.Join(archDir, "build.sh"): `#!/usr/bin/env bash
set -e
echo "Building Arch package using container..."

ENGINE="docker"
if ! command -v docker >/dev/null 2>&1; then ENGINE="podman"; fi

# Persist both the downloaded packages AND the repository metadata databases
$ENGINE run --rm \
  -v "$(pwd)/..:/workspace" \
  -v arch_pacman_cache:/var/cache/pacman/pkg \
  -v arch_pacman_db:/var/lib/pacman/sync \
  -w /workspace/arch \
  archlinux:base-devel \
  bash -c 'ORIG_UID=$(stat -c "%u" /workspace/arch) && \
           ORIG_GID=$(stat -c "%g" /workspace/arch) && \
           pacman -Sy --noconfirm && \
           useradd -mG wheel builduser && \
           echo "%wheel ALL=(ALL) NOPASSWD: ALL" >> /etc/sudoers && \
           chown -R builduser /workspace/arch && \
           su - builduser -c "cd /workspace/arch && makepkg -cfsA --noconfirm" && \
           chown -R $ORIG_UID:$ORIG_GID /workspace/arch'

echo "Done! Package is in $(pwd)"
`,

		// DEBIAN
		filepath.Join(debDir, "DEBIAN", "control"): fmt.Sprintf(`Package: %[1]s
Version: %[2]s
Section: utils
Priority: optional
Architecture: %[5]s
Maintainer: %[3]s
Depends: %[6]s
Description: %[4]s
`, packageName, version, maintainerFull, desc, debArch, debianControlDepends),
		filepath.Join(debDir, "build.sh"): fmt.Sprintf(`#!/usr/bin/env bash
set -e
echo "Building Debian package using container..."

ENGINE="docker"
if ! command -v docker >/dev/null 2>&1; then ENGINE="podman"; fi

# Dynamically detect the mapped owner to gracefully handle both rootless and rootful engines.
# (Debian uses native tools pre-installed in the slim image, so no package cache volume is needed).
$ENGINE run --rm \
  -v "$(pwd)/..:/workspace" \
  -w /workspace/debian \
  debian:bookworm-slim \
  bash -c 'ORIG_UID=$(stat -c "%%u" /workspace/debian) && \
           ORIG_GID=$(stat -c "%%g" /workspace/debian) && \
           mkdir -p build_root && cp -a ../payload/* build_root/ && cp -a DEBIAN build_root/ && \
           dpkg-deb --root-owner-group --build build_root "%[1]s_%[2]s_%[3]s.deb" && \
           rm -rf build_root && \
           chown $ORIG_UID:$ORIG_GID *.deb'

echo "Done! Package is in $(pwd)"
`, packageName, version, debArch),

		// FEDORA
		filepath.Join(fedDir, packageName+".spec"): fmt.Sprintf(`%%global __os_install_post %%{nil}

Name:           %[1]s
Version:        %[2]s
Release:        %[3]s%%{?dist}
Summary:        %[4]s
License:        %[5]s

%[7]s

%%description
%[6]s

%%install
mkdir -p "%%{buildroot}/opt" "%%{buildroot}/usr"
cp -a "%%{_sourcedir}"/opt/* "%%{buildroot}/opt/"
cp -a "%%{_sourcedir}"/usr/* "%%{buildroot}/usr/"

%%files
/opt/%[1]s
/usr/share/applications/%[1]s.desktop
/usr/share/icons/hicolor/*/apps/%[1]s.*
/usr/bin/%[1]s
`, packageName, version, release, summaryEscaped, license, descEscaped, fedoraSpecRequiresStr),
		filepath.Join(fedDir, "build.sh"): fmt.Sprintf(`#!/usr/bin/env bash
set -e
echo "Building Fedora RPM package using container..."

ENGINE="docker"
if ! command -v docker >/dev/null 2>&1; then ENGINE="podman"; fi

# DNF5 (Fedora 41+) uses /var/cache/libdnf5. Map the volume there to persist metadata and packages.
$ENGINE run --rm \
  -v "$(pwd)/..:/workspace" \
  -v fedora_dnf_cache:/var/cache/libdnf5 \
  -w /workspace/fedora \
  fedora:latest \
  bash -c 'ORIG_UID=$(stat -c "%%u" /workspace/fedora) && \
           ORIG_GID=$(stat -c "%%g" /workspace/fedora) && \
           dnf install -y --setopt=keepcache=1 --setopt=metadata_expire=86400 rpm-build && \
           mkdir -p rpmbuild/{BUILD,RPMS,SOURCES,SPECS,SRPMS,DB} && \
           rpmbuild -bb --quiet --target %[2]s "%[1]s.spec" --nodeps --define "_topdir /workspace/fedora/rpmbuild" --define "_sourcedir /workspace/payload" --define "_dbpath /workspace/fedora/rpmbuild/DB" && \
           mv rpmbuild/RPMS/*/*.rpm . && \
           rm -rf rpmbuild && \
           chown $ORIG_UID:$ORIG_GID *.rpm'

echo "Done! Package is in $(pwd)"
`, packageName, rpmArch),

		// README
		filepath.Join(stageDir, "README-packaging.txt"): fmt.Sprintf(`%[1]s production staging

The files are separated by distribution to prevent cross-contamination.

Shared Payload:
  payload/opt/%[1]s/%[1]s
  payload/opt/%[1]s/<bundled source script or binary>
  payload/usr/share/applications/%[1]s.desktop
  payload/usr/bin/%[1]s

Building Packages:
  Debian: cd debian && ./build.sh
  Fedora: cd fedora && ./build.sh
  Arch:   cd arch && ./build.sh

Local run:
  payload/run-local.sh
`, packageName),
	}

	for path, content := range files {
		if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
			return err
		}

		mode := os.FileMode(0644)
		if strings.HasSuffix(path, "build.sh") {
			mode = 0755
		}

		if err := os.WriteFile(path, []byte(content), mode); err != nil {
			return err
		}
	}

	return nil
}

// executePackageBuilds iterates through the staging directories and executes the build scripts.
func executePackageBuilds(stageDir, targetArch, targetBuilds string, logLevel logger.LogLevel) error {
	logger.Info(logLevel, "🚀 Launching automated package builds for %s...", targetArch)
	packageManagers := []string{"arch", "debian", "fedora"}

	// Determine which package managers are allowed based on the targetBuilds CSV
	allowedPMs := make(map[string]bool)
	if targetBuilds == "" {
		// Empty string means build for all
		for _, pm := range packageManagers {
			allowedPMs[pm] = true
		}
	} else {
		// Initialize a CSV reader to properly parse quotes and enclosed commas
		reader := csv.NewReader(strings.NewReader(targetBuilds))
		reader.TrimLeadingSpace = true

		// Read parses a single line of comma-separated values
		builds, err := reader.Read()
		if err != nil {
			// If the string is entirely empty, safely proceed with an empty slice
			if err.Error() != "EOF" {
				logger.Error(logLevel, "Failed to parse target builds CSV: %v", err)
				return err
			}
		}

		// Clean and evaluate the properly parsed values
		for _, b := range builds {
			cleanBuild := strings.TrimSpace(b)

			if cleanBuild == "" {
				continue
			}

			// Map specific requested file extensions to their respective directories
			if cleanBuild == ".pkg.tar.zst_"+targetArch {
				allowedPMs["arch"] = true
			} else if cleanBuild == ".deb_"+targetArch {
				allowedPMs["debian"] = true
			} else if cleanBuild == ".rpm_"+targetArch {
				allowedPMs["fedora"] = true
			}
		}
	}

	for _, pm := range packageManagers {
		// Skip if this package manager wasn't requested for the current architecture
		if !allowedPMs[pm] {
			continue
		}

		pmDir := filepath.Join(stageDir, pm)
		buildScript := filepath.Join(pmDir, "build.sh")

		// Check if the package manager folder and build script exist
		if _, err := os.Stat(buildScript); err == nil {
			logger.Info(logLevel, "\n❯ Executing %s build for %s...", strings.Title(pm), targetArch)

			cmd := exec.Command("./build.sh")
			cmd.Dir = pmDir
			// Pipe the output directly to the current terminal window
			cmd.Stdout = os.Stdout
			cmd.Stderr = os.Stderr

			if err := cmd.Run(); err != nil {
				// Return error to halt pipeline if a package build chokes
				return fmt.Errorf("automated build failed for %s (%s): %w", pm, targetArch, err)
			}
			logger.Info(logLevel, "✅ Finished %s build for %s successfully!", strings.Title(pm), targetArch)
		}
	}
	return nil
}
