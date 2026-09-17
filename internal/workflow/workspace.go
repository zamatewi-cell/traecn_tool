package workflow

import (
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// Workspace represents an agent workspace
type Workspace struct {
	ID        string
	RootDir   string
	CreatedAt time.Time
}

// NewWorkspace creates new workspace
func NewWorkspace(id, rootDir string) (*Workspace, error) {
	// Create directory
	path := filepath.Join(rootDir, id)
	if err := os.MkdirAll(path, 0755); err != nil {
		return nil, fmt.Errorf("failed to create workspace: %w", err)
	}

	return &Workspace{
		ID:        id,
		RootDir:   path,
		CreatedAt: time.Now(),
	}, nil
}

// GetPath gets subdirectory path
func (w *Workspace) GetPath(subdir string) string {
	return filepath.Join(w.RootDir, subdir)
}

// EnsureSubdir ensures subdirectory exists
func (w *Workspace) EnsureSubdir(subdir string) error {
	path := w.GetPath(subdir)
	return os.MkdirAll(path, 0755)
}

// WriteFile writes file to workspace
func (w *Workspace) WriteFile(subdir, filename, content string) error {
	dir := w.GetPath(subdir)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}

	path := filepath.Join(dir, filename)
	return os.WriteFile(path, []byte(content), 0644)
}

// ReadFile reads file from workspace
func (w *Workspace) ReadFile(subdir, filename string) ([]byte, error) {
	path := filepath.Join(w.GetPath(subdir), filename)
	return os.ReadFile(path)
}

// FileExists checks if file exists in workspace
func (w *Workspace) FileExists(subdir, filename string) (bool, error) {
	path := filepath.Join(w.GetPath(subdir), filename)
	_, err := os.Stat(path)
	if err == nil {
		return true, nil
	}
	if os.IsNotExist(err) {
		return false, nil
	}
	return false, err
}

// ListFiles lists files in subdirectory
func (w *Workspace) ListFiles(subdir string) ([]string, error) {
	path := w.GetPath(subdir)
	entries, err := os.ReadDir(path)
	if err != nil {
		return nil, err
	}

	files := make([]string, 0)
	for _, entry := range entries {
		if !entry.IsDir() {
			files = append(files, entry.Name())
		}
	}

	return files, nil
}

// Cleanup removes workspace
func (w *Workspace) Cleanup() error {
	return os.RemoveAll(w.RootDir)
}
