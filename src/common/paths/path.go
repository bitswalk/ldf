// Package paths provides common path manipulation utilities for ldf applications.
package paths

import (
	"os"
	"os/user"
	"path/filepath"
	"strings"
)

// Expand expands special path prefixes:
// - ~ expands to the user's home directory
// - Environment variables are expanded via os.ExpandEnv
func Expand(path string) string {
	// First expand environment variables
	path = os.ExpandEnv(path)

	// Then expand ~ to home directory
	if strings.HasPrefix(path, "~/") {
		if usr, err := user.Current(); err == nil {
			return filepath.Join(usr.HomeDir, path[2:])
		}
	} else if path == "~" {
		if usr, err := user.Current(); err == nil {
			return usr.HomeDir
		}
	}

	return path
}

// ExpandHome expands only the ~ prefix to the user's home directory
func ExpandHome(path string) string {
	if strings.HasPrefix(path, "~/") {
		if usr, err := user.Current(); err == nil {
			return filepath.Join(usr.HomeDir, path[2:])
		}
	} else if path == "~" {
		if usr, err := user.Current(); err == nil {
			return usr.HomeDir
		}
	}
	return path
}

// EnsureDir ensures that the directory for the given path exists.
// If the path is a file path, it creates the parent directory.
func EnsureDir(path string) error {
	dir := filepath.Dir(path)
	return os.MkdirAll(dir, 0755)
}

// EnsureDirPath ensures that the given directory path exists.
func EnsureDirPath(dirPath string) error {
	return os.MkdirAll(dirPath, 0755)
}

// Exists returns true if the path exists
func Exists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

// IsDir returns true if the path exists and is a directory
func IsDir(path string) bool {
	info, err := os.Stat(path)
	if err != nil {
		return false
	}
	return info.IsDir()
}

// IsFile returns true if the path exists and is a regular file
func IsFile(path string) bool {
	info, err := os.Stat(path)
	if err != nil {
		return false
	}
	return info.Mode().IsRegular()
}
