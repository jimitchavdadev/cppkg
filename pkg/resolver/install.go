// File: cpp-package-manager/pkg/resolver/install.go
package resolver

import (
	"cpp-package-manager/pkg/config"
	"cpp-package-manager/pkg/git"
	"cpp-package-manager/pkg/types"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/Masterminds/semver/v3"
)

// AddNewPackage handles 'install <url#version>'
func AddNewPackage(pkgStr string) error {
	cfg, err := config.LoadConfig()
	if err != nil {
		return fmt.Errorf("could not load cppkg.json, did you run 'cppkg init'?: %w", err)
	}
	parts := strings.Split(pkgStr, "#")
	if len(parts) != 2 {
		return fmt.Errorf("invalid package format. Use 'url#version', e.g., 'https://github.com/user/repo.git#^1.0.0'")
	}
	url, version := parts[0], parts[1]
	name := strings.TrimSuffix(filepath.Base(url), ".git")
	if cfg.Dependencies == nil {
		cfg.Dependencies = make(map[string]string)
	}
	cfg.Dependencies[name] = fmt.Sprintf("%s#%s", url, version)
	return config.SaveConfig(cfg)
}

// UninstallPackage removes a dependency and re-resolves the tree.
func UninstallPackage(name string) error {
	fmt.Printf("Uninstalling %s...\n", name)
	cfg, err := config.LoadConfig()
	if err != nil {
		return err
	}

	if _, ok := cfg.Dependencies[name]; !ok {
		return fmt.Errorf("package %s not found in cppkg.json", name)
	}

	delete(cfg.Dependencies, name)
	if err := config.SaveConfig(cfg); err != nil {
		return fmt.Errorf("failed to update cppkg.json: %w", err)
	}

	fmt.Println("Re-resolving dependencies after uninstall...")
	return InstallDependencies(false)
}

// InstallDependencies is the new entry point for installation.
func InstallDependencies(isUpgrade bool) error {
	if err := os.RemoveAll(config.GetModulesDir()); err != nil {
		return fmt.Errorf("failed to clean modules directory: %w", err)
	}

	if isUpgrade {
		fmt.Println("Checking for new package versions...")
	} else {
		fmt.Println("Resolving dependency graph...")
	}

	discovered, err := discoverAllDependencies()
	if err != nil {
		return fmt.Errorf("failed during dependency discovery: %w", err)
	}

	finalDeps, err := resolveConflicts(discovered)
	if err != nil {
		return fmt.Errorf("failed during version resolution: %w", err)
	}

	if err := os.MkdirAll(config.GetModulesDir(), 0755); err != nil {
		return fmt.Errorf("failed to create modules directory: %w", err)
	}

	fmt.Println("Installing packages...")
	newLockFile := &types.LockFile{Dependencies: make(map[string]types.LockedDependency)}
	for name, dep := range finalDeps {
		fmt.Printf("  - Installing %s @ %s\n", name, dep.Version)
		if err := installPackage(name, dep.URL, dep.Commit); err != nil {
			return fmt.Errorf("failed to install package %s: %w", name, err)
		}
		newLockFile.Dependencies[name] = dep
	}

	if err := config.SaveLockfile(newLockFile); err != nil {
		return err
	}

	if err := generateCMakeFile(newLockFile); err != nil {
		return fmt.Errorf("failed to generate cmake file: %w", err)
	}

	rootCfg, err := config.LoadConfig()
	if err != nil {
		return err
	}
	if err := runHooks(rootCfg); err != nil {
		return fmt.Errorf("error running post-install hooks: %w", err)
	}

	return nil
}

type discoveryResult struct {
	urls        map[string]string
	constraints map[string][]string
}

func discoverAllDependencies() (*discoveryResult, error) {
	rootCfg, err := config.LoadConfig()
	if err != nil {
		return nil, err
	}

	result := &discoveryResult{
		urls:        make(map[string]string),
		constraints: make(map[string][]string),
	}
	queue := make([]string, 0)
	processed := make(map[string]bool)

	for name, pkgStr := range rootCfg.Dependencies {
		url, constraint := parsePkgStr(pkgStr)
		result.urls[name] = url
		result.constraints[name] = append(result.constraints[name], constraint)
		if !processed[name] {
			queue = append(queue, name)
			processed[name] = true
		}
	}

	for len(queue) > 0 {
		name := queue[0]
		queue = queue[1:]

		tempResolvedVersion, _, tempDir, err := resolveVersion(result.urls[name], result.constraints[name][0])
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
				tUrl, tConstraint := parsePkgStr(tPkgStr)
				result.urls[tName] = tUrl
				result.constraints[tName] = append(result.constraints[tName], tConstraint)
				if !processed[tName] {
					queue = append(queue, tName)
					processed[tName] = true
				}
			}
		}
		os.RemoveAll(tempDir)
	}
	return result, nil
}

func resolveConflicts(discovered *discoveryResult) (map[string]types.LockedDependency, error) {
	finalDeps := make(map[string]types.LockedDependency)
	for name, constraints := range discovered.constraints {
		url := discovered.urls[name]
		fmt.Printf("  - Resolving constraints for %s: %v\n", name, constraints)

		bestVersions := make([]*semver.Version, 0)
		for _, cons := range constraints {
			resolved, _, tempDir, err := resolveVersion(url, cons)
			if err != nil {
				return nil, err
			}
			os.RemoveAll(tempDir)
			v, err := semver.NewVersion(resolved)
			if err != nil {
				return nil, fmt.Errorf("resolved version %s for %s is not a valid semver", resolved, name)
			}
			bestVersions = append(bestVersions, v)
		}

		highestVersion := bestVersions[0]
		for i := 1; i < len(bestVersions); i++ {
			if bestVersions[i].GreaterThan(highestVersion) {
				highestVersion = bestVersions[i]
			}
		}

		finalVersionString := highestVersion.Original()
		_, commit, tempDir, err := resolveVersion(url, finalVersionString)
		if err != nil {
			return nil, err
		}
		os.RemoveAll(tempDir)

		finalDeps[name] = types.LockedDependency{
			URL:     url,
			Version: finalVersionString,
			Commit:  commit,
		}
	}
	return finalDeps, nil
}

func resolveVersion(url, versionConstraint string) (string, string, string, error) {
	tempDir, err := os.MkdirTemp("", "cppkg-resolve-*")
	if err != nil {
		return "", "", "", err
	}

	// For temporary resolution, we don't need a progress bar, so pass nil.
	if err := git.Clone(url, tempDir, nil); err != nil {
		os.RemoveAll(tempDir)
		return "", "", "", err
	}

	tags, err := git.ListTags(tempDir)
	if err != nil {
		os.RemoveAll(tempDir)
		return "", "", "", fmt.Errorf("failed to list tags: %w", err)
	}

	constraint, err := semver.NewConstraint(versionConstraint)
	if err != nil {
		commit, getErr := git.GetCommitHash(tempDir, versionConstraint)
		if getErr != nil {
			os.RemoveAll(tempDir)
			return "", "", "", fmt.Errorf("version '%s' is not a valid semver range and not a valid tag/commit: %w", versionConstraint, getErr)
		}
		return versionConstraint, commit, tempDir, nil
	}

	var bestVersion *semver.Version
	for _, t := range tags {
		v, err := semver.NewVersion(t)
		if err == nil && constraint.Check(v) {
			if bestVersion == nil || v.GreaterThan(bestVersion) {
				bestVersion = v
			}
		}
	}

	if bestVersion == nil {
		os.RemoveAll(tempDir)
		return "", "", "", fmt.Errorf("no version found that satisfies constraint '%s' for %s", versionConstraint, url)
	}

	bestVersionString := bestVersion.Original()
	commit, err := git.GetCommitHash(tempDir, bestVersionString)
	if err != nil {
		os.RemoveAll(tempDir)
		return "", "", "", fmt.Errorf("could not find commit for version %s: %w", bestVersionString, err)
	}
	if err := git.Checkout(tempDir, commit); err != nil {
		os.RemoveAll(tempDir)
		return "", "", "", fmt.Errorf("could not checkout commit %s: %w", commit, err)
	}

	return bestVersionString, commit, tempDir, nil
}

func installPackage(name, url, commit string) error {
	pkgCachePath := filepath.Join(config.GetCacheDir(), fmt.Sprintf("%s-%s", name, commit[:12]))
	pkgDestPath := filepath.Join(config.GetModulesDir(), name)

	if _, err := os.Stat(pkgCachePath); err == nil {
		return git.CopyDir(pkgCachePath, pkgDestPath)
	}

	fmt.Printf("  -> Downloading %s from %s\n", name, url)
	tempDir, err := os.MkdirTemp("", "cppkg-install-*")
	if err != nil {
		return err
	}
	defer os.RemoveAll(tempDir)

	// Clone with progress bar by passing os.Stderr to the clone function.
	if err := git.Clone(url, tempDir, os.Stderr); err != nil {
		return err
	}

	if err := git.Checkout(tempDir, commit); err != nil {
		return err
	}
	if err := os.RemoveAll(filepath.Join(tempDir, ".git")); err != nil {
		return err
	}
	if err := git.CopyDir(tempDir, pkgCachePath); err != nil {
		return fmt.Errorf("failed to copy to cache: %w", err)
	}
	return git.CopyDir(tempDir, pkgDestPath)
}

func generateCMakeFile(lockFile *types.LockFile) error {
	cmakeFilename := "cppkg.cmake"
	var contentBuilder strings.Builder

	contentBuilder.WriteString("# This file is auto-generated by cppkg.\n")
	contentBuilder.WriteString("# Do not edit this file manually.\n\n")
	contentBuilder.WriteString("# Add include directories for all installed dependencies.\n")

	for name := range lockFile.Dependencies {
		includePath := filepath.Join(config.GetModulesDir(), name, "include")
		cmakePath := fmt.Sprintf("include_directories(${CMAKE_CURRENT_SOURCE_DIR}/%s)\n", filepath.ToSlash(includePath))
		contentBuilder.WriteString(cmakePath)
	}

	fmt.Printf("  - Generating %s\n", cmakeFilename)
	return os.WriteFile(cmakeFilename, []byte(contentBuilder.String()), 0644)
}

func runHooks(cfg *types.PackageConfig) error {
	if cfg.Scripts == nil {
		return nil
	}

	postInstallScript, ok := cfg.Scripts["postinstall"]
	if !ok {
		return nil
	}

	fmt.Printf("  - Executing post-install hook: '%s'\n", postInstallScript)
	cmd := exec.Command("sh", "-c", postInstallScript)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	return cmd.Run()
}

func parsePkgStr(pkgStr string) (url, constraint string) {
	parts := strings.Split(pkgStr, "#")
	return parts[0], parts[1]
}
