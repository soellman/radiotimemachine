package main

import (
	"fmt"

	redigo "github.com/garyburd/redigo/redis"
	"github.com/go-redis/redis"
)

// A SSDBBackend implements Backend and connects to ssdb
// ssdb is basically redis, but doesn't support KEYS <wildcard>
type SSDBBackend struct {
	host  string
	port  int
	redis *RedisBackend
}

func (b SSDBBackend) NewClient() *redis.Client {
	return redis.NewClient(&redis.Options{
		Addr:     fmt.Sprintf("%s:%d", b.host, b.port),
		Password: "", // no password set
		DB:       0,  // use default DB
	})
}

// Implements Backend
func (b *SSDBBackend) Init(host string, port int) error {
	b.host = host
	b.port = port
	b.redis = &RedisBackend{}
	return b.redis.Init(host, port)
}

// Implements RecordedTape
func (b SSDBBackend) RecordedTape(name string, i Incrementer) *RecordedTape {
	return b.redis.RecordedTape(name, i)
}

// Implements BlankTape
func (b SSDBBackend) BlankTape(name string, i Incrementer) *BlankTape {
	return b.redis.BlankTape(name, i)
}

// Implements PresetBackend
func (b SSDBBackend) ReadPreset(name string) (data []byte, err error) {
	return b.redis.ReadPreset(name)
}

func (b SSDBBackend) ReadAllPresets() (data [][]byte, err error) {
	addr := fmt.Sprintf("%s:%d", b.host, b.port)
	c, e := redigo.Dial("tcp", addr)
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
	return b.redis.WritePreset(name, data)
}
