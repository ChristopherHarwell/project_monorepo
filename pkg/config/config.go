package config

import (
	"christopherharwell/project_monorepo/pkg/types"
	"encoding/json"
	"os"
)

// LoadConfig reads and parses the configuration file.
// It returns a Config struct populated with the settings from the JSON file.
//
// Parameters:
//   - configFile: Path to the JSON configuration file
//
// Returns:
//   - types.Config: The parsed configuration
//   - error: Any error that occurred during file reading or JSON parsing
//
// Example:
//   cfg, err := LoadConfig("config.json")
//   if err != nil {
//       log.Fatal(err)
//   }
func LoadConfig(configFile string) (types.Config, error) {
	data, err := os.ReadFile(configFile)
	if err != nil {
		return types.Config{}, err
	}

	var cfg types.Config
	err = json.Unmarshal(data, &cfg)
	if err != nil {
		return types.Config{}, err
	}
	return cfg, nil
} 