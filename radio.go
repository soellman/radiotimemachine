package main

import (
	"context"
	"fmt"
	"io"
	"log"
	"net/http"
	"time"

	"github.com/pkg/errors"
)

type Location time.Location

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
	Tune(ctx context.Context) (s Stream, err error)
}

// RadioOptions enables some radio features
type RadioOptions struct {
	Broadcast bool
	Listen    bool
}

// A Radio manages all the stations and recordings
type Radio struct {
	ctx      context.Context
	Address  string
	Stations map[string]*Station
	Presets  *Presets
	TapeDeck *TapeDeck
	Options  RadioOptions
}

// Listen tunes into a station and records to a blank tape
func (r *Radio) Listen(s *Station) error {
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

// Turn on the radio and start recording presets
func (r *Radio) On() {
	if r.Options.Listen {
		log.Println("Loading presets")
		stations, err := r.Presets.Load()
		if err != nil {
			log.Printf("error loading presets: %v", err)
		}

		for _, s := range stations {
			err = s.Init()
			if err != nil {
				log.Printf("error loading preset %s: %v", s.Name, err)
				continue
			}
			r.Stations[s.Name] = s
		}

		for _, s := range r.Stations {
			r.Listen(s)
		}
	}

	if r.Options.Broadcast {
		log.Println("Starting broadcast")
		http.HandleFunc("/", r.Broadcast)
		// TODO: graceful
		go http.ListenAndServe(r.Address, nil)
	}
}

// Broadcast listens for requests and streams a station
func (r *Radio) Broadcast(rw http.ResponseWriter, req *http.Request) {
	sp, err := ParsePath(req.URL.Path[1:])
	if err != nil {
		http.NotFound(rw, req)
		fmt.Fprintln(rw, "want path: /<station>/<location e.g. America/New_York>")
		return
	}

	s, err := r.Presets.Lookup(sp.stationName)
	if err != nil {
		http.NotFound(rw, req)
		return
	}

	listenerTime := s.ListenerTime(sp.listenerLocation)
	tape := r.TapeDeck.RecordedTape(s.Name, listenerTime)

	log.Printf("Streaming %s to %s\n", s.Name, req.RemoteAddr)

	// thanks http://engineering.pivotal.io/post/http-trailers/
	trailerKey := http.CanonicalHeaderKey("X-Streaming-Error")

	// NOTE: We set this in the Header because of the HTTP spec
	// http://www.w3.org/Protocols/rfc2616/rfc2616-sec14.html#sec14.40
	// Even though we cannot test it, because the `net/http.Get()` strips
	// "Trailer" out of the Header
	rw.Header().Set("Trailer", trailerKey)

	if err := r.Stream(tape, rw); err != nil {
		rw.(http.Flusher).Flush()
		trailers := http.Header{}
		trailers.Set(trailerKey, err.Error())

		rw.(http.Flusher).Flush()
		conn, buf, _ := rw.(http.Hijacker).Hijack()

		buf.WriteString("0\r\n") // eof
		trailers.Write(buf)

		buf.WriteString("\r\n") // end of trailers
		buf.Flush()
		conn.Close()
	}
}

var (
	errStreamCancelled  = errors.New("stream cancelled")
	errStreamReadError  = errors.New("backend error")
	errStreamWriteError = errors.New("client error")
)

func (r *Radio) Stream(t *RecordedTape, rw http.ResponseWriter) error {
	pushchunk := func() error {
		chunk, err := t.tape.Read()
		if err != nil {
			log.Printf("read error from tape: %+v\n", err)
			return errStreamReadError
		}
		if _, err := rw.Write(chunk); err != nil {
			log.Printf("write error to client: %+v\n", err)
			return errStreamWriteError
		}
		rw.(http.Flusher).Flush()
		return nil
	}

	// push some chunks to the client's buffer
	for i := 0; i < BufferChunks; i++ {
		if err := pushchunk(); err != nil {
			return err
		}
	}

	ticker := time.NewTicker(time.Second * time.Duration(ChunkSeconds))
	defer ticker.Stop()

	for {
		select {
		case <-r.ctx.Done():
			log.Printf("cancelling stream\n")
			return errStreamCancelled
		case <-ticker.C:
			if err := pushchunk(); err != nil {
				return err
			}
		}
	}
}

// TODO: where should errors go?
func (r *Radio) AddStation(s *Station) {
	r.Stations[s.Name] = s
	r.Presets.Add(s)
	r.Listen(s)
}

type Backend interface {
	Init(host string, port int) error
	PresetBackend
	TapeBackend
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
