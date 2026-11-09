// Package container mirrors @medusajs/framework/src/container: the
// dependency container routes, workflows and modules resolve from.
package container

import "sync"

// MedusaContainer is a registry of named values.
type MedusaContainer struct {
	mu     sync.RWMutex
	values map[string]any
	parent *MedusaContainer
}

// New creates an empty container.
func New() *MedusaContainer {
	return &MedusaContainer{values: map[string]any{}}
}

// Register registers a value under key.
func (c *MedusaContainer) Register(key string, v any) {
	c.mu.Lock()
	c.values[key] = v
	c.mu.Unlock()
}

// Resolve returns the value registered under key (nil when missing).
func (c *MedusaContainer) Resolve(key string) any {
	c.mu.RLock()
	v, ok := c.values[key]
	c.mu.RUnlock()
	if !ok && c.parent != nil {
		return c.parent.Resolve(key)
	}
	return v
}

// Has reports whether key is registered.
func (c *MedusaContainer) Has(key string) bool {
	c.mu.RLock()
	_, ok := c.values[key]
	c.mu.RUnlock()
	if !ok && c.parent != nil {
		return c.parent.Has(key)
	}
	return ok
}

// CreateScope mirrors createScope: a child container.
func (c *MedusaContainer) CreateScope() *MedusaContainer {
	return &MedusaContainer{values: map[string]any{}, parent: c}
}
