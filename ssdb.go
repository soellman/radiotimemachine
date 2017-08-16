package main

import (
	"fmt"

	redigo "github.com/garyburd/redigo/redis"
	"github.com/go-redis/redis"
)

// A SSDBBackend implements Backend and connects to redis
// with a 24h expiration on stored entries
type SSDBBackend struct{}

func NewSSDBClient() *redis.Client {
	return redis.NewClient(&redis.Options{
		Addr:     dbhost + ":" + dbport,
		Password: "", // no password set
		DB:       0,  // use default DB
	})
}

// Implements Backend
func (b SSDBBackend) Init() error {
	_, err := NewRedisClient().Ping().Result()
	return err
}

// Implements RecordedTape
func (b SSDBBackend) RecordedTape(name string, i Incrementer) *RecordedTape {
	return &RecordedTape{
		tape: &RedisTape{
			name:   name,
			i:      i,
			client: NewRedisClient(),
		},
	}
}

// Implements BlankTape
func (b SSDBBackend) BlankTape(name string, i Incrementer) *BlankTape {
	return &BlankTape{
		tape: &RedisTape{
			name:   name,
			i:      i,
			client: NewRedisClient(),
		},
	}
}

// Implements PresetBackend
func (b SSDBBackend) ReadPreset(name string) (data []byte, err error) {
	k := fmt.Sprintf("preset:%s", name)
	return NewRedisClient().Get(k).Bytes()
}

func (b SSDBBackend) ReadAllPresets() (data [][]byte, err error) {
	c, e := redigo.Dial("tcp", dbhost+":"+dbport)
	if e != nil {
		err = e
		return
	}
	defer c.Close()

	keys, e := redigo.Strings(c.Do("KEYS", "preset:", "preset:zzz", "1000"))
	if e != nil {
		err = e
		return
	}

	for _, key := range keys {
		d, e := redigo.Bytes(c.Do("GET", key))
		if e != nil {
			err = e
			return
		}
		data = append(data, d)
	}

	return
}

func (b SSDBBackend) WritePreset(name string, data []byte) error {
	k := fmt.Sprintf("preset:%s", name)
	if err := NewRedisClient().Set(k, data, 0); err != nil {
		return err.Err()
	}
	return nil
}
