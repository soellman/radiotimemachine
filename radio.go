package main

import (
	"fmt"
	"log"
	"net/http"
	"strings"
	"sync"
	"time"

	"golang.org/x/net/context"

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

	// TODO: wrap in backoff loop
	for {
		stream, err := s.Tune(ctx)
		if err != nil {
			if strings.HasSuffix(err.Error(), "context canceled") {
				log.Printf("canceled stream")
				return
			}

			log.Println(err)
			// r.LogStatus(s.Name, Status{state: StatusErr, err: err})
			continue
		}

		// should this maybe throw an error?
		// if so, set status to error
		tape := r.TapeDeck.BlankTape(s.Name, s.CurrentTime())

		size := stream.Chunksize()
		log.Printf("Recording %s with chunksize %d", s.Name, size)
		// r.LogStatus(s.Name, Status{state: StatusRunning})

		if err := ChunkPipe(size, stream, tape); err != nil {
			if strings.HasSuffix(err.Error(), "context canceled") {
				log.Printf("canceled stream")
				return
			}

			log.Println(err)
			// r.LogStatus(s.Name, Status{state: StatusErr})
			continue
		}
		log.Println("chunkpipe returned")
	}
}

// Turn on the radio and start recording presets
func (r *Radio) On() {
	r.stop = make(stopChan)
	r.wg = &sync.WaitGroup{}
	r.wg.Add(1)

	if r.Options.Record {
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
			go r.StartRecording(s)
		}

		// r.ManageRecordings(r.stop, r.wg)
	}

	if r.Options.Broadcast {
		http.HandleFunc("/", r.Broadcast)
		log.Println("Starting broadcast")
		go r.Server.ListenAndServe()
	}
}

func (r *Radio) Off() {
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

	log.Printf("Streaming %s to %s\n", s.Name, req.RemoteAddr)

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
		case <-r.stop:
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
	r.Presets.Add(s)
	r.StartRecording(s)
}

type Backend interface {
	Init(host string, port int) error
	PresetBackend
	TapeBackend
}
