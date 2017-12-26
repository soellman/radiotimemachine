package main

import (
	"io"
	"net/http"
	"time"

	"golang.org/x/net/context"

	"github.com/pkg/errors"
)

// Station represents a radio station and its location
type Station struct {
	Name     string `json:"name"`
	Url      string `json:"url"`
	Location string `json:"location"`
	loc      *time.Location
	stream   *Stream
}

// Init will initialize the location
func (s *Station) Init() (err error) {
	s.loc, err = time.LoadLocation(s.Location)
	return
}

// Tune into the station, and return a stream, or error
func (s *Station) Tune(ctx context.Context) (*Stream, error) {
	req, err := http.NewRequest("GET", s.Url, nil)
	req = req.WithContext(ctx)

	client := &http.Client{}
	res, err := client.Do(req)
	if err != nil {
		return nil, errors.Wrapf(err, "error connecting to %s", s.Name)
	}

	bitrate, err := DetectBitrate(res.Body)
	if err != nil {
		res.Body.Close()
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
	now := time.Now().In(s.loc)
	trunc := (now.Second() % ChunkSeconds) * int(time.Second)
	return now.Add(time.Duration(-trunc))
}

// ListenerTime returns a time.Time in the station's location
// with the hours set to the current hours in the specified location
// and truncated to the chunkduration boundary
func (s *Station) ListenerTime(loc *time.Location) time.Time {
	now := s.CurrentTime()
	distance := s.ListenerDistance(loc)
	if distance >= 0 {
		distance -= 24 * 60
	}
	return now.Add(time.Duration(distance) * time.Minute)
}

// ListenerDistance returns the number of minutes offset
// the listener is from the station
func (s *Station) ListenerDistance(l *time.Location) int {
	return LocationDistanceInMinutes(time.Now(), l, s.loc)
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
	Tune() (s Stream, err error)
}
