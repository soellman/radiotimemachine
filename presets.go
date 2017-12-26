package main

import (
	"encoding/json"

	"github.com/pkg/errors"
)

type Presets struct {
	backend PresetBackend
}

type PresetBackend interface {
	ReadPreset(key string) (data []byte, err error)
	ReadAllPresets() (data [][]byte, err error)
	WritePreset(key string, data []byte) error
}

// does this need to return an error? ping during init or something?
func PresetsWithBackend(b PresetBackend) (*Presets, error) {
	return &Presets{b}, nil
}

func (p *Presets) Load() ([]*Station, error) {
	stations := []*Station{}

	data, err := p.backend.ReadAllPresets()
	if err != nil {
		return stations, errors.Wrap(err, "failed to read stations")
	}

	for _, d := range data {
		s, err := stationFromData(d)
		if err != nil {
			return nil, err
		}
		stations = append(stations, s)
	}
	return stations, nil
}

func (p *Presets) Add(s *Station) error {
	data, err := json.Marshal(s)
	if err != nil {
		return errors.Wrap(err, "failed to marshal station")
	}

	if err = p.backend.WritePreset(s.Name, data); err != nil {
		return errors.Wrap(err, "failed to write station")
	}

	return nil
}

// Lookup returns an initialized Station from the backend or an error
func (p *Presets) Lookup(name string) (*Station, error) {
	data, err := p.backend.ReadPreset(name)
	if err != nil {
		return nil, errors.Wrap(err, "station not found")
	}

	return stationFromData(data)
}

func stationFromData(data []byte) (*Station, error) {
	s := &Station{}
	err := json.Unmarshal(data, s)
	if err != nil {
		return nil, errors.Wrap(err, "failed to unmarshal station")
	}

	if err = s.Init(); err != nil {
		return nil, errors.Wrap(err, "failed to initialize station")
	}

	return s, nil
}
