package memdb

import (
	"gopkg.in/yaml.v3"
	"os"
	"sync"
)

type Memdb[T any] struct {
	Fname string
	mx    sync.RWMutex
	Db    map[string]T
}

func (m *Memdb[T]) Save() error {
	bs, err := yaml.Marshal(m.Db)
	if err != nil {
		return err
	}
	return os.WriteFile(m.Fname, bs, 0600)
}

func (m *Memdb[T]) Load() error {
	bs, err := os.ReadFile(m.Fname)
	if err != nil {
		return err
	}
	db := make(map[string]T)

	err = yaml.Unmarshal(bs, &db)
	if err != nil {
		return err
	}
	m.Db = db
	return nil
}

func (m *Memdb[T]) Keys() []string {
	m.mx.Lock()
	defer m.mx.Unlock()
	var ret = make([]string, len(m.Db))
	counter := 0
	for k := range m.Db {
		ret[counter] = k
		counter++
	}
	return ret
}

func (m *Memdb[T]) Get(i string) T {
	m.mx.RLock()
	defer m.mx.RUnlock()
	return m.Db[i]
}

func (m *Memdb[T]) Has(i string) bool {
	m.mx.RLock()
	defer m.mx.RUnlock()
	_, ok := m.Db[i]
	return ok
}

func (m *Memdb[T]) Set(i string, t T) {
	m.mx.Lock()
	defer m.mx.Unlock()
	m.Db[i] = t
}

func (m *Memdb[T]) Del(i string) {
	m.mx.Lock()
	defer m.mx.Unlock()
	delete(m.Db, i)
}

func (m *Memdb[T]) ForEach(f func(k string, v T) bool) {
	m.mx.RLock()
	defer m.mx.RUnlock()
	for k, v := range m.Db {
		ret := f(k, v)
		if !ret {
			return
		}
	}
}

func (m *Memdb[T]) Items() []T {
	m.mx.RLock()
	defer m.mx.RUnlock()
	ret := make([]T, len(m.Db))

	counter := 0

	for _, v := range m.Db {
		ret[counter] = v
		counter++
	}

	return ret
}

func New[T any]() *Memdb[T] {
	var ret = Memdb[T]{
		Db: make(map[string]T),
	}
	return &ret
}
