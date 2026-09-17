package workflow

import (
	"os"
	"path/filepath"
)

// EnsureDir ensures directory exists
func EnsureDir(dir string) error {
	return os.MkdirAll(dir, 0755)
}

// FileExists checks if file exists
func FileExists(path string) (bool, error) {
	_, err := os.Stat(path)
	if err == nil {
		return true, nil
	}
	if os.IsNotExist(err) {
		return false, nil
	}
	return false, err
}

// ReadFile reads file content
func ReadFile(path string) ([]byte, error) {
	return os.ReadFile(path)
}

// WriteFile writes content to file
func WriteFile(path string, content []byte) error {
	dir := filepath.Dir(path)
	if err := EnsureDir(dir); err != nil {
		return err
	}
	return os.WriteFile(path, content, 0644)
}

// JoinPath joins path components
func JoinPath(elem ...string) string {
	return filepath.Join(elem...)
}

// GetWorkDir gets working directory for agent
func GetWorkDir(baseDir, agentID string) string {
	return JoinPath(baseDir, "agents", agentID)
}

// GetTaskDir gets task directory
func GetTaskDir(baseDir, taskID string) string {
	return JoinPath(baseDir, "tasks", taskID)
}
