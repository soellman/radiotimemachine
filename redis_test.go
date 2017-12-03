package main

import (
	"bytes"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/alicebob/miniredis"
)

var (
	b    = &RedisBackend{}
	name = "wkrp"
	data = []byte("this is not the station you are looking for")
)

func TestRedis(t *testing.T) {
	if err := b.Init("localhost", 9); err == nil {
		t.Fatalf("there is a ghost redis on tcp port 9")
	}

	// set up our backend
	s, err := miniredis.Run()
	if err != nil {
		panic(err)
	}
	defer s.Close()

	parts := strings.Split(s.Addr(), ":")
	if len(parts) != 2 {
		t.Fatalf("unable to get miniredis address")
	}

	var port int
	if port, err = strconv.Atoi(parts[1]); err != nil {
		t.Fatalf("unable to get miniredis port")
	}

	if b.Init(parts[0], port); err != nil {
		t.Fatalf("miniredis can't ping: %v", err)
	}

	// Now run subtests with our prepared backend
	t.Run("Presets", testRedisPresets)
	t.Run("Tapes", testRedisTapes)
}

func testRedisPresets(t *testing.T) {
	if err := b.WritePreset(name, data); err != nil {
		t.Fatalf("miniredis failed")
	}

	d, err := b.ReadPreset(name)
	if err != nil {
		t.Fatalf("miniredis failed")
	}

	if !bytes.Equal(d, data) {
		t.Errorf("retrieved data doesn't match. expected %b, got %b\n", data, d)
	}

	ds, err := b.ReadAllPresets()
	if err != nil {
		t.Fatalf("miniredis failed")
	}

	if len(ds) != 1 {
		t.Errorf("number of presets incorrect. expected 1, got %d\n", len(ds))
		return
	}

	if !bytes.Equal(ds[0], data) {
		t.Errorf("retrieved data doesn't match. expected %b, got %b\n", data, ds[0])
	}
}

func testRedisTapes(t *testing.T) {
	cue := time.Now()
	blank := b.BlankTape(name, Incrementer{cue})
	if _, err := blank.Write(data); err != nil {
		t.Fatalf("miniredis failed")
	}

	tape := b.RecordedTape(name, Incrementer{cue})
	d, err := tape.tape.Read()
	if err != nil {
		t.Fatalf("miniredis failed")
	}

	if !bytes.Equal(d, data) {
		t.Errorf("retrieved data doesn't match. expected %b, got %b\n", data, d)
	}
}
