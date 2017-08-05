package main

import (
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
