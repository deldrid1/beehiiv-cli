package auth

import (
	"errors"

	goKeyring "github.com/zalando/go-keyring"
)

type KeyringBackend interface {
	Set(service, user, password string) error
	Get(service, user string) (string, error)
	Delete(service, user string) error
}

type KeyringStore struct {
	service string
	user    string
	backend KeyringBackend
}

type defaultKeyringBackend struct{}

func NewKeyringStore(service, user string) *KeyringStore {
	if service == "" {
		service = DefaultKeyringService
	}
	if user == "" {
		user = DefaultKeyringUser
	}
	return &KeyringStore{
		service: service,
		user:    user,
		backend: defaultKeyringBackend{},
	}
}

func NewKeyringStoreWithBackend(service, user string, backend KeyringBackend) *KeyringStore {
	store := NewKeyringStore(service, user)
	store.backend = backend
	return store
}

func (s *KeyringStore) Load() (SecretRecord, error) {
	value, err := s.backend.Get(s.service, s.user)
	if err != nil {
		if errors.Is(err, goKeyring.ErrNotFound) {
			return SecretRecord{}, ErrSecretNotFound
		}
		return SecretRecord{}, err
	}
	return unmarshalSecret(value)
}

func (s *KeyringStore) Save(record SecretRecord) error {
	value, err := marshalSecret(record)
	if err != nil {
		return err
	}
	return s.backend.Set(s.service, s.user, value)
}

func (s *KeyringStore) Delete() error {
	err := s.backend.Delete(s.service, s.user)
	if errors.Is(err, goKeyring.ErrNotFound) {
		return nil
	}
	return err
}

func (defaultKeyringBackend) Set(service, user, password string) error {
	return goKeyring.Set(service, user, password)
}

func (defaultKeyringBackend) Get(service, user string) (string, error) {
	return goKeyring.Get(service, user)
}

func (defaultKeyringBackend) Delete(service, user string) error {
	return goKeyring.Delete(service, user)
}
