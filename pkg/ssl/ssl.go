package ssl

import (
	"bytes"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"fmt"
	"math/big"
	"net"
	"os"
	"path"
	"time"
)

// CertConfig is a structure that holds the necessary information for generating
// a self-signed certificate and associated private key. The fields are as follows:
//
// - Organization: The organization that the certificate will be issued to.
// - Country: The country where the organization is located.
// - Locality: The locality where the organization is located.
// - Validity: The duration that the certificate will be valid for.
// - IpAddr: The IP address that the certificate will be issued for.
// - CertPath: The path where the generated certificate and key files will be saved.
// - CAFilename: The filename of the certificate authority's certificate file.
// - CertFilename: The filename of the generated certificate file.
// - KeyFilename: The filename of the generated private key file.
type CertConfig struct {
	Organization string
	Country      string
	Locality     string
	Validity     time.Duration
	IpAddr       net.IP
	CertPath     string
	CAFilename   string
	CertFilename string
	KeyFilename  string
}

// GenerateTLSCertificatesForIPAddr generates a CA certificate, a TLS certificate, and a private key
// for the IP address specified in the CertConfig. The function uses the given CertConfig to configure the
// certificates and determine where to store the generated files.
// It also sets the certificates to be used for both server and client authentication.
func GenerateTLSCertificatesForIPAddr(cfg CertConfig) error {
	ca := &x509.Certificate{
		SerialNumber: big.NewInt(2019),
		Subject: pkix.Name{
			Organization: []string{cfg.Organization},
			Country:      []string{cfg.Country},
			Locality:     []string{cfg.Locality},
		},
		NotBefore:             time.Now(),
		NotAfter:              time.Now().Add(cfg.Validity),
		IsCA:                  true,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth, x509.ExtKeyUsageServerAuth},
		KeyUsage:              x509.KeyUsageDigitalSignature | x509.KeyUsageCertSign,
		BasicConstraintsValid: true,
	}

	caPrivKey, err := rsa.GenerateKey(rand.Reader, 4096)
	if err != nil {
		return fmt.Errorf("unable to generate CA private key: %w", err)
	}

	caBytes, err := x509.CreateCertificate(rand.Reader, ca, ca, &caPrivKey.PublicKey, caPrivKey)
	if err != nil {
		return fmt.Errorf("unable to generate CA TLS certificate: %w", err)
	}

	caPEM := new(bytes.Buffer)
	pem.Encode(caPEM, &pem.Block{
		Type:  "CERTIFICATE",
		Bytes: caBytes,
	})

	caPath := path.Join(cfg.CertPath, cfg.CAFilename)

	caOut, err := os.Create(caPath)
	if err != nil {
		return fmt.Errorf("unable to open %s for writing: %w", caPath, err)
	}

	pem.Encode(caOut, &pem.Block{
		Type:  "CERTIFICATE",
		Bytes: caBytes,
	})

	err = caOut.Close()
	if err != nil {
		return fmt.Errorf("an error occured while closing %s: %w", caPath, err)
	}

	caPrivKeyPEM := new(bytes.Buffer)
	pem.Encode(caPrivKeyPEM, &pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: x509.MarshalPKCS1PrivateKey(caPrivKey),
	})

	cert := &x509.Certificate{
		SerialNumber: big.NewInt(1658),
		Subject: pkix.Name{
			Organization: []string{cfg.Organization},
			Country:      []string{cfg.Country},
			Locality:     []string{cfg.Locality},
		},
		IPAddresses:  []net.IP{cfg.IpAddr, net.IPv6loopback},
		DNSNames:     []string{"kubernetes.default.svc"},
		NotBefore:    time.Now(),
		NotAfter:     time.Now().Add(cfg.Validity),
		SubjectKeyId: []byte{1, 2, 3, 4, 6},
		ExtKeyUsage:  []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth, x509.ExtKeyUsageServerAuth},
		KeyUsage:     x509.KeyUsageDigitalSignature,
	}

	certPrivKey, err := rsa.GenerateKey(rand.Reader, 4096)
	if err != nil {
		return fmt.Errorf("unable to generate TLS certificate private key: %w", err)
	}

	certBytes, err := x509.CreateCertificate(rand.Reader, cert, ca, &certPrivKey.PublicKey, caPrivKey)
	if err != nil {
		return fmt.Errorf("unable to generate TLS certificate: %w", err)
	}

	certificatePath := path.Join(cfg.CertPath, cfg.CertFilename)

	certOut, err := os.Create(certificatePath)
	if err != nil {
		return fmt.Errorf("unable to open %s for writing: %w", certificatePath, err)
	}

	pem.Encode(certOut, &pem.Block{
		Type:  "CERTIFICATE",
		Bytes: certBytes,
	})

	err = certOut.Close()
	if err != nil {
		return fmt.Errorf("an error occured while closing %s: %w", certificatePath, err)
	}

	keyPath := path.Join(cfg.CertPath, cfg.KeyFilename)

	keyOut, err := os.OpenFile(keyPath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0600)
	if err != nil {
		return fmt.Errorf("unable to open %s for writing: %w", keyPath, err)
	}

	privBytes, err := x509.MarshalPKCS8PrivateKey(certPrivKey)
	if err != nil {
		return fmt.Errorf("unable to marshal private key: %w", err)
	}

	pem.Encode(keyOut, &pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: privBytes,
	})

	err = keyOut.Close()
	if err != nil {
		return fmt.Errorf("an error occured while closing %s: %w", keyPath, err)
	}

	return nil
}
