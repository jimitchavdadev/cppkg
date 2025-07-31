// File: pkg/types/types.go
package types

// PackageConfig matches the structure of cppkg.json
type PackageConfig struct {
	Name         string            `json:"name"`
	Version      string            `json:"version"`
	Dependencies map[string]string `json:"dependencies"`
	Scripts      map[string]string `json:"scripts,omitempty"` // For post-install hooks
}

// LockFile matches the structure of cppkg.lock
type LockFile struct {
	Dependencies map[string]LockedDependency `json:"dependencies"`
}

// LockedDependency stores the exact version information.
type LockedDependency struct {
	URL     string `json:"url"`
	Version string `json:"version"`
	Commit  string `json:"commit"`
}
