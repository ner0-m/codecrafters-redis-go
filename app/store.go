package main

import (
	"sync"
	"time"
)

type Value struct {
	Value      string
	InsertTime time.Time
	Expiry     *time.Duration
}

type Store struct {
	Mutex sync.RWMutex
	Store map[string]Value
}

func (s Store) Write(key string, value string, expiry *time.Duration) {
	s.Mutex.Lock()
	s.Store[key] = Value{
		Value:      value,
		InsertTime: time.Now(),
		Expiry:     expiry,
	}
	s.Mutex.Unlock()
}

func (s Store) Contains(key string) bool {
	s.Mutex.Lock()
	v, ok := s.Store[key]
	s.Mutex.Unlock()

	if v.Expiry == nil {
		return ok
	} else {
		expiryTime := v.InsertTime.Add(*v.Expiry)
		return ok && time.Now().Before(expiryTime)
	}
}

func (s Store) Read(key string) (string, bool) {
	contains := s.Contains(key)

	if !contains {
		return "", false
	}

	v, ok := s.Store[key]

	return v.Value, ok
}
