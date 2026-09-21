// Copyright (c) 2026 Nick Rowley
// All rights reserved.
// Use of this source code is governed by an MIT-style license that can be
// found in the LICENSE file.

package sudoers

import (
	"fmt"
	"os"
	"os/user"
	"runtime"
	"strconv"
	"syscall"
)

// RunAsSudoUser temporarily drops root privileges, executes the provided function,
// and seamlessly regains root privileges afterward.
func RunAsSudoUser(action func() error) error {
	// Snapshot original environment variables
	origHome, homeSet := os.LookupEnv("HOME")
	origUser, userSet := os.LookupEnv("USER")
	origXdg, xdgSet := os.LookupEnv("XDG_RUNTIME_DIR")
	origDbus, dbusSet := os.LookupEnv("DBUS_SESSION_BUS_ADDRESS")
	origQt, qtSet := os.LookupEnv("QT_QPA_PLATFORM")

	sudoUser := os.Getenv("SUDO_USER")

	// 1. Check for pkexec fallback if SUDO_USER is empty
	if sudoUser == "" {
		pkexecUID := os.Getenv("PKEXEC_UID")
		if pkexecUID != "" {
			u, err := user.LookupId(pkexecUID)
			if err == nil {
				sudoUser = u.Username
			}
		}
	}

	if sudoUser == "" {
		// Not running under sudo or pkexec, just run the action
		return action()
	}

	u, err := user.Lookup(sudoUser)
	if err != nil {
		return fmt.Errorf("failed to lookup user: %w", err)
	}

	uid, _ := strconv.Atoi(u.Uid)
	gid, _ := strconv.Atoi(u.Gid)

	// 2. Lock the Goroutine to the current OS thread
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()

	// 3. Guarantee we switch BACK to root when this function exits
	defer func() {
		// Regain Root Real and Effective UID/GID by calling from the Saved Set-User-ID (0)
		syscall.Setresuid(0, 0, 0)
		syscall.Setresgid(0, 0, 0)

		// Safely restore the exact environment state root started with
		if homeSet {
			os.Setenv("HOME", origHome)
		} else {
			os.Unsetenv("HOME")
		}
		if userSet {
			os.Setenv("USER", origUser)
		} else {
			os.Unsetenv("USER")
		}
		if xdgSet {
			os.Setenv("XDG_RUNTIME_DIR", origXdg)
		} else {
			os.Unsetenv("XDG_RUNTIME_DIR")
		}
		if dbusSet {
			os.Setenv("DBUS_SESSION_BUS_ADDRESS", origDbus)
		} else {
			os.Unsetenv("DBUS_SESSION_BUS_ADDRESS")
		}
		if qtSet {
			os.Setenv("QT_QPA_PLATFORM", origQt)
		} else {
			os.Unsetenv("QT_QPA_PLATFORM")
		}
	}()

	// 4. Temporarily drop to standard user Real and Effective UID/GID
	// We MUST drop the Real UID, otherwise Qt binaries detect a setuid mismatch and core dump.
	if err := syscall.Setresgid(gid, gid, 0); err != nil {
		return fmt.Errorf("failed to temporarily drop GID: %w", err)
	}
	if err := syscall.Setresuid(uid, uid, 0); err != nil {
		return fmt.Errorf("failed to temporarily drop UID: %w", err)
	}

	// 5. Temporarily spoof the environment variables for KDE
	os.Setenv("HOME", u.HomeDir)
	os.Setenv("USER", u.Username)
	os.Setenv("XDG_RUNTIME_DIR", fmt.Sprintf("/run/user/%s", u.Uid))
	os.Setenv("DBUS_SESSION_BUS_ADDRESS", fmt.Sprintf("unix:path=/run/user/%s/bus", u.Uid))
	os.Setenv("QT_QPA_PLATFORM", "offscreen") // Force headless execution to prevent Qt core dumps

	// 6. FAIL-SAFE: Abort if we are still running as root
	if syscall.Geteuid() == 0 {
		return fmt.Errorf("security fail-safe tripped: de-elevation failed, still running as root")
	}

	// 7. Execute the KDE specific code
	return action()
}
