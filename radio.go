package main

import (
	"context"
	"io"
	"log"
	"net/http"
	"time"

	"github.com/pkg/errors"
)

// Station represents a radio station and its location
type Station struct {
	Name     string
	Url      string
	Location *time.Location
	stream   *Stream
}

// Tune into the station, and return a stream, or error
func (s *Station) Tune(ctx context.Context) (*Stream, error) {
	res, err := http.Get(s.Url)
	if err != nil {
		return nil, errors.Wrapf(err, "error connecting to %s", s.Name)
	}
	// TODO: where do we close the body?
	//defer res.Body.Close()

	bitrate, err := DetectBitrate(res.Body)
	if err != nil {
		return nil, errors.Wrapf(err, "error detecting bitrate for %s", s.Name)
	}

	s.stream = &Stream{
		Bitrate: bitrate,
		Reader:  res.Body,
	}

	return s.stream, nil
}

// CurrentTime returns a time.Time in station's location,
// truncated to the chunkduration boundary
func (s *Station) CurrentTime() time.Time {
	now := time.Now().In(s.Location)
	trunc := (now.Second() % ChunkSeconds) * int(time.Second)
	return now.Add(time.Duration(-trunc))
}

// ListenerTime returns a time.Time in the station's location
// with the hours set to the current hours in the specified location
// and truncated to the chunkduration boundary
func (s *Station) ListenerTime(l *time.Location) time.Time {

	return time.Now()
}

// ListenerDistance returns the number of minutes offset
// the listener is from the station
func (s *Station) ListenerDistance(l *time.Location) int {
	return LocationDistanceInMinutes(time.Now(), l, s.Location)
}

// A stream represents a tuned-in radio station
type Stream struct {
	Bitrate int
	io.Reader
}

// Chunksize in bytes
func (s *Stream) Chunksize() int {
	return s.Bitrate * ChunkSeconds / 8
}

// The Dial is used to tune in to a station
// does this need to be an interface?
type Dialer interface {
	Tune(ctx context.Context) (s Stream, err error)
}

// A Radio manages all the stations and recordings
type Radio struct {
	ctx      context.Context
	Stations []*Station
	TapeDeck *TapeDeck
}

// Record tunes into a station and records to a blank tape
func (r *Radio) Record(s *Station) error {
	stream, err := s.Tune(r.ctx)
	if err != nil {
		log.Printf("error recording station %s: %v", s.Name, err)
		return err
	}

	size := stream.Chunksize()
	tape := r.TapeDeck.BlankTape(s.Name, s.CurrentTime())

	log.Printf("Recording %s with chunksize %d", s.Name, size)
	go ChunkPump(size, stream, tape)

	return nil
}

// TODO: where should errors go?
func (r *Radio) AddStation(s *Station) {
	r.Stations = append(r.Stations, s)
	r.Record(s)
}

// ChunkPump pumps reads from the reader to the writer
// with a specified chunksize. This will return when
// the reader finishes or errors.
func ChunkPump(size int, r io.Reader, w io.Writer) error {
	b := make([]byte, size)

	for {
		if _, err := io.ReadFull(r, b); err != nil {
			if err == io.EOF {
				return nil
			}
			return err
		}
		if _, err := w.Write(b); err != nil {
			return err
		}
	}
}
