package config

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/viper"
	"github.com/zalando/go-keyring"
)

const (
	keyringService = "nfn-cli"
	keyringKey     = "api-key"
)

func Init() {
	configDir, err := os.UserConfigDir()
	if err != nil {
		configDir = filepath.Join(os.Getenv("HOME"), ".config")
	}
	nfnDir := filepath.Join(configDir, "nfn")
	_ = os.MkdirAll(nfnDir, 0o700)

	viper.SetConfigName("config")
	viper.SetConfigType("yaml")
	viper.AddConfigPath(nfnDir)

	viper.SetDefault("base_url", "https://nordicfinancialnews.com/api/v1")
	viper.SetDefault("format", "table")

	viper.SetEnvPrefix("NFN")
	viper.AutomaticEnv()

	_ = viper.ReadInConfig() // ignore error if config file doesn't exist
}

// GetAPIKey returns the API key from env, keyring, or error.
func GetAPIKey() (string, error) {
	if key := os.Getenv("NFN_API_KEY"); key != "" {
		return key, nil
	}

	key, err := keyring.Get(keyringService, keyringKey)
	if err != nil {
		return "", fmt.Errorf("no API key found — run 'nfn auth login' to authenticate")
	}

	return key, nil
}

// SetAPIKey stores the API key in the system keyring.
func SetAPIKey(key string) error {
	return keyring.Set(keyringService, keyringKey, key)
}

// DeleteAPIKey removes the API key from the system keyring.
func DeleteAPIKey() error {
	if err := keyring.Delete(keyringService, keyringKey); err != nil {
		if errors.Is(err, keyring.ErrNotFound) {
			return nil // already gone
		}
		return fmt.Errorf("removing API key from keyring: %w", err)
	}
	return nil
}

// GetBaseURL returns the configured API base URL.
func GetBaseURL() string {
	return viper.GetString("base_url")
}

// GetFormat returns the configured output format.
func GetFormat() string {
	return viper.GetString("format")
}
