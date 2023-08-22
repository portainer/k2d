package token

import (
	"encoding/base64"
	"fmt"

	"github.com/portainer/k2d/pkg/filesystem"
	"go.uber.org/zap"
	"k8s.io/apimachinery/pkg/util/uuid"
)

// RetrieveOrCreateEncodedSecret takes a logger, a secret, and a token file path as input parameters.
// If the token file does not exist and no secret is provided, the function generates a new secret using a UUID and encodes it in base64.
// If the token file does exist, the function reads the existing encoded secret from the file.
// If the token file does not exist, the function creates the token file with the encoded secret.
// The function returns the encoded secret as a string, or an error if any file operations fail.
func RetrieveOrCreateEncodedSecret(logger *zap.SugaredLogger, secret, tokenPath string) (string, error) {
	tokenFileExists, err := filesystem.FileExists(tokenPath)
	if err != nil {
		return "", fmt.Errorf("unable to check if token file exists: %w", err)
	}

	// If the token file does not exist and no secret was specified, we generate a new secret
	if !tokenFileExists && secret == "" {
		logger.Debug("token file not found, generating new secret")

		secret = string(uuid.NewUUID())
	}

	// We encode the secret in base64
	encodedSecret := base64.StdEncoding.EncodeToString([]byte(secret))

	// If the token file exists, we use the existing secret (already encoded in base64)
	if tokenFileExists {
		logger.Debug("token file found, using existing secret")

		encodedSecret, err = filesystem.ReadFileAsString(tokenPath)
		if err != nil {
			return "", fmt.Errorf("unable to read token file: %w", err)
		}

		return encodedSecret, nil
	}

	// We then create the token file
	err = filesystem.CreateFileWithDirectories(tokenPath, []byte(encodedSecret))
	if err != nil {
		return "", fmt.Errorf("unable to create token file: %w", err)
	}

	return encodedSecret, nil
}
