// Copyright (c) 2026 Nick Rowley
// All rights reserved.
// Use of this source code is governed by an MIT-style license that can be
// found in the LICENSE file.

package main

import (
	_ "embed"

	"guify_my_script/internal_utils"
)

//go:embed guify_my_script_config.json
var embeddedGuifyConfig []byte

func init() {
	internal_utils.SetGuifyMyScriptConfigJSON(embeddedGuifyConfig)
}
