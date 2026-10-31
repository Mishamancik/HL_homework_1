package store

import (
	"errors"
	"sync"
)

var (
	ErrNotFound      = errors.New("item not found")
	ErrAlreadyExists = errors.New("item already exists")
)

// Item is a simple in-memory record.
type Item struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

// Store is a thread-safe in-memory CRUD map.
type Store struct {
	mu    sync.RWMutex
	items map[string]Item
}

func New() *Store {
	return &Store{items: make(map[string]Item)}
}

func (s *Store) Create(item Item) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.items[item.ID]; ok {
		return ErrAlreadyExists
	}
	s.items[item.ID] = item
	return nil
}

func (s *Store) Get(id string) (Item, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	item, ok := s.items[id]
	if !ok {
		return Item{}, ErrNotFound
	}
	return item, nil
}

func (s *Store) Update(item Item) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.items[item.ID]; !ok {
		return ErrNotFound
	}
	s.items[item.ID] = item
	return nil
}
