package main

import (
	"testing"
	"time"
)

func TestIncrementer(t *testing.T) {
	ts := "Sun Jul 30 10:13:00 2017"
	it, _ := time.Parse(time.ANSIC, ts)
	i := Incrementer{
		name: "test",
		t:    it,
	}

	str := i.Key()
	if str != "test-2017-07-30T10:13:00Z" {
		t.Error("Incrementer Key() failed")
	}

	str = i.Key()
	if str != "test-2017-07-30T10:13:20Z" {
		t.Errorf("Incrementer Key() failed: got %s", str)
	}
}
