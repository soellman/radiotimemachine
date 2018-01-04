package main

import (
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"

	"golang.org/x/net/context"

	"github.com/cenkalti/backoff"
	"github.com/go-kit/kit/log/level"
	"github.com/gorilla/handlers"
	"github.com/pkg/errors"
)

type stopChan chan struct{}

// RadioOptions enables some radio features
type RadioOptions struct {
	Broadcast bool
	Record    bool
}

// A Radio manages all the stations and recordings
type Radio struct {
	Server   *http.Server
	Presets  *Presets
	TapeDeck *TapeDeck
	Options  RadioOptions

	stop stopChan
	wg   *sync.WaitGroup
}

// Record tunes into a station and records to a blank tape
func (r *Radio) StartRecording(s *Station) {
	r.wg.Add(1)
	defer r.wg.Done()

	// Use a context to provide cancellation of the http client
	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		<-r.stop
		cancel()
	}()

	rec := func() error {
		stream, err := s.Tune(ctx)
		if err != nil {
			if strings.HasSuffix(err.Error(), "context canceled") {
				level.Debug(logger).Log(
					"msg", "canceled stream",
					"station", s.Name)
				return nil
			}

			level.Warn(logger).Log(
				"msg", "error tuning into station",
				"station", s.Name,
				"err", err)
			// r.LogStatus(s.Name, Status{state: StatusErr, err: err})
			return err
		}

		// should this maybe throw an error?
		// if so, set status to error
		tape := r.TapeDeck.BlankTape(s.Name, s.CurrentTime())

		size := stream.Chunksize()
		level.Info(logger).Log(
			"msg", fmt.Sprintf("Recording station with chunksize %d", size),
			"station", s.Name)
		// r.LogStatus(s.Name, Status{state: StatusRunning})

		if err := ChunkPipe(size, stream, tape); err != nil {
			if strings.HasSuffix(err.Error(), "context canceled") {
				level.Debug(logger).Log(
					"msg", "canceled stream",
					"station", s.Name)
				return nil
			}

			level.Warn(logger).Log(
				"msg", "error in chunkpipe",
				"err", err)
			// r.LogStatus(s.Name, Status{state: StatusErr})
			return err
		}

		level.Debug(logger).Log(
			"msg", "chunkpipe returned",
			"station", s.Name)
		return nil
	}

	b := backoff.NewExponentialBackOff()
	b.InitialInterval = 2 * time.Second
	b.RandomizationFactor = 0

	err := backoff.Retry(rec, b)
	if err != nil {
		level.Warn(logger).Log(
			"msg", "error after retrying",
			"err", err)
	}
}

// Turn on the radio and start recording presets
func (r *Radio) On() {
	level.Info(logger).Log(
		"msg", "Powering on the time machine",
		"record", r.Options.Record,
		"broadcast", r.Options.Broadcast)

	r.stop = make(stopChan)
	r.wg = &sync.WaitGroup{}
	r.wg.Add(1)

	if r.Options.Record {
		level.Info(logger).Log("msg", "Starting recording presets")

		stations, err := r.Presets.Load()
		if err != nil {
			level.Warn(logger).Log(
				"msg", "error loading presets",
				"err", err)
		}

		for _, s := range stations {
			err = s.Init()
			if err != nil {
				level.Warn(logger).Log(
					"msg", "error loading preset",
					"station", s.Name,
					"err", err)
				continue
			}
			go r.StartRecording(&s)
		}

		// r.ManageRecordings(r.stop, r.wg)
	}

	if r.Options.Broadcast {
		r.Presets.RegisterServiceHandlers(http.DefaultServeMux)
		http.HandleFunc("/listen", r.Broadcast)

		// enable cors
		cors := handlers.CORS(
			handlers.AllowedHeaders([]string{"Content-Type"}),
			handlers.AllowedMethods([]string{"GET", "POST"}),
			handlers.AllowedOrigins([]string{"*"}))
		r.Server.Handler = cors(http.DefaultServeMux)

		level.Info(logger).Log("msg", "Starting broadcast and preset service")

		go r.Server.ListenAndServe()
	}
}

func (r *Radio) Off() {
	level.Info(logger).Log("msg", "Powering down the time machine")

	timeout, _ := context.WithTimeout(context.Background(), 5*time.Second)
	r.Server.Shutdown(timeout)
	close(r.stop)
	r.wg.Done()
	r.wg.Wait()
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

	level.Info(logger).Log(
		"msg", "Broadcasting station",
		"station", s.Name,
		"client", req.RemoteAddr)

	// Set up trailers
	// thanks http://engineering.pivotal.io/post/http-trailers/
	trailerKey := http.CanonicalHeaderKey("X-Streaming-Error")
	rw.Header().Set("Trailer", trailerKey)

	if err := r.Stream(tape, rw); err != nil {
		writeTrailers(err, rw, trailerKey)
	}
}

func writeTrailers(err error, rw http.ResponseWriter, trailerKey string) {
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

var (
	errStreamCanceled   = errors.New("stream canceled")
	errStreamReadError  = errors.New("backend error")
	errStreamWriteError = errors.New("client error")
)

func (r *Radio) Stream(t *RecordedTape, rw http.ResponseWriter) error {
	pushchunk := func() error {
		chunk, err := t.tape.Read()
		if err != nil {
			level.Warn(logger).Log(
				"msg", "error reading from tape",
				"err", err)
			return errStreamReadError
		}
		if _, err := rw.Write(chunk); err != nil {
			level.Warn(logger).Log(
				"msg", "error writing to client",
				"err", err)
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
		case <-r.stop:
			level.Debug(logger).Log("msg", "canceling stream")
			return errStreamCanceled
		case <-ticker.C:
			if err := pushchunk(); err != nil {
				return err
			}
		}
	}
}

type Backend interface {
	Init(host string, port int) error
	PresetBackend
	TapeBackend
}
