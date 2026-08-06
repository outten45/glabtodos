// Package secrets provides helpers for retrieving secrets from external stores.
package secrets

import (
	"fmt"
	"os/exec"
	"strings"
)

// GitLabToken reads a GitLab token from 1Password using the 1Password CLI.
// The path must be a 1Password secret reference, for example:
// op://Personal/GitLab/API Token.
func GitLabToken(opCommand, path string) (string, error) {
	if strings.TrimSpace(path) == "" {
		return "", fmt.Errorf("1Password secret path is empty")
	}

	out, err := exec.Command(opCommand, "read", path).Output()
	if err != nil {
		return "", err
	}

	token := strings.TrimSpace(string(out))
	if token == "" {
		return "", fmt.Errorf("1Password returned an empty GitLab token")
	}
	return token, nil
}
