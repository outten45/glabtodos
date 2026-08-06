package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/pelletier/go-toml/v2"
)

// fileConfig contains settings that may be persisted in a TOML file. Tokens
// are intentionally not supported here; use GLAB_TOKEN or 1Password instead.
type fileConfig struct {
	Host      string `toml:"host"`
	APIPath   string `toml:"api_path"`
	OPPath    string `toml:"op_path"`
	OPCommand string `toml:"op_command"`
	Delay     string `toml:"delay"`
	Notify    string `toml:"notify"`
	Icon      string `toml:"icon"`
}

func defaultConfigPath() (string, error) {
	dir, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "glabtodos", "config.toml"), nil
}

func configArguments(args []string) (path string, explicit bool, disabled bool, err error) {
	for i := 1; i < len(args); i++ {
		arg := args[i]
		switch {
		case arg == "--no-config":
			disabled = true
		case arg == "--config":
			if i+1 >= len(args) {
				return "", false, false, fmt.Errorf("--config requires a path")
			}
			i++
			path, explicit = args[i], true
		case len(arg) > len("--config=") && arg[:len("--config=")] == "--config=":
			path, explicit = arg[len("--config="):], true
		}
	}
	return path, explicit, disabled, nil
}

func loadFileConfig(args []string) (fileConfig, string, error) {
	var cfg fileConfig
	path, explicit, disabled, err := configArguments(args)
	if err != nil {
		return cfg, "", err
	}
	if disabled {
		return cfg, path, nil
	}
	if !explicit {
		path, err = defaultConfigPath()
		if err != nil {
			return cfg, "", err
		}
	}

	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) && !explicit {
			return cfg, path, nil
		}
		return cfg, path, fmt.Errorf("read config file %q: %w", path, err)
	}
	if err := toml.Unmarshal(data, &cfg); err != nil {
		return cfg, path, fmt.Errorf("parse config file %q: %w", path, err)
	}
	return cfg, path, nil
}
