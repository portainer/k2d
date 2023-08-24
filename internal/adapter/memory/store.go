package memory

import (
	"sync"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/kubernetes/pkg/apis/core"
)

type SecretData struct {
	// Data map[string][]byte `json:"data"`
	Data map[string][]byte
}

type (
	InMemoryStore struct {
		m         sync.RWMutex
		secretMap map[string]SecretData
	}
)

func NewInMemoryStore() *InMemoryStore {
	return &InMemoryStore{
		m:         sync.RWMutex{},
		secretMap: make(map[string]SecretData),
	}
}

func (s *InMemoryStore) DeleteSecret(secretName string) error {
	s.m.Lock()
	defer s.m.Unlock()
	delete(s.secretMap, secretName)
	return nil
}

func (s *InMemoryStore) GetSecret(secretName string) (*core.Secret, error) {
	s.m.RLock()
	defer s.m.RUnlock()
	data, found := s.secretMap[secretName]
	if !found {
		return nil, nil
	}

	return &core.Secret{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Secret",
			APIVersion: "v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:        secretName,
			Annotations: map[string]string{},
			Namespace:   "default",
		},
		Data: data.Data,
		// Type: core.SecretTypeOpaque,
	}, nil
}

func (s *InMemoryStore) GetSecrets(selector labels.Selector) (core.SecretList, error) {
	s.m.RLock()
	defer s.m.RUnlock()
	var secrets []core.Secret

	for name, data := range s.secretMap {

		secret := core.Secret{
			TypeMeta: metav1.TypeMeta{
				Kind:       "Secret",
				APIVersion: "v1",
			},
			ObjectMeta: metav1.ObjectMeta{
				Name:        name,
				Annotations: map[string]string{},
				Namespace:   "default",
			},
			Data: data.Data,
			// Type: core.SecretTypeOpaque,
		}

		secrets = append(secrets, secret)
	}

	return core.SecretList{
		Items: secrets,
	}, nil
}

func (s *InMemoryStore) StoreSecret(secret *corev1.Secret) error {
	s.m.Lock()
	defer s.m.Unlock()

	s.secretMap[secret.Name] = SecretData{
		Data: secret.Data,
	}

	return nil
}
