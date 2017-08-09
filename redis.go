package main

import "github.com/go-redis/redis"

// A RedisDevice implements RecordingDevice and connects to redis
// with a 24h expiration on stored entries
type RedisDevice struct{}

func NewRedisClient() *redis.Client {
	return redis.NewClient(&redis.Options{
		Addr:     "localhost:6379",
		Password: "", // no password set
		DB:       0,  // use default DB
	})
}

// Implements RecordedTape
func (rd RedisDevice) RecordedTape(i Incrementer) *RecordedTape {
	return &RecordedTape{
		tape: &RedisTape{
			i:      i,
			client: NewRedisClient(),
		},
	}
}

// Implements BlankTape
func (rd RedisDevice) BlankTape(i Incrementer) *BlankTape {
	return &BlankTape{
		tape: &RedisTape{
			i:      i,
			client: NewRedisClient(),
		},
	}
}

// A RedisTape implements BlankTape and RecordedTape
// and stores entries with an expiration according to TTL
type RedisTape struct {
	i      Incrementer
	client *redis.Client
}

func (rt *RedisTape) Write(data []byte) error {
	if err := rt.client.Set(rt.i.Key(), data, TTL); err != nil {
		return err.Err()
	}
	return nil
}

func (rt *RedisTape) Read() ([]byte, error) {
	return rt.client.Get(rt.i.Key()).Bytes()
}

// Implements PresetBackend
func (rd RedisDevice) Read(key string) (data []byte, err error) {
	return NewRedisClient().Get(key).Bytes()
}

// TODO: key should be injected
func (rd RedisDevice) ReadAll() (data [][]byte, err error) {
	client := NewRedisClient()
	keys, e := client.Keys("preset-*").Result()
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

func (rd RedisDevice) Write(key string, data []byte) error {
	if err := NewRedisClient().Set(key, data, 0); err != nil {
		return err.Err()
	}
	return nil
}
