// File: pkg/utils/utils.go
package utils

import "strings"

// ParsePkgStr splits a package string of the form 'url#version' into url and version/constraint.
func ParsePkgStr(pkgStr string) (url, constraint string) {
	parts := strings.Split(pkgStr, "#")
	return parts[0], parts[1]
}
