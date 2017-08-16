package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"
)

const ChunkSeconds = 20
const BufferChunks = 2

func main() {
	var backend Backend
	//backend = &EtcdBackend{}
	backend = &RedisBackend{}

	if err := backend.Init(); err != nil {
		log.Fatalf("cannot init backend: %v\n", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	r := Radio{
		ctx:      ctx,
		Stations: make(map[string]*Station),
		TapeDeck: &TapeDeck{
			backend: backend,
		},
		Presets: &Presets{
			backend: backend,
		},
		Options: RadioOptions{
			Broadcast: true,
			Listen:    false,
		},
	}
	r.On()

	if false {
		s := &Station{
			Name:     "wamc",
			Url:      "http://playerservices.streamtheworld.com/api/livestream-redirect/WAMCFM.mp3",
			Location: "America/New_York",
		}

		// ignore error
		_ = s.Init()
		r.AddStation(s)
	}

	sigs := make(chan os.Signal, 1)
	done := make(chan bool, 1)

	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		sig := <-sigs
		log.Printf("Received signal %s", sig)
		cancel()
		time.Sleep(100 * time.Millisecond) // leave time for cancellation
		done <- true
	}()

	<-done
}
