package main

import (
	"encoding/json"
	"fmt"
	"os"
)

// TODO: Consider refactoring this and using dependency injectionx
// We're using global state here only for config, so we don't have to access
// the file over and over, and can share the state across different parts of the app.
// Other than the obvious, an issue with this is testing.
var config RqConfig

type RqConfig struct {
	PermittedFileExtensions string `json:"permitted_file_extensions"`
	UploadDirectory         string `json:"upload_directory"`
}

func LoadConfigFile(profile string) error {
	exeDir, err := os.Getwd()
	if err != nil {
		return err
	}
	configPath := fmt.Sprintf("%v/%v", exeDir, "config.json")

	configFile, err := os.ReadFile(configPath)

	// map of config profiles which can exist within one file
	configs := map[string]RqConfig{}

	if err := json.Unmarshal(configFile, &configs); err != nil {
		return err
	}

	configValue, ok := configs[profile]
	if !ok {
		err := fmt.Errorf("config profile not found: %s", profile)
		return err
	}
	config = configValue
	return nil

}
