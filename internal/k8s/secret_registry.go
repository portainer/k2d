package k8s

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strings"

	"k8s.io/kubernetes/pkg/apis/core"
)

// dockerConfig represents a part of the docker config json file
type dockerConfig struct {
	Auths map[string]struct {
		Auth string `json:"auth"`
	} `json:"auths"`
}

// GetRegistryAuthFromSecret extracts the username and password for a given registry URL from a Kubernetes Secret.
// The secret is expected to contain a field named ".dockerconfigjson" with the encoded Docker registry configuration.
// The function iterates through the "Auths" section of the Docker config to find the matching registry URL and
// decodes the base64-encoded authentication string to return the username and password.
//
// Parameters:
// secret - Pointer to the Kubernetes Secret object containing the Docker registry configuration.
// registryURL - The URL of the registry for which the credentials are needed.
//
// Returns:
//   - string: The username associated with the registry.
//   - string: The password associated with the registry.
//   - error: An error if the Docker config is not found, if there is a failure in decoding the JSON,
//     if the registry is not found in the Docker config, if the auth string cannot be decoded,
//     or if the auth string is in an invalid format.
func GetRegistryAuthFromSecret(secret *core.Secret, registryURL string) (string, string, error) {
	if _, ok := secret.Data[".dockerconfigjson"]; !ok {
		return "", "", fmt.Errorf("docker config json not found in secret")
	}

	dockerConfigJSON := secret.Data[".dockerconfigjson"]

	var dockerConfig dockerConfig
	if err := json.Unmarshal(dockerConfigJSON, &dockerConfig); err != nil {
		return "", "", fmt.Errorf("unable to decode registry docker config: %w", err)
	}

	registryKey := ""
	for registry := range dockerConfig.Auths {
		if strings.Contains(registry, registryURL) {
			registryKey = registry
		}
	}

	if registryKey == "" {
		return "", "", fmt.Errorf("registry %s not found in docker config", registryURL)
	}

	auth := dockerConfig.Auths[registryKey].Auth
	decodedString, err := base64.StdEncoding.DecodeString(auth)
	if err != nil {
		return "", "", fmt.Errorf("unable to decode auth string: %w", err)
	}

	decodedAuth := string(decodedString)
	authData := strings.Split(decodedAuth, ":")
	if len(authData) != 2 {
		return "", "", fmt.Errorf("invalid auth string: %s", decodedAuth)
	}

	return authData[0], authData[1], nil
}
