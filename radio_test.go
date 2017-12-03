package main

import (
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
