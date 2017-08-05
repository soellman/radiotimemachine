package main

import (
	"crypto/rand"
	"io"
	"os"
	"path/filepath"
	"testing"
)

// TODO: make this do something
func TestListen(t *testing.T) {
	dts := []struct {
		filename string
		bitrate  int
	}{
		{"wamc.mp3", 64000},
	}

	for _, dt := range dts {
		f, err := os.Open(filepath.Join("fixtures", dt.filename))
		if err != nil {
			t.Errorf("unable to open test fixture")
			continue
		}
		defer f.Close()
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

func TestChunkPump(t *testing.T) {
	cs := 64000
	times := 8
	r := &testReader{maxreads: times}
	w := &testWriter{}
	ChunkPump(cs, r, w)
	if w.writes != times || w.bytes != cs*times {
		t.Error("ChunkPump busted")
	}
}
