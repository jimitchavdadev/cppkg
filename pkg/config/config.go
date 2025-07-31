// File: cpp-package-manager/pkg/config/config.go
package config

import (
	"encoding/json"
	"os"

	"cpp-package-manager/pkg/types"
)

const (
	// ConfigFile is the name of the package configuration file.
	ConfigFile = "cppkg.json"
	// LockFileName is the name of the lock file.
	LockFileName = "cppkg.lock"
	// ModulesDir is the directory where dependencies are installed.
	ModulesDir = "cpp_modules"
	// CacheDir is the directory where packages are cached.
	CacheDir = ".cppkg_cache"
)

// ...types moved to pkg/types/types.go...

// LoadConfig reads and parses the root cppkg.json from the current directory.
func LoadConfig() (*types.PackageConfig, error) {
	return LoadConfigFromPath(ConfigFile)
}

// LoadConfigFromPath reads and parses a cppkg.json file from a specific path.
func LoadConfigFromPath(path string) (*types.PackageConfig, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var cfg types.PackageConfig
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, err
	}
	// A package might not have any dependencies.
	if cfg.Dependencies == nil {
		cfg.Dependencies = make(map[string]string)
	}
	return &cfg, err
}

// SaveConfig writes the config data to cppkg.json
func SaveConfig(cfg *types.PackageConfig) error {
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(ConfigFile, data, 0644)
}

// LoadLockfile reads and parses cppkg.lock
func LoadLockfile() (*types.LockFile, error) {
	if _, err := os.Stat(LockFileName); os.IsNotExist(err) {
		return &types.LockFile{Dependencies: make(map[string]types.LockedDependency)}, nil
	}
	data, err := os.ReadFile(LockFileName)
	if err != nil {
		return nil, err
	}
	var lock types.LockFile
	err = json.Unmarshal(data, &lock)
	return &lock, err
}

// SaveLockfile writes the lock data to cppkg.lock
func SaveLockfile(lock *types.LockFile) error {
	data, err := json.MarshalIndent(lock, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(LockFileName, data, 0644)
}

// GetModulesDir returns the path to the dependency installation directory.
func GetModulesDir() string {
	return ModulesDir
}

// GetCacheDir returns the path to the cache directory.
func GetCacheDir() string {
	return CacheDir
}
