package main

import (
	"fmt"
	"time"

	"golang.org/x/net/context"

	"github.com/go-redis/redis"
)

const TTL = time.Duration(24 * time.Hour)

// A TapeDeck is the backend that records and plays tapes
type TapeDeck struct {
	ctx    context.Context
	device RecordingDevice
}

// TODO: FIXME calc time from location
func (deck *TapeDeck) BlankTape(name string, cue time.Time) *BlankTape {
	i := Incrementer{name: name, t: cue}
	return deck.device.BlankTape(i)
}

func (deck *TapeDeck) RecordedTape(name string, cue time.Time) *RecordedTape {
	i := Incrementer{name: name, t: cue}
	return deck.device.RecordedTape(i)
}

// Incrementer increments time
// TODO: add duration here
type Incrementer struct {
	name string
	t    time.Time
}

// TODO: use format that doesn't include offset?
func (i *Incrementer) Key() string {
	ts := i.t.Format(time.RFC3339)
	i.t = i.t.Add(time.Duration(time.Second * ChunkSeconds))
	return fmt.Sprintf("%s-%s", i.name, ts)
}

type RecordingDevice interface {
	BlankTape(i Incrementer) *BlankTape
	RecordedTape(i Incrementer) *RecordedTape
}

// A RecordedTape plays chunks from the datastore via the Reader interface
// The implementation would have been instantiated with a Station
// and frequency and start time and backend
type RecordedTape struct {
	tape TapePlayer
}

// A BlankTape writes data to the datastore via the Writer interface
// the implementation would have been instantiated with a Station
// and start time and backend
type BlankTape struct {
	tape TapeRecorder
}

// Writer interface
func (tape BlankTape) Write(p []byte) (n int, err error) {
	if err = tape.tape.Write(p); err != nil {
		return
	}
	n = len(p)
	return
}

// TapePlayer exposes a simple interface to read a chunk
type TapePlayer interface {
	Read() ([]byte, error)
}

// TapeRecorder exposes a simple interface to write a chunk
type TapeRecorder interface {
	Write(data []byte) error
}

// A RedisDevice implements RecordingDevice and connects to redis
// with a 24h expiration on stored entries
type RedisDevice struct{}

// Implements RecordedTape
func (rd RedisDevice) RecordedTape(i Incrementer) *RecordedTape {
	client := redis.NewClient(&redis.Options{
		Addr:     "localhost:6379",
		Password: "", // no password set
		DB:       0,  // use default DB
	})

	return &RecordedTape{
		tape: &RedisTape{
			i:      i,
			client: client,
		},
	}
}

func (rd RedisDevice) BlankTape(i Incrementer) *BlankTape {
	client := redis.NewClient(&redis.Options{
		Addr:     "localhost:6379",
		Password: "", // no password set
		DB:       0,  // use default DB
	})

	return &BlankTape{
		tape: &RedisTape{
			i:      i,
			client: client,
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
