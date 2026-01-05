package ticktick

import (
	"encoding/json"
	"os"
	"path/filepath"
)

type Config struct {
	ClientID     string `json:"client_id"`
	ClientSecret string `json:"client_secret"`
	RedirectURI  string `json:"redirect_uri,omitempty"`
}

func GetConfigDir() (string, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	configDir := filepath.Join(homeDir, ".ticktickup")
	if err := os.MkdirAll(configDir, 0700); err != nil {
		return "", err
	}
	return configDir, nil
}

func SaveToken(token *Token) error {
	configDir, err := GetConfigDir()
	if err != nil {
		return err
	}

	tokenPath := filepath.Join(configDir, "token.json")
	data, err := json.MarshalIndent(token, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(tokenPath, data, 0600)
}

func LoadToken() (*Token, error) {
	configDir, err := GetConfigDir()
	if err != nil {
		return nil, err
	}

	tokenPath := filepath.Join(configDir, "token.json")
	data, err := os.ReadFile(tokenPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}

	var token Token
	if err := json.Unmarshal(data, &token); err != nil {
		return nil, err
	}

	return &token, nil
}

func SaveConfig(config *Config) error {
	configDir, err := GetConfigDir()
	if err != nil {
		return err
	}

	configPath := filepath.Join(configDir, "config.json")
	data, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(configPath, data, 0600)
}

func LoadConfig() (*Config, error) {
	configDir, err := GetConfigDir()
	if err != nil {
		return nil, err
	}

	configPath := filepath.Join(configDir, "config.json")
	data, err := os.ReadFile(configPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}

	var config Config
	if err := json.Unmarshal(data, &config); err != nil {
		return nil, err
	}

	return &config, nil
}

func DeleteToken() error {
	configDir, err := GetConfigDir()
	if err != nil {
		return err
	}

	tokenPath := filepath.Join(configDir, "token.json")
	if err := os.Remove(tokenPath); err != nil && !os.IsNotExist(err) {
		return err
	}
	return nil
}
