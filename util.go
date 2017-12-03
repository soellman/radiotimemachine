package main

import (
	"errors"
	"io"
	"strings"
	"time"

	"github.com/tcolgate/mp3"
)

// PATH FUNCTIONS

// StreamPath returns the path components
type streamPath struct {
	stationName      string
	listenerLocation *time.Location
}

func ParsePath(path string) (*streamPath, error) {
	pieces := strings.Split(path, "/")
	if len(pieces) != 3 {
		return nil, errors.New("incorrect number of path elements")
	}

	name := pieces[0]
	locName := pieces[1] + "/" + pieces[2]
	loc, err := time.LoadLocation(locName)
	if err != nil {
		return nil, err
	}

	return &streamPath{stationName: name, listenerLocation: loc}, nil
}

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

// IO FUNCTIONS

// ChunkPipe pumps reads from the reader to the writer
// with a specified chunksize. This will return when
// the reader finishes or errors.
func ChunkPipe(size int, r io.Reader, w io.Writer) error {
	b := make([]byte, size)

	for {
		if _, err := io.ReadFull(r, b); err != nil {
			if err == io.EOF {
				return nil
			}
			return err
		}
		if _, err := w.Write(b); err != nil {
			return err
		}
	}
}
