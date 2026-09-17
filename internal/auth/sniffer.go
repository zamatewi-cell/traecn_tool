package auth

import (
	"os"
	"path/filepath"
	"runtime"
)

// EnvTokenVars lists environment variables checked for a manually provided
// token, in priority order.
var EnvTokenVars = []string{"TRAE_CN_TOKEN", "TRAECN_TOKEN", "TRAE_TOKEN"}

// ideStorageRelPath is the storage.json location relative to the IDE config
// root for each known Trae distribution, in priority order (CN first).
var ideStorageRelPath = []string{
	filepath.Join("Trae CN", "User", "globalStorage", "storage.json"),
	filepath.Join("Trae", "User", "globalStorage", "storage.json"),
}

// configRoot returns the per-OS application config root:
// Windows %APPDATA%, macOS ~/Library/Application Support, Linux ~/.config.
func configRoot() string {
	switch runtime.GOOS {
	case "windows":
		if v := os.Getenv("APPDATA"); v != "" {
			return v
		}
		home, _ := os.UserHomeDir()
		return filepath.Join(home, "AppData", "Roaming")
	case "darwin":
		home, _ := os.UserHomeDir()
		return filepath.Join(home, "Library", "Application Support")
	default: // linux and others follow the XDG base dir spec
		if v := os.Getenv("XDG_CONFIG_HOME"); v != "" {
			return v
		}
		home, _ := os.UserHomeDir()
		return filepath.Join(home, ".config")
	}
}

// StorageCandidates returns all candidate storage.json paths for the current
// OS, ordered by priority, whether or not they exist.
func StorageCandidates() []string {
	root := configRoot()
	paths := make([]string, 0, len(ideStorageRelPath))
	for _, rel := range ideStorageRelPath {
		paths = append(paths, filepath.Join(root, rel))
	}
	return paths
}

// SniffStoragePaths returns candidate storage.json paths that actually exist
// on this machine.
func SniffStoragePaths() []string {
	var found []string
	for _, p := range StorageCandidates() {
		if st, err := os.Stat(p); err == nil && !st.IsDir() {
			found = append(found, p)
		}
	}
	return found
}

// SniffedAccount is an automatically discovered credential source.
type SniffedAccount struct {
	Name   string
	Source CredentialSource
}

// SniffAccounts auto-discovers credential sources on this machine: existing
// IDE storage.json files first, then environment variables.
func SniffAccounts() []SniffedAccount {
	var out []SniffedAccount
	for i, p := range SniffStoragePaths() {
		name := "sniffed"
		if i > 0 {
			name = name + string(rune('0'+i))
		}
		out = append(out, SniffedAccount{
			Name:   name,
			Source: CredentialSource{Type: SourceStorage, StoragePath: p},
		})
	}
	for _, envVar := range EnvTokenVars {
		if os.Getenv(envVar) != "" {
			out = append(out, SniffedAccount{
				Name:   "env:" + envVar,
				Source: CredentialSource{Type: SourceEnv, EnvVar: envVar},
			})
		}
	}
	return out
}

// DefaultStoragePath returns the primary (highest priority) storage.json
// candidate path for the current OS, regardless of existence.
func DefaultStoragePath() string {
	return StorageCandidates()[0]
}
