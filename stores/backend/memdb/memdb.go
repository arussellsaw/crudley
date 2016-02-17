package memdb

import (
	"encoding/json"
	"fmt"
	"sync"
)

type Memdb struct {
	sync.Mutex
	store map[string]*Collection
}

func New() *Memdb {
	return &Memdb{
		store: make(map[string]*Collection),
	}
}

func (m *Memdb) Collection(id string) *Collection {
	m.Lock()
	defer m.Unlock()
	_, ok := m.store[id]
	if !ok {
		m.store[id] = &Collection{store: make(map[string][]byte)}
	}
	return m.store[id]
}

type Collection struct {
	sync.Mutex
	store map[string][]byte
}

func (c *Collection) Doc(id string, out interface{}) (bool, error) {
	buf, found, err := c.DocRaw(id)
	if err != nil {
		return false, err
	}
	if !found {
		return found, nil
	}
	return found, json.Unmarshal(buf, out)
}

func (c *Collection) DocRaw(id string) ([]byte, bool, error) {
	c.Lock()
	defer c.Unlock()
	doc, ok := c.store[id]
	return doc, ok, nil
}

func (c *Collection) AllRaw() map[string][]byte {
	if c.store != nil {
		return c.store
	}
	return make(map[string][]byte)
}

func (c *Collection) Set(id string, doc interface{}) error {
	c.Lock()
	defer c.Unlock()
	buf, err := json.Marshal(doc)
	if err != nil {
		return err
	}
	c.store[id] = buf
	return nil
}

func (c *Collection) Remove(id string) error {
	c.Lock()
	defer c.Unlock()
	if _, ok := c.store[id]; ok {
		delete(c.store, id)
		return nil
	}
	return fmt.Errorf("key not found")
}
