package main

import (
	"io"
	"time"

	"github.com/tcolgate/mp3"
)

// TIME FUNCTIONS

// LocationDistanceInMinutes returns the minutes relative to listener's location
// Since we're using locations, this should even work during the weird daylight savings week
func LocationDistanceInMinutes(t time.Time, loc, stationLoc *time.Location) int {
	ts := t.Format(time.ANSIC)
	locTime, _ := time.ParseInLocation(time.ANSIC, ts, loc)
	stationTime, _ := time.ParseInLocation(time.ANSIC, ts, stationLoc)
	return int(stationTime.Sub(locTime) / time.Minute)
}

// MP3 FUNCTIONS

// DetectBitrate will read a single mp3 frame from the
// reader, and return the bitrate in bits per second
// or an error if the bitrate cannot be determined
//
// DetectBitrate does not do what you want for VBR
// CBR only recommended
func DetectBitrate(r io.Reader) (bps int, err error) {
	d := mp3.NewDecoder(r)
	skipped := 0
	var f mp3.Frame

	if err = d.Decode(&f, &skipped); err != nil {
		return
	}

	bps = int(f.Header().BitRate())

	return
}
