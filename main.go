package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"sync"
	"syscall"
)

const (
	ChunkSeconds = 20
	BufferChunks = 2
)

var (
	driver    string
	dbhost    string
	dbport    int
	record    bool
	broadcast bool
	addr      string
)

func init() {
	flag.StringVar(&driver, "driver", "ssdb", "Database driver: etcd|redis|ssdb")
	flag.StringVar(&dbhost, "dbhost", "localhost", "Database host")
	flag.IntVar(&dbport, "dbport", 8888, "Database port")
	flag.BoolVar(&record, "record", true, "Record presets")
	flag.BoolVar(&broadcast, "broadcast", true, "Broadcast to users")
	flag.StringVar(&addr, "addr", ":8080", "Broadcast address")
}

func main() {
	flag.Parse()

	var backend Backend
	switch driver {
	case "etcd":
		backend = &EtcdBackend{}
	case "redis":
		backend = &RedisBackend{}
	case "ssdb":
		backend = &RedisBackend{ssdb: true}
	default:
		fmt.Printf("No %s driver found\n", driver)
		os.Exit(1)
	}

	if err := backend.Init(dbhost, dbport); err != nil {
		log.Fatalf("Cannot init backend: %v\n", err)
	}

	stop := make(stopChan)
	r := Radio{
		stop:     stop,
		Address:  addr,
		Stations: make(map[string]*Station),
		TapeDeck: &TapeDeck{
			backend: backend,
		},
		Presets: &Presets{
			backend: backend,
		},
		Options: RadioOptions{
			Broadcast: broadcast,
			Record:    record,
		},
		//RecordingEngineer: RecordingEngineer{
		//	ch: make(chan StatusMessage, 1),
		//	s:  make(map[string]Status),
		//},
		wg: &sync.WaitGroup{},
	}
	log.Println("Powering on the time machine")
	log.Printf("  with options: %+v\n", r.Options)
	log.Printf("  and driver: %s\n", driver)
	log.Printf("  and db addr: %s:%d\n", dbhost, dbport)
	r.On()

	sigs := make(chan os.Signal, 1)
	done := make(chan bool, 1)

	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		sig := <-sigs
		log.Printf("Received signal %s", sig)
		log.Println("Powering down the time machine")
		close(r.stop)
		done <- true
	}()

	<-done

	r.wg.Wait()
	log.Println("Done")
}
