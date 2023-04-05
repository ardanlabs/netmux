// Package db provides an implementation of a simple database with the ability
// to persist and read the data.
package db

import (
	"errors"
	"fmt"
	"io"
	"sync"

	"gopkg.in/yaml.v3"
)

// Set of error variables.
var (
	ErrNotFound = errors.New("not found")
)

// =============================================================================

// KV represents a key and value stored in the database.
type KV[T any] struct {
	Key   string
	Value T
}

// KVS represents a collection of key/value pairs.
type KVS[T any] []KV[T]

// Values returns the list of values in the collection of key/value pairs.
func (kvs KVS[T]) Values() []T {
	values := make([]T, len(kvs))

	counter := 0
	for _, kv := range kvs {
		values[counter] = kv.Value
		counter++
	}

	return values
}

// Keys returns the list of keys in the collection of key/value pairs.
func (kvs KVS[T]) Keys() []string {
	keys := make([]string, len(kvs))

	counter := 0
	for _, kv := range kvs {
		keys[counter] = kv.Key
		counter++
	}

	return keys
}

// =============================================================================

// DB represents a simple database.
type DB[T any] struct {
	rw   io.ReadWriter
	mu   sync.RWMutex
	data map[string]T
}

// New constructs a new database accepting a filename for persistence.
func New[T any](rw io.ReadWriter) *DB[T] {
	return &DB[T]{
		rw:   rw,
		data: make(map[string]T),
	}
}

// Save writes the contents of the database to disk.
func (db *DB[T]) Save() error {
	data, err := yaml.Marshal(db.data)
	if err != nil {
		return fmt.Errorf("yaml.Marshal: %w", err)
	}

	db.mu.Lock()
	defer db.mu.Unlock()

	if _, err := io.WriteString(db.rw, string(data)); err != nil {
		return fmt.Errorf("io.WriteString: %w", err)
	}

	return nil
}

// Load reads the contents of the databse from disk.
func (db *DB[T]) Load() error {
	db.mu.Lock()
	defer db.mu.Unlock()

	data, err := io.ReadAll(db.rw)
	if err != nil {
		return fmt.Errorf("os.ReadFile: %w", err)
	}

	m := make(map[string]T)
	err = yaml.Unmarshal(data, &m)
	if err != nil {
		return fmt.Errorf("yaml.Unmarshal: %w", err)
	}

	db.data = m

	return nil
}

// Set adds or updates a key in the database with the specified value.
func (db *DB[T]) Set(key string, value T) {
	db.mu.Lock()
	defer db.mu.Unlock()

	db.data[key] = value
}

// Get returns the value for the specified key from the database.
func (db *DB[T]) Get(key string) (T, error) {
	db.mu.RLock()
	defer db.mu.RUnlock()

	value, exists := db.data[key]
	if !exists {
		return *(new(T)), ErrNotFound
	}

	return value, nil
}

// Exists returns true or false based on if the key exists in the database.
func (db *DB[T]) Exists(key string) bool {
	db.mu.RLock()
	defer db.mu.RUnlock()

	if _, err := db.Get(key); err != nil {
		return false
	}

	return true
}

// Delete removes a key from the database.
func (db *DB[T]) Delete(key string) {
	db.mu.Lock()
	defer db.mu.Unlock()

	delete(db.data, key)
}

// ForEach executes the specified function for each key in the database. This
// function holds a read lock while it iterates.
func (db *DB[T]) ForEach(f func(k string, v T) error) error {
	db.mu.RLock()
	defer db.mu.RUnlock()

	for k, v := range db.data {
		if err := f(k, v); err != nil {
			return err
		}
	}

	return nil
}

// KeyValues returns the list of keys and values stored in the database.
func (db *DB[T]) KeyValues() KVS[T] {
	db.mu.RLock()
	defer db.mu.RUnlock()

	values := make(KVS[T], len(db.data))

	counter := 0
	for k, v := range db.data {
		values[counter] = KV[T]{k, v}
		counter++
	}

	return values
}
