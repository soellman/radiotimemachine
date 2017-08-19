package main

import (
	"fmt"

	"github.com/go-redis/redis"
)

// A RedisBackend implements Backend and connects to redis
// with a 24h expiration on stored entries
type RedisBackend struct {
	host string
	port int
}

func (b RedisBackend) NewClient() *redis.Client {
	return redis.NewClient(&redis.Options{
		Addr:     fmt.Sprintf("%s:%d", b.host, b.port),
		Password: "", // no password set
		DB:       0,  // use default DB
	})
}

// Implements Backend
func (b *RedisBackend) Init(host string, port int) error {
	b.host = host
	b.port = port
	_, err := b.NewClient().Ping().Result()
	return err
}

// Implements RecordedTape
func (b RedisBackend) RecordedTape(name string, i Incrementer) *RecordedTape {
	return &RecordedTape{
		tape: &RedisTape{
			name:   name,
			i:      i,
			client: b.NewClient(),
		},
	}
}

// Implements BlankTape
func (b RedisBackend) BlankTape(name string, i Incrementer) *BlankTape {
	return &BlankTape{
		tape: &RedisTape{
			name:   name,
			i:      i,
			client: b.NewClient(),
		},
	}
}

// A RedisTape implements BlankTape and RecordedTape
// and stores entries with an expiration according to TTL
type RedisTape struct {
	name   string
	i      Incrementer
	client *redis.Client
}

func (t *RedisTape) Write(data []byte) error {
	k := fmt.Sprintf("chunk:%s:%s", t.name, t.i.Key())
	if err := t.client.Set(k, data, TTL); err != nil {
		return err.Err()
	}
	return nil
}

func (t *RedisTape) Read() ([]byte, error) {
	k := fmt.Sprintf("chunk:%s:%s", t.name, t.i.Key())
	return t.client.Get(k).Bytes()
}

// Implements PresetBackend
func (b RedisBackend) ReadPreset(name string) (data []byte, err error) {
	k := fmt.Sprintf("preset:%s", name)
	return b.NewClient().Get(k).Bytes()
}

func (b RedisBackend) ReadAllPresets() (data [][]byte, err error) {
	client := b.NewClient()
	keys, e := client.Keys("preset:*").Result()
	if e != nil {
		err = e
		return
	}

	for _, key := range keys {
		d, e := client.Get(key).Bytes()
		if e != nil {
			err = e
			return
		}
		data = append(data, d)
	}

	return
}

func (b RedisBackend) WritePreset(name string, data []byte) error {
	k := fmt.Sprintf("preset:%s", name)
	if err := b.NewClient().Set(k, data, 0); err != nil {
		return err.Err()
	}
	return nil
}
