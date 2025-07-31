// File: cpp-package-manager/pkg/git/git.go
package git

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// runGitCommand executes a git command. If progress is not nil, it streams stderr.
// Otherwise, it returns the combined output.
func runGitCommand(dir string, progress io.Writer, args ...string) (string, error) {
	cmd := exec.Command("git", args...)
	if dir != "" {
		cmd.Dir = dir
	}

	// If a progress writer is provided, stream stderr to it.
	if progress != nil {
		cmd.Stderr = progress
		// Use cmd.Run() when streaming, as we don't need to capture output.
		err := cmd.Run()
		if err != nil {
			return "", fmt.Errorf("git command failed: %w", err)
		}
		return "", nil
	}

	// Original behavior: capture all output.
	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("git error: %s\n%s", err, output)
	}
	return strings.TrimSpace(string(output)), nil
}

// Clone clones a repository from a URL to a destination path, showing progress.
func Clone(url, dest string, progress io.Writer) error {
	// Add --progress flag to ensure git prints progress information.
	_, err := runGitCommand("", progress, "clone", "--progress", url, dest)
	return err
}

// Checkout switches the repository at a given path to a specific tag or commit.
func Checkout(repoPath, ref string) error {
	_, err := runGitCommand(repoPath, nil, "checkout", ref)
	return err
}

// ListTags lists all tags in a given repository.
func ListTags(repoPath string) ([]string, error) {
	output, err := runGitCommand(repoPath, nil, "tag", "-l")
	if err != nil {
		return nil, err
	}
	if output == "" {
		return []string{}, nil
	}
	return strings.Split(output, "\n"), nil
}

// GetCommitHash resolves a tag/branch to its full commit SHA.
func GetCommitHash(repoPath, ref string) (string, error) {
	// Fetch latest tags from remote before resolving
	_, err := runGitCommand(repoPath, nil, "fetch", "--all", "--tags")
	if err != nil {
		return "", fmt.Errorf("could not fetch tags: %w", err)
	}
	// Using "tags/" prefix is a robust way to reference a tag
	return runGitCommand(repoPath, nil, "rev-parse", "tags/"+ref)
}

// CopyDir recursively copies a directory from src to dst.
func CopyDir(src, dst string) error {
	return filepath.Walk(src, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		relPath, err := filepath.Rel(src, path)
		if err != nil {
			return err
		}
		dstPath := filepath.Join(dst, relPath)
		if info.IsDir() {
			return os.MkdirAll(dstPath, info.Mode())
		}
		srcFile, err := os.Open(path)
		if err != nil {
			return err
		}
		defer srcFile.Close()
		dstFile, err := os.Create(dstPath)
		if err != nil {
			return err
		}
		defer dstFile.Close()
		if _, err := io.Copy(dstFile, srcFile); err != nil {
			return err
		}
		return os.Chmod(dstPath, info.Mode())
	})
}
