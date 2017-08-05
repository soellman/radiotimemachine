package main

import (
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"
)

const ChunkSeconds = 20

func main() {

	r := Radio{
		TapeDeck: &TapeDeck{
			device: RedisDevice{},
		},
	}

	loc, _ := time.LoadLocation("America/New_York")
	s := &Station{
		Name:     "wamc",
		Url:      "http://playerservices.streamtheworld.com/api/livestream-redirect/WAMCHD2.mp3",
		Location: loc,
	}

	r.AddStation(s)

	sigs := make(chan os.Signal, 1)
	done := make(chan bool, 1)

	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		sig := <-sigs
		log.Printf("Received signal %s", sig)
		done <- true
	}()

	<-done
}
