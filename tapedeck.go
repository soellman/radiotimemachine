package main

import (
	"fmt"
	"time"

	"golang.org/x/net/context"
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
	return fmt.Sprintf("chunk-%s-%s", i.name, ts)
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
