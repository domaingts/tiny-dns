package main

import (
	"container/list"
	"sync"
)

type lru[K comparable, V any] struct {
	capacity int
	cache    map[K]*list.Element
	list     *list.List
	mu       sync.RWMutex
}

type entry[K comparable, V any] struct {
	key   K
	value V
}

func newLRU[K comparable, V any](capacity int) *lru[K, V] {
	return &lru[K, V]{
		capacity: capacity,
		cache:    make(map[K]*list.Element),
		list:     list.New(),
	}
}

func (l *lru[K, V]) get(key K) (V, bool) {
	l.mu.Lock()
	defer l.mu.Unlock()

	if elem, ok := l.cache[key]; ok {
		l.list.MoveToFront(elem)
		return elem.Value.(entry[K, V]).value, true
	}

	var zero V
	return zero, false
}

func (l *lru[K, V]) put(key K, value V) {
	l.mu.Lock()
	defer l.mu.Unlock()

	if l.list.Len() == l.capacity {
		oldest := l.list.Back()
		if oldest != nil {
			l.list.Remove(oldest)
			kv := oldest.Value.(entry[K, V])
			delete(l.cache, kv.key)
		}
	}

	elem := list.New().PushFront(entry[K, V]{key, value})
	l.cache[key] = elem
}
