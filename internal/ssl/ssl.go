package ssl

import (
	"context"
	"fmt"
	"net"
	"path"
	"time"

	"github.com/portainer/k2d/internal/logging"
	"github.com/portainer/k2d/pkg/filesystem"
	"github.com/portainer/k2d/pkg/ssl"
)

const (
	SSL_FOLDER    = "ssl"
	CA_FILENAME   = "ca.pem"
	CERT_FILENAME = "cert.pem"
	KEY_FILENAME  = "key.pem"
)

// SSLCAPath constructs and returns the file path of the CA certificate.
// The path is formed by joining the provided data path, the predefined SSL folder,
// and the CA filename.
func SSLCAPath(dataPath string) string {
	return path.Join(dataPath, SSL_FOLDER, CA_FILENAME)
}

// SSLCertPath constructs and returns the file path of the SSL certificate.
// The path is formed by joining the provided data path, the predefined SSL folder,
// and the SSL certificate filename.
func SSLCertPath(dataPath string) string {
	return path.Join(dataPath, SSL_FOLDER, CERT_FILENAME)
}

// SSLKeyPath constructs and returns the file path of the SSL key.
// The path is formed by joining the provided data path, the predefined SSL folder,
// and the SSL key filename.
func SSLKeyPath(dataPath string) string {
	return path.Join(dataPath, SSL_FOLDER, KEY_FILENAME)
}

// EnsureTLSCertificatesExist generates TLS certificates for the provided IP address
// and stores them in a specified directory. If the certificates already exist,
// the function simply verifies their existence and returns.
//
// The function first creates a directory at the provided `dataPath`, if it does not exist.
// It then checks for the existence of the TLS certificates in this directory. If the certificates
// do not exist, the function generates new ones.
//
// Parameters:
// - `dataPath`: The path where the SSL folder and the certificates are (or will be) located.
// - `ipAddr`: The IP address for which the certificates are generated.
//
// It returns an error if any occurs during the directory creation, certificate existence check,
// or certificate generation processes.
//
// The generated certificates have a validity period of 25 years.
//
// This function depends on the ssl.GenerateTLSCertificatesForIPAddr and filesystem.CreateDir functions.
func EnsureTLSCertificatesExist(ctx context.Context, dataPath string, ipAddr net.IP) error {
	certPath := path.Join(dataPath, SSL_FOLDER)

	err := filesystem.CreateDir(certPath)
	if err != nil {
		return fmt.Errorf("unable to create directory %s: %w", certPath, err)
	}

	cfg := ssl.CertConfig{
		Organization: "Portainer.io",
		Country:      "NZ",
		Locality:     "Auckland",
		// 25 years validity
		Validity:     25 * 365 * 24 * time.Hour,
		IpAddr:       ipAddr,
		CertPath:     path.Join(dataPath, SSL_FOLDER),
		CAFilename:   CA_FILENAME,
		CertFilename: CERT_FILENAME,
		KeyFilename:  KEY_FILENAME,
	}

	tlsFilesExist, err := areTLSCertificatesPresent(cfg)
	if err != nil {
		return fmt.Errorf("unable to check if TLS files exist: %w", err)
	}

	if !tlsFilesExist {
		logger := logging.LoggerFromContext(ctx)
		logger.Infow("TLS certificates not found. Generating new ones",
			"ip_address", ipAddr,
		)

		err = ssl.GenerateTLSCertificatesForIPAddr(cfg)
		if err != nil {
			return fmt.Errorf("unable to generate TLS certificates: %w", err)
		}
	}

	return nil
}

func areTLSCertificatesPresent(cfg ssl.CertConfig) (bool, error) {
	files := []string{cfg.CAFilename, cfg.CertFilename, cfg.KeyFilename}

	for _, filename := range files {
		exists, err := filesystem.FileExists(path.Join(cfg.CertPath, filename))
		if err != nil {
			return false, err
		}
		if !exists {
			return false, nil
		}
	}

	return true, nil
}
