package main

import (
	"fmt"
	"time"

	"context"

	"github.com/go-redis/redis"
)

// A RedisBackend implements Backend and connects to redis
// with a 24h expiration on stored entries
type RedisBackend struct {
	ssdb   bool
	host   string
	port   int
	client *redis.Client
}

// Implements Backend
func (b *RedisBackend) Init() error {
	b.client = redis.NewClient(&redis.Options{
		Addr:     fmt.Sprintf("%s:%d", b.host, b.port),
		Password: "", // no password set
		DB:       0,  // use default DB
	})
	_, err := b.client.Ping().Result()
	return err
}

// Implements RecordedTape
func (b RedisBackend) RecordedTape(ctx context.Context, name string, i Incrementer) (*RecordedTape, error) {
	return &RecordedTape{
		tape: &RedisTape{
			name:   name,
			i:      i,
			client: b.client,
		},
	}, nil
}

// Implements BlankTape
func (b RedisBackend) BlankTape(ctx context.Context, name string, i Incrementer) (*BlankTape, error) {
	return &BlankTape{
		tape: &RedisTape{
			ssdb:   b.ssdb,
			name:   name,
			i:      i,
			client: b.client,
		},
	}, nil
}

// A RedisTape implements BlankTape and RecordedTape
// and stores entries with an expiration according to TTL
type RedisTape struct {
	ssdb   bool
	name   string
	i      Incrementer
	client *redis.Client
}

func (t *RedisTape) Write(data []byte) error {
	k := fmt.Sprintf("chunk:%s:%s", t.name, t.i.Key())
	if t.ssdb {
		ttl := int(TTL / time.Second)
		if err := SSDBSetx(t.client, k, string(data), ttl).Err(); err != nil {
			return err
		}
	} else {
		if err := t.client.Set(k, data, TTL).Err(); err != nil {
			return err
		}
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
	return b.client.Get(k).Bytes()
}

func (b RedisBackend) ReadAllPresets() (data [][]byte, err error) {
	var keys []string
	if b.ssdb {
		keys, err = SSDBKeys(b.client, "preset:", "preset:zzz", "1000").Result()
	} else {
		keys, err = b.client.Keys("preset:*").Result()
	}
	if err != nil {
		return
	}

	for _, key := range keys {
		d, e := b.client.Get(key).Bytes()
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
	if err := b.client.Set(k, data, 0).Err(); err != nil {
		return err
	}
	return nil
}

// ssdb command support
// this is very weird. why does it need a stringslicecommand?
func SSDBSetx(client *redis.Client, key, value string, ttl int) *redis.StringSliceCmd {
	cmd := redis.NewStringSliceCmd("setx", key, value, ttl)
	client.Process(cmd)
	return cmd
}

func SSDBKeys(client *redis.Client, start, end, limit string) *redis.StringSliceCmd {
	cmd := redis.NewStringSliceCmd("keys", start, end, limit)
	client.Process(cmd)
	return cmd
}
