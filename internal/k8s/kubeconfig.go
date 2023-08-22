package k8s

import (
	"fmt"
	"os"

	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/tools/clientcmd/api"
)

// GenerateKubeconfig generates a Kubernetes configuration file (kubeconfig) with the provided CA path, server address, and authentication token.
// The function returns the generated kubeconfig as a byte slice and an error if any.
func GenerateKubeconfig(caPath, serverAddr, token string) ([]byte, error) {
	caData, err := os.ReadFile(caPath)
	if err != nil {
		return []byte{}, fmt.Errorf("unable to read TLS CA file: %w", err)
	}

	kubeconfig := api.Config{
		APIVersion: "v1",
		Kind:       "Config",
		Clusters: map[string]*api.Cluster{
			"k2d": {
				Server:                   serverAddr,
				CertificateAuthorityData: caData,
			},
		},
		Contexts: map[string]*api.Context{
			"k2d": {
				Cluster:  "k2d",
				AuthInfo: "k2d-root",
			},
		},
		CurrentContext: "k2d",
		AuthInfos: map[string]*api.AuthInfo{
			"k2d-root": {
				Token: token,
			},
		},
	}

	return clientcmd.Write(kubeconfig)
}
