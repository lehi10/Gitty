package cmd

import (
	"encoding/json"
	"os"
	"path/filepath"
)

const configFileName = ".gttconfig"

type Config struct {
	Trunk string `json:"trunk"`
}

func getRepoRoot() (string, error) {
	output, err := runGitCommand("rev-parse", "--show-toplevel")
	if err != nil {
		return "", err
	}
	return output, nil
}

func getConfigPath() (string, error) {
	root, err := getRepoRoot()
	if err != nil {
		return "", err
	}
	return filepath.Join(root, configFileName), nil
}

func LoadConfig() (*Config, error) {
	configPath, err := getConfigPath()
	if err != nil {
		return nil, err
	}

	data, err := os.ReadFile(configPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil // No config file, return nil
		}
		return nil, err
	}

	var config Config
	if err := json.Unmarshal(data, &config); err != nil {
		return nil, err
	}

	return &config, nil
}

func SaveConfig(config *Config) error {
	configPath, err := getConfigPath()
	if err != nil {
		return err
	}

	data, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(configPath, data, 0644)
}

func ConfigExists() bool {
	configPath, err := getConfigPath()
	if err != nil {
		return false
	}
	_, err = os.Stat(configPath)
	return err == nil
}
