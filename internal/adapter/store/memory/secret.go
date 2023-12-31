package memory

import (
	"errors"
	"fmt"
	"strings"
	"sync"

	adaptererr "github.com/portainer/k2d/internal/adapter/errors"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/kubernetes/pkg/apis/core"
)

type secretData struct {
	Data map[string][]byte
	Type string
}

// InMemoryStore is a simple in-memory that can be used
// to store Secrets.
type InMemoryStore struct {
	m         sync.RWMutex
	secretMap map[string]secretData
}

func buildSecretKey(secretName, namespace string) string {
	return fmt.Sprintf("%s-%s", namespace, secretName)
}

func getSecretNameFromKey(key, namespace string) string {
	return strings.TrimPrefix(key, fmt.Sprintf("%s-", namespace))
}

// NewInMemoryStore creates a new in-memory store
// Secrets are stored in a map with the key using a specific format:
// <namespace>-<secretName>
func NewInMemoryStore() *InMemoryStore {
	return &InMemoryStore{
		m:         sync.RWMutex{},
		secretMap: make(map[string]secretData),
	}
}

// DeleteSecret deletes a secret from the in-memory store
func (s *InMemoryStore) DeleteSecret(secretName, namespace string) error {
	s.m.Lock()
	defer s.m.Unlock()
	delete(s.secretMap, buildSecretKey(secretName, namespace))
	return nil
}

// The secret implementation does not support filesystem bindings.
func (s *InMemoryStore) GetSecretBinds(secret *core.Secret) (map[string]string, error) {
	return map[string]string{}, errors.New("in-memory store does not support filesystem bindings")
}

// GetSecret gets a secret from the in-memory store
func (s *InMemoryStore) GetSecret(secretName, namespace string) (*core.Secret, error) {
	s.m.RLock()
	defer s.m.RUnlock()
	data, found := s.secretMap[buildSecretKey(secretName, namespace)]
	if !found {
		return nil, adaptererr.ErrResourceNotFound
	}

	return &core.Secret{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Secret",
			APIVersion: "v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:        secretName,
			Annotations: map[string]string{},
			Namespace:   namespace,
		},
		Data: data.Data,
		Type: core.SecretType(data.Type),
	}, nil
}

// GetSecrets gets all secrets from the in-memory store
func (s *InMemoryStore) GetSecrets(namespace string, selector labels.Selector) (core.SecretList, error) {
	s.m.RLock()
	defer s.m.RUnlock()
	var secrets []core.Secret

	for key, data := range s.secretMap {

		if !strings.HasPrefix(key, namespace) {
			continue
		}

		secret := core.Secret{
			TypeMeta: metav1.TypeMeta{
				Kind:       "Secret",
				APIVersion: "v1",
			},
			ObjectMeta: metav1.ObjectMeta{
				Name:        getSecretNameFromKey(key, namespace),
				Annotations: map[string]string{},
				Namespace:   namespace,
			},
			Data: data.Data,
			Type: core.SecretType(data.Type),
		}

		secrets = append(secrets, secret)
	}

	return core.SecretList{
		Items: secrets,
	}, nil
}

// StoreSecret stores a secret in the in-memory store
func (s *InMemoryStore) StoreSecret(secret *corev1.Secret) error {
	s.m.Lock()
	defer s.m.Unlock()

	s.secretMap[buildSecretKey(secret.Name, secret.Namespace)] = secretData{
		Data: secret.Data,
		Type: string(secret.Type),
	}

	return nil
}
