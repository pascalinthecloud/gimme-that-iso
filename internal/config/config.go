package config

import (
	"encoding/json"
	"os"
)

type ISO struct {
	Name            string `json:"name"`
	URL             string `json:"url"`
	SignatureURL    string `json:"signature_url"`
	ChecksumFileURL string `json:"checksum_file_url"`
	GPGKeyURL       string `json:"gpg_key_url"`
}

type Config struct {
	ISOs []ISO `json:"isos"`
}

func LoadConfig(filePath string) (*Config, error) {
	configFile, err := os.ReadFile(filePath)
	if err != nil {
		return nil, err
	}

	var config Config
	err = json.Unmarshal(configFile, &config)
	if err != nil {
		return nil, err
	}

	return &config, nil
}
