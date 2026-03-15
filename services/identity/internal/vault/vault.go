package vault

import (
	"fmt"
	"sync"
)

// Client is the interface for secret storage.
type Client interface {
	Store(path string, data map[string][]byte) error
	Read(path string) (map[string][]byte, error)
	Delete(path string) error
}

// InMemoryClient is a dev-mode in-memory Vault client.
type InMemoryClient struct {
	mu      sync.RWMutex
	secrets map[string]map[string][]byte
}

// NewInMemoryClient creates a new in-memory Vault client.
func NewInMemoryClient() *InMemoryClient {
	return &InMemoryClient{
		secrets: make(map[string]map[string][]byte),
	}
}

func (c *InMemoryClient) Store(path string, data map[string][]byte) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.secrets[path] = data
	return nil
}

func (c *InMemoryClient) Read(path string) (map[string][]byte, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	data, ok := c.secrets[path]
	if !ok {
		return nil, fmt.Errorf("secret not found: %s", path)
	}
	return data, nil
}

func (c *InMemoryClient) Delete(path string) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	delete(c.secrets, path)
	return nil
}
