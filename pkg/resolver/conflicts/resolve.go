package conflicts

import (
	"cpp-package-manager/pkg/types"
	"fmt"

	"github.com/Masterminds/semver/v3"
)

// ResolveConflicts resolves version conflicts for discovered dependencies.
func ResolveConflicts(discovered *types.DiscoveryResult, resolveVersion func(url, versionConstraint string) (string, string, string, error)) (map[string]types.LockedDependency, error) {
	finalDeps := make(map[string]types.LockedDependency)
	for name, constraints := range discovered.Constraints {
		url := discovered.Urls[name]
		fmt.Printf("  - Resolving constraints for %s: %v\n", name, constraints)

		bestVersions := make([]*semver.Version, 0)
		for _, cons := range constraints {
			resolved, _, _, err := resolveVersion(url, cons)
			if err != nil {
				return nil, err
			}
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
		_, commit, _, err := resolveVersion(url, finalVersionString)
		if err != nil {
			return nil, err
		}
		finalDeps[name] = types.LockedDependency{
			URL:     url,
			Version: finalVersionString,
			Commit:  commit,
		}
	}
	return finalDeps, nil
}
