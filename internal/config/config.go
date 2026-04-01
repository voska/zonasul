package config

import (
	"encoding/json"
	"os"
	"path/filepath"
)

type Config struct {
	CEP          string `json:"cep,omitempty"`
	Street       string `json:"street,omitempty"`
	Number       string `json:"number,omitempty"`
	Complement   string `json:"complement,omitempty"`
	Neighborhood string `json:"neighborhood,omitempty"`
	City         string `json:"city,omitempty"`
	State        string `json:"state,omitempty"`
	OrderFormID  string `json:"orderFormId,omitempty"`
}

type Credentials struct {
	Email string `json:"email,omitempty"`
}

func DefaultDir() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".config", "zonasul")
}

func DefaultPath() string {
	return filepath.Join(DefaultDir(), "config.json")
}

func CredentialsPath() string {
	return filepath.Join(DefaultDir(), "credentials.json")
}

func LoadCredentials() (*Credentials, error) {
	return loadJSON[Credentials](CredentialsPath())
}

func SaveCredentials(c *Credentials) error {
	return saveJSON(CredentialsPath(), c)
}

func loadJSON[T any](path string) (*T, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return new(T), nil
		}
		return nil, err
	}
	var v T
	if err := json.Unmarshal(data, &v); err != nil {
		return nil, err
	}
	return &v, nil
}

func saveJSON(path string, v any) error {
	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		return err
	}
	data, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0600)
}

func Load(path string) (*Config, error) {
	return loadJSON[Config](path)
}

func Save(path string, cfg *Config) error {
	return saveJSON(path, cfg)
}

// Lists maps list names to SKU arrays.
type Lists map[string][]string

func ListsPath() string {
	return filepath.Join(DefaultDir(), "lists.json")
}

func LoadLists() (Lists, error) {
	data, err := os.ReadFile(ListsPath())
	if err != nil {
		if os.IsNotExist(err) {
			return Lists{}, nil
		}
		return nil, err
	}
	var l Lists
	if err := json.Unmarshal(data, &l); err != nil {
		return nil, err
	}
	return l, nil
}

func SaveLists(l Lists) error {
	return saveJSON(ListsPath(), l)
}

func (l Lists) Add(name, sku string) bool {
	for _, s := range l[name] {
		if s == sku {
			return false
		}
	}
	l[name] = append(l[name], sku)
	return true
}

func (l Lists) Remove(name, sku string) bool {
	skus := l[name]
	for i, s := range skus {
		if s == sku {
			l[name] = append(skus[:i], skus[i+1:]...)
			return true
		}
	}
	return false
}
