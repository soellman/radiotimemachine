package main

import "time"

const TTL = time.Duration(24 * time.Hour)

// A TapeDeck is the backend that records and plays tapes
type TapeDeck struct {
	backend TapeBackend
}

func (deck *TapeDeck) BlankTape(name string, cue time.Time) *BlankTape {
	return deck.backend.BlankTape(name, Incrementer{cue})
}

func (deck *TapeDeck) RecordedTape(name string, cue time.Time) *RecordedTape {
	return deck.backend.RecordedTape(name, Incrementer{cue})
}

// Incrementer increments time
// TODO: add duration here
type Incrementer struct {
	t time.Time
}

// TODO: use format that doesn't include offset?
func (i *Incrementer) Key() string {
	ts := i.t.Format(time.RFC3339)
	i.t = i.t.Add(time.Duration(time.Second * ChunkSeconds))
	return ts
}

type TapeBackend interface {
	BlankTape(name string, i Incrementer) *BlankTape
	RecordedTape(name string, i Incrementer) *RecordedTape
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
