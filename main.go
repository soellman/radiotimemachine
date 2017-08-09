package main

import (
	"log"
	"os"
	"os/signal"
	"syscall"
)

const ChunkSeconds = 20

func main() {

	backend := RedisDevice{}
	presets, err := PresetsWithBackend(backend)
	if err != nil {
		log.Fatal(err)
	}

	r := Radio{
		TapeDeck: &TapeDeck{
			device: backend,
		},
		Presets: presets,
	}

	r.On()

	//s := &Station{
	//	Name:     "wamc",
	//	Url:      "http://playerservices.streamtheworld.com/api/livestream-redirect/WAMCHD2.mp3",
	//	Location: "America/New_York",
	//}

	// ignore error
	//_ = s.Init()
	//r.AddStation(s)

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