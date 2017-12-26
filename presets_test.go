package main

import (
	"encoding/json"
	"reflect"
	"testing"
)

var (
	ts = &Station{
		Name:     "wamc",
		Url:      "http://playerservices.streamtheworld.com/api/livestream-redirect/WAMCHD2.mp3",
		Location: "America/New_York",
	}
	tsjson = `{"name":"wamc","url":"http://playerservices.streamtheworld.com/api/livestream-redirect/WAMCHD2.mp3","location":"America/New_York"}`
)

func TestStationMarshal(t *testing.T) {
	data, err := json.Marshal(ts)
	if err != nil {
		t.Errorf("marshal failed: %v", err)
	}
	if string(data) != tsjson {
		t.Errorf("marshaled data. expected: %s, got %s", tsjson, data)
	}
}

type testPresetBackend struct {
	data map[string][]byte
}

func (b *testPresetBackend) ReadPreset(name string) (data []byte, err error) {
	return []byte{}, nil
}

func (b *testPresetBackend) ReadAllPresets() (data [][]byte, err error) {
	for _, d := range b.data {
		data = append(data, d)
	}
	return
}

func (b *testPresetBackend) WritePreset(name string, data []byte) error {
	b.data[name] = data
	return nil
}

func TestPresets(t *testing.T) {
	p, err := PresetsWithBackend(&testPresetBackend{
		data: make(map[string][]byte),
	})
	if err != nil {
		t.Errorf("PresetsWithBackend failed, %v", err)
	}

	err = p.Add(ts)
	if err != nil {
		t.Errorf("Presets.Add failed, %v", err)
	}

	stations, err := p.Load()
	if err != nil {
		t.Errorf("Presets.Load failed, %v", err)
	}

	ts.Init()
	expected := []*Station{ts}
	if !reflect.DeepEqual(stations, expected) {
		t.Errorf("Presets.Load didn't match. Expected %v, got %v", expected, stations)
	}
}
