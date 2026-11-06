package internal_utils

import (
	"encoding/json"
	"fmt"
)

var guifyConfigJSON []byte

// SetGuifyMyScriptConfigJSON supplies the config embedded by the application
// binary so version lookup does not depend on the process working directory.
func SetGuifyMyScriptConfigJSON(data []byte) {
	guifyConfigJSON = append(guifyConfigJSON[:0], data...)
}

// GetGuifyMyScriptVersion reads deploy.version from the embedded config JSON.
func GetGuifyMyScriptVersion() (string, error) {
	if len(guifyConfigJSON) == 0 {
		return "", fmt.Errorf("Guify My Script config is not embedded in this binary")
	}
	var config struct {
		Deploy struct {
			Version string `json:"version"`
		} `json:"deploy"`
	}
	if err := json.Unmarshal(guifyConfigJSON, &config); err != nil {
		return "", fmt.Errorf("parse embedded Guify My Script config: %w", err)
	}
	if config.Deploy.Version == "" {
		return "", fmt.Errorf("deploy.version is empty in embedded Guify My Script config")
	}
	return config.Deploy.Version, nil
}
