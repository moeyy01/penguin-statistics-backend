package cache

import (
	"fmt"
	"reflect"
	"sync"
	"time"

	"github.com/patrickmn/go-cache"
	"github.com/rs/zerolog/log"
)

func NewSet(prefix string) *Set {
	return &Set{
		prefix: prefix + ":",
		c:      cache.New(cache.NoExpiration, time.Minute*10),
	}
}

type Set struct {
	// m is a mutex for MutexGetSet for concurrent prevention
	m sync.Mutex

	prefix string

	c *cache.Cache
}

func (c *Set) key(key string) string {
	return c.prefix + key
}

func (c *Set) Get(key string, dest interface{}) error {
	key = c.key(key)
	result, ok := c.c.Get(key)
	if !ok {
		return ErrNotFound
	}

	// copy value to dest
	var r reflect.Value
	if reflect.ValueOf(result).Kind() == reflect.Ptr {
		r = reflect.ValueOf(result).Elem()
	} else {
		r = reflect.ValueOf(result)
	}
	fmt.Println(result, dest)
	fmt.Println(reflect.TypeOf(result), reflect.TypeOf(dest))
	fmt.Println(r.Type(), reflect.TypeOf(dest))
	reflect.ValueOf(dest).Elem().Set(r)
	return nil
}

func (c *Set) Set(key string, value interface{}, expire time.Duration) error {
	log.Trace().Str("key", key).Msg("setting value to cache")
	key = c.key(key)
	c.c.Set(key, value, expire)
	return nil
}

// MutexGetSet gets value from cache and writes to dest, or if the key does not exists, it executes valueFunc
// to get cache value if the key still not exists when serially dispatched, sets value to cache and
// writes value to dest.
// The first return value means whether the value is got from cache or not. True means calculated; False means got from cache.
func (c *Set) MutexGetSet(key string, dest interface{}, valueFunc func() (interface{}, error), expire time.Duration) (bool, error) {
	err := c.Get(key, dest)
	if err == nil {
		return false, nil
	}
	// onwards, cache key does not exist

	return true, c.slowMutexGetSet(key, dest, valueFunc, expire)
}

func (c *Set) slowMutexGetSet(key string, dest interface{}, valueFunc func() (interface{}, error), expire time.Duration) error {
	c.m.Lock()
	defer c.m.Unlock()
	err := c.Get(key, dest)

	if err == nil {
		return nil
	}

	value, err := valueFunc()
	if err != nil {
		log.Error().Err(err).Str("key", key).Msg("failed to get value from valueFunc() in MutexGetSet")
		return err
	}

	err = c.Set(key, value, expire)
	if err != nil {
		log.Error().Err(err).Str("key", key).Msg("failed to set value to cache in MutexGetSet")
		return err
	}

	// copy value to dest
	var r reflect.Value
	if reflect.ValueOf(value).Kind() == reflect.Ptr {
		r = reflect.ValueOf(value).Elem()
	} else {
		r = reflect.ValueOf(value)
	}
	reflect.ValueOf(dest).Elem().Set(r)

	return nil
}

func (c *Set) Delete(key string) error {
	key = c.key(key)
	if l := log.Trace(); l.Enabled() {
		l.Str("key", key).Msg("deleting value from cache")
	}
	c.c.Delete(key)

	return nil
}

func (c *Set) Clear() error {
	c.c.Flush()
	return nil
}
