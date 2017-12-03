package main

import (
	"crypto/rand"
	"io"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestLocationDistanceInMinutes(t *testing.T) {
	ots := []struct {
		ts      string
		src     string
		dst     string
		minutes int
	}{
		{"Sun Jul 30 10:13:25 2017", "America/New_York", "Europe/Stockholm", -360},
		{"Sun Jul 30 10:13:25 2017", "Europe/Stockholm", "America/New_York", 360},
		{"Sun Mar 19 10:13:25 2017", "Europe/Stockholm", "America/New_York", 300}, // US has already started DST but not Europe yet
		{"Wed Nov 1 10:13:25 2017", "Europe/Stockholm", "America/New_York", 300},  // Europe has finished DST but US not yet
		{"Wed Nov 1 10:13:25 2017", "America/Los_Angeles", "America/New_York", -180},
	}

	for i, ot := range ots {
		ts, _ := time.Parse(time.ANSIC, ot.ts)
		sloc, _ := time.LoadLocation(ot.src)
		dloc, _ := time.LoadLocation(ot.dst)
		d := LocationDistanceInMinutes(ts, sloc, dloc)
		if d != ot.minutes {
			t.Errorf("distance wrong. trial %d expected %d, got %d", i, ot.minutes, d)
		}
	}
}

func TestDetect(t *testing.T) {
	dts := []struct {
		filename string
		valid    bool
		bitrate  int
	}{
		{"falling.mp3", true, 128000},
		{"wamc.mp3", true, 64000},
		{"zero.dd", false, 0},
	}

	for _, dt := range dts {
		f, err := os.Open(filepath.Join("fixtures", dt.filename))
		if err != nil {
			t.Errorf("unable to open test fixture")
			continue
		}
		defer f.Close()

		bps, err := DetectBitrate(f)
		if dt.valid && err != nil {
			t.Errorf("unexpected failure to determine bitrate")
			continue
		}

		if bps != dt.bitrate {
			t.Errorf("detected bitrate was incorrect")
		}
	}
}

type testWriter struct {
	writes int
	bytes  int
}

func (t *testWriter) Write(p []byte) (n int, err error) {
	t.writes += 1
	t.bytes += len(p)
	return len(p), nil
}

func TestTestWriter(t *testing.T) {
	tw := testWriter{}
	b := make([]byte, 130)
	tw.Write(b)
	if tw.writes != 1 || tw.bytes != 130 {
		t.Error("testWriter busted")
	}

	b = make([]byte, 270)
	tw.Write(b)
	if tw.writes != 2 || tw.bytes != 400 {
		t.Error("testWriter busted")
	}
}

type testReader struct {
	maxreads int
	reads    int
}

func (t *testReader) Read(p []byte) (n int, err error) {
	if t.reads == t.maxreads {
		return 0, io.EOF
	}
	t.reads += 1
	return rand.Read(p)
}

func TestTestReader(t *testing.T) {
	tr := testReader{maxreads: 4}
	b := []byte{}
	for {
		if _, err := tr.Read(b); err != nil {
			break
		}
	}
	if tr.reads != 4 {
		t.Error("testReader busted")
	}
}
func TestChunkPipe(t *testing.T) {
	cs := 64000
	times := 8
	r := &testReader{maxreads: times}
	w := &testWriter{}
	ChunkPipe(cs, r, w)
	if w.writes != times || w.bytes != cs*times {
		t.Error("ChunkPipe busted")
	}
}
