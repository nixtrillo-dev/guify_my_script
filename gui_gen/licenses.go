// Copyright (c) 2026 Nick Rowley
// All rights reserved.
// Use of this source code is governed by an MIT-style license that can be
// found in the LICENSE file.

package gui_gen

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
)

const guifyMyScriptUUID = "com.nixtrillo.guifymyscript"

// stageLicenseBundle copies mandatory and platform-specific license files and,
// for generated applications, the source project's own license into the runtime
// LICENSES directory beside the application executable.
func stageLicenseBundle(sourceDir, appDir, appUUID, appTitle, licenseType, sourceScriptPath, platform string) error {
	platform = strings.TrimSpace(platform)
	if platform != "Linux" && platform != "Windows" {
		return fmt.Errorf("unsupported license platform %q", platform)
	}
	licenseDir := filepath.Join(appDir, "LICENSES")
	if err := os.MkdirAll(licenseDir, 0755); err != nil {
		return fmt.Errorf("create license directory: %w", err)
	}
	for _, category := range []string{"Mandatory", platform} {
		if err := os.MkdirAll(filepath.Join(licenseDir, category), 0755); err != nil {
			return fmt.Errorf("create %s license directory: %w", strings.ToLower(category), err)
		}
	}

	isGuifyMyScript := strings.EqualFold(strings.TrimSpace(appUUID), guifyMyScriptUUID)
	err := fs.WalkDir(bundledLicenseFiles, "assets/LICENSES", func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.IsDir() {
			return nil
		}
		relativePath := strings.TrimPrefix(path, "assets/LICENSES/")
		category := strings.SplitN(filepath.ToSlash(relativePath), "/", 2)[0]
		if category != "Mandatory" && category != platform {
			return nil
		}
		if strings.HasPrefix(entry.Name(), ".") {
			return nil
		}
		if isGuifyMyScript && strings.EqualFold(filepath.Base(relativePath), "Guify My Script MIT.txt") {
			return nil
		}
		contents, err := bundledLicenseFiles.ReadFile(path)
		if err != nil {
			return fmt.Errorf("read bundled license %q: %w", path, err)
		}
		destination := filepath.Join(licenseDir, filepath.FromSlash(relativePath))
		if err := os.MkdirAll(filepath.Dir(destination), 0755); err != nil {
			return err
		}
		if err := os.WriteFile(destination, contents, 0644); err != nil {
			return fmt.Errorf("stage bundled license %q: %w", relativePath, err)
		}
		return nil
	})
	if err != nil {
		return fmt.Errorf("walk bundled license catalog: %w", err)
	}

	licenseContents, licensePath, err := findApplicationLicense(sourceDir, sourceScriptPath)
	if err != nil {
		return err
	}
	if licensePath == "" {
		return nil
	}
	if strings.TrimSpace(licenseType) == "" {
		licenseType = detectLicenseType(licenseContents)
	}
	if strings.TrimSpace(licenseType) == "" {
		licenseType = "License"
	}
	appLabel := strings.TrimSpace(appTitle)
	if appLabel == "" {
		appLabel = "Application"
	}
	appLabel = safeLicenseFilenamePart(appLabel)
	licenseType = safeLicenseFilenamePart(strings.TrimSpace(licenseType))
	destination := filepath.Join(licenseDir, "Mandatory", appLabel+" "+licenseType+".txt")
	if err := os.MkdirAll(filepath.Dir(destination), 0755); err != nil {
		return fmt.Errorf("create application license directory: %w", err)
	}
	if err := os.WriteFile(destination, licenseContents, 0644); err != nil {
		return fmt.Errorf("stage application license from %q: %w", licensePath, err)
	}
	return nil
}

func findApplicationLicense(configDir, sourceScriptPath string) ([]byte, string, error) {
	var directories []string
	if sourceScriptPath != "" {
		info, err := os.Stat(sourceScriptPath)
		if err == nil && info.IsDir() {
			directories = append(directories, sourceScriptPath)
		} else {
			directories = append(directories, filepath.Dir(sourceScriptPath))
		}
	}
	directories = append(directories, configDir)

	seen := make(map[string]struct{}, len(directories))
	for _, dir := range directories {
		if dir == "" {
			continue
		}
		absDir, err := filepath.Abs(dir)
		if err == nil {
			dir = absDir
		}
		if _, ok := seen[dir]; ok {
			continue
		}
		seen[dir] = struct{}{}
		for _, name := range []string{"LICENSE.txt", "LICENSE", "LICENSE.md"} {
			path := filepath.Join(dir, name)
			contents, err := os.ReadFile(path)
			if os.IsNotExist(err) {
				continue
			}
			if err != nil {
				return nil, "", fmt.Errorf("read application license %q: %w", path, err)
			}
			return contents, path, nil
		}
	}
	return nil, "", nil
}

func safeLicenseFilenamePart(value string) string {
	return strings.Map(func(r rune) rune {
		if r == '/' || r == '\\' || r == ':' || r == '*' || r == '?' || r == '"' || r == '<' || r == '>' || r == '|' {
			return '-'
		}
		return r
	}, value)
}

func detectLicenseType(contents []byte) string {
	text := strings.ToUpper(string(contents))
	switch {
	case strings.Contains(text, "GNU LESSER GENERAL PUBLIC LICENSE") || strings.Contains(text, "LGPL"):
		return "LGPL"
	case strings.Contains(text, "GNU AFFERO GENERAL PUBLIC LICENSE") || strings.Contains(text, "AGPL"):
		return "AGPL"
	case strings.Contains(text, "GNU GENERAL PUBLIC LICENSE") || strings.Contains(text, "GPL"):
		return "GPL"
	case strings.Contains(text, "APACHE LICENSE"):
		return "Apache"
	case strings.Contains(text, "MIT LICENSE") || strings.Contains(text, "PERMISSION IS HEREBY GRANTED, FREE OF CHARGE"):
		return "MIT"
	case strings.Contains(text, "BSD 3-CLAUSE") || strings.Contains(text, "BSD 3-CLAUSE LICENSE"):
		return "BSD-3-Clause"
	case strings.Contains(text, "BSD 2-CLAUSE") || strings.Contains(text, "BSD 2-CLAUSE LICENSE"):
		return "BSD-2-Clause"
	case strings.Contains(text, "MOZILLA PUBLIC LICENSE"):
		return "MPL"
	case strings.Contains(text, "ISC LICENSE"):
		return "ISC"
	default:
		return ""
	}
}

func configuredLicenseType(deploy DeployConfig) string {
	if strings.TrimSpace(deploy.License) != "" {
		return deploy.License
	}
	return deploy.Licence
}
