package auth

import (
	"encoding/json"
	"errors"
	"time"
)

const (
	DefaultKeyringService = "beehiiv-cli"
	DefaultKeyringUser    = "default"
)

var ErrSecretNotFound = errors.New("secret not found")

type SecretRecord struct {
	APIKey       string      `json:"api_key,omitempty"`
	OAuth        OAuthSecret `json:"oauth,omitempty"`
	ClientSecret string      `json:"client_secret,omitempty"`
}

type OAuthSecret struct {
	AccessToken  string    `json:"access_token,omitempty"`
	RefreshToken string    `json:"refresh_token,omitempty"`
	TokenType    string    `json:"token_type,omitempty"`
	Scope        string    `json:"scope,omitempty"`
	CreatedAt    time.Time `json:"created_at,omitempty"`
	ExpiresAt    time.Time `json:"expires_at,omitempty"`
}

type Store interface {
	Load() (SecretRecord, error)
	Save(SecretRecord) error
	Delete() error
}

func marshalSecret(record SecretRecord) (string, error) {
	data, err := json.Marshal(record)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

func unmarshalSecret(value string) (SecretRecord, error) {
	var record SecretRecord
	if err := json.Unmarshal([]byte(value), &record); err != nil {
		return SecretRecord{}, err
	}
	return record, nil
}
