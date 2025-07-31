package dependency

import (
	"cpp-package-manager/pkg/config"
	"cpp-package-manager/pkg/types"
	"cpp-package-manager/pkg/utils"
	"fmt"
	"os"
	"path/filepath"
)

// DiscoverAllDependenciesWithResolver discovers all dependencies recursively, using the provided resolveVersion function.
func DiscoverAllDependenciesWithResolver(resolveVersion func(url, versionConstraint string) (string, string, string, error)) (*types.DiscoveryResult, error) {
	rootCfg, err := config.LoadConfig()
	if err != nil {
		return nil, err
	}

	result := &types.DiscoveryResult{
		Urls:        make(map[string]string),
		Constraints: make(map[string][]string),
	}
	queue := make([]string, 0)
	processed := make(map[string]bool)

	for name, pkgStr := range rootCfg.Dependencies {
		url, constraint := utils.ParsePkgStr(pkgStr)
		result.Urls[name] = url
		result.Constraints[name] = append(result.Constraints[name], constraint)
		if !processed[name] {
			queue = append(queue, name)
			processed[name] = true
		}
	}

	for len(queue) > 0 {
		name := queue[0]
		queue = queue[1:]

		tempResolvedVersion, _, tempDir, err := resolveVersion(result.Urls[name], result.Constraints[name][0])
		if err != nil {
			return nil, fmt.Errorf("could not temporarily resolve %s: %w", name, err)
		}

		depCfgPath := filepath.Join(tempDir, config.ConfigFile)
		if _, err := os.Stat(depCfgPath); !os.IsNotExist(err) {
			depCfg, err := config.LoadConfigFromPath(depCfgPath)
			if err != nil {
				os.RemoveAll(tempDir)
				return nil, fmt.Errorf("could not read cppkg.json for %s: %w", name, err)
			}
			fmt.Printf("  - Discovered dependencies in %s @ %s...\n", name, tempResolvedVersion)
			for tName, tPkgStr := range depCfg.Dependencies {
				tUrl, tConstraint := utils.ParsePkgStr(tPkgStr)
				result.Urls[tName] = tUrl
				result.Constraints[tName] = append(result.Constraints[tName], tConstraint)
				if !processed[tName] {
					queue = append(queue, tName)
					processed[tName] = true
				}
			}
		}
		os.RemoveAll(tempDir) // Clean up temp clone
	}
	return result, nil
}

// resolveVersion is required by DiscoverAllDependencies, so we forward-declare it here for now.
var resolveVersion func(url, versionConstraint string) (string, string, string, error)
