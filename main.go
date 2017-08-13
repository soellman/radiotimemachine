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
	//backend := &RedisBackend{}
	//if err := backend.Ping(); err != nil {
	//	log.Fatalf("cannot init backend: %v\n", err)
	//}

	backend := &EtcdBackend{}
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
	}
	r.On()

	if false {
		s := &Station{
			Name:     "wamc",
			Url:      "http://playerservices.streamtheworld.com/api/livestream-redirect/WAMCHD2.mp3",
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
		time.Sleep(100 * time.Millisecond)
		done <- true
	}()

	<-done
}
