package auth

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// TokenStorage Token storage interface
type TokenStorage interface {
	// Save saves token
	Save(accountID string, token *TokenInfo) error
	// Load loads token
	Load(accountID string) (*TokenInfo, error)
	// LoadAll loads all tokens
	LoadAll() (map[string]*TokenInfo, error)
	// Delete deletes token
	Delete(accountID string) error
	// Exists checks if token exists
	Exists(accountID string) bool
}

// FileTokenStorage File-based token storage implementation
type FileTokenStorage struct {
	mu        sync.RWMutex
	filePath  string
	cache     map[string]*TokenInfo
	cacheTime time.Time
	cacheTTL  time.Duration // Cache time-to-live
}

// NewFileTokenStorage creates new file storage
func NewFileTokenStorage(filePath string) (*FileTokenStorage, error) {
	// Ensure directory exists
	dir := filepath.Dir(filePath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create directory: %w", err)
	}

	storage := &FileTokenStorage{
		filePath: filePath,
		cache:    make(map[string]*TokenInfo),
		cacheTTL: 5 * time.Minute, // 5 minutes cache
	}

	// Try to load existing data
	if err := storage.loadFromFile(); err != nil && !os.IsNotExist(err) {
		return nil, fmt.Errorf("failed to load existing data: %w", err)
	}

	return storage, nil
}

// Save saves token to file
func (fs *FileTokenStorage) Save(accountID string, token *TokenInfo) error {
	fs.mu.Lock()
	defer fs.mu.Unlock()

	// Update cache
	fs.cache[accountID] = token
	fs.cacheTime = time.Now()

	// Persist to file
	return fs.persistToFile()
}

// Load loads token
func (fs *FileTokenStorage) Load(accountID string) (*TokenInfo, error) {
	fs.mu.RLock()
	defer fs.mu.RUnlock()

	// Check if cache is valid
	if time.Since(fs.cacheTime) < fs.cacheTTL {
		if token, exists := fs.cache[accountID]; exists {
			return token, nil
		}
	}

	// Cache expired, reload from file
	if err := fs.loadFromFile(); err != nil {
		return nil, err
	}

	token, exists := fs.cache[accountID]
	if !exists {
		return nil, fmt.Errorf("token not found for account: %s", accountID)
	}

	return token, nil
}

// LoadAll loads all tokens
func (fs *FileTokenStorage) LoadAll() (map[string]*TokenInfo, error) {
	fs.mu.RLock()
	defer fs.mu.RUnlock()

	// Check if cache is valid
	if time.Since(fs.cacheTime) < fs.cacheTTL {
		// Return cache copy
		result := make(map[string]*TokenInfo, len(fs.cache))
		for k, v := range fs.cache {
			result[k] = v
		}
		return result, nil
	}

	// Cache expired, reload from file
	if err := fs.loadFromFile(); err != nil {
		return nil, err
	}

	// Return copy
	result := make(map[string]*TokenInfo, len(fs.cache))
	for k, v := range fs.cache {
		result[k] = v
	}
	return result, nil
}

// Delete deletes token
func (fs *FileTokenStorage) Delete(accountID string) error {
	fs.mu.Lock()
	defer fs.mu.Unlock()

	// Delete from cache
	delete(fs.cache, accountID)
	fs.cacheTime = time.Now()

	// Persist to file
	return fs.persistToFile()
}

// Exists checks if token exists
func (fs *FileTokenStorage) Exists(accountID string) bool {
	fs.mu.RLock()
	defer fs.mu.RUnlock()

	_, exists := fs.cache[accountID]
	return exists
}

// loadFromFile loads data from file
func (fs *FileTokenStorage) loadFromFile() error {
	data, err := os.ReadFile(fs.filePath)
	if err != nil {
		return err
	}

	var tokens map[string]*TokenInfo
	if err := json.Unmarshal(data, &tokens); err != nil {
		return fmt.Errorf("failed to unmarshal tokens: %w", err)
	}

	fs.cache = tokens
	fs.cacheTime = time.Now()

	return nil
}

// persistToFile persists to file
func (fs *FileTokenStorage) persistToFile() error {
	data, err := json.MarshalIndent(fs.cache, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal tokens: %w", err)
	}

	// Atomic write (write to temp file first, then rename)
	tmpFile := fs.filePath + ".tmp"
	if err := os.WriteFile(tmpFile, data, 0600); err != nil {
		return fmt.Errorf("failed to write temp file: %w", err)
	}

	if err := os.Rename(tmpFile, fs.filePath); err != nil {
		return fmt.Errorf("failed to rename file: %w", err)
	}

	return nil
}

// MemoryTokenStorage Memory-based token storage (for testing)
type MemoryTokenStorage struct {
	mu     sync.RWMutex
	tokens map[string]*TokenInfo
}

// NewMemoryTokenStorage creates new memory storage
func NewMemoryTokenStorage() *MemoryTokenStorage {
	return &MemoryTokenStorage{
		tokens: make(map[string]*TokenInfo),
	}
}

// Save saves token to memory
func (ms *MemoryTokenStorage) Save(accountID string, token *TokenInfo) error {
	ms.mu.Lock()
	defer ms.mu.Unlock()
	ms.tokens[accountID] = token
	return nil
}

// Load loads token from memory
func (ms *MemoryTokenStorage) Load(accountID string) (*TokenInfo, error) {
	ms.mu.RLock()
	defer ms.mu.RUnlock()

	token, exists := ms.tokens[accountID]
	if !exists {
		return nil, fmt.Errorf("token not found for account: %s", accountID)
	}

	return token, nil
}

// LoadAll loads all tokens
func (ms *MemoryTokenStorage) LoadAll() (map[string]*TokenInfo, error) {
	ms.mu.RLock()
	defer ms.mu.RUnlock()

	result := make(map[string]*TokenInfo, len(ms.tokens))
	for k, v := range ms.tokens {
		result[k] = v
	}
	return result, nil
}

// Delete deletes token
func (ms *MemoryTokenStorage) Delete(accountID string) error {
	ms.mu.Lock()
	defer ms.mu.Unlock()
	delete(ms.tokens, accountID)
	return nil
}

// Exists checks if token exists
func (ms *MemoryTokenStorage) Exists(accountID string) bool {
	ms.mu.RLock()
	defer ms.mu.RUnlock()
	_, exists := ms.tokens[accountID]
	return exists
}
