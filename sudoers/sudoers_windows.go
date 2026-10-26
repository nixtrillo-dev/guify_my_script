// Copyright (c) 2026 Nick Rowley
// All rights reserved.
// Use of this source code is governed by an MIT-style license that can be
// found in the LICENSE file.

package sudoers

// RunAsSudoUser executes the action directly on Windows. The Linux helper's
// UID/GID switching and pkexec environment handling have no Windows equivalent.
func RunAsSudoUser(action func() error) error {
	return action()
}
