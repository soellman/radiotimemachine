package main

import (
	"encoding/json"
	"fmt"

	"github.com/pkg/errors"
)

type Presets struct {
	Backend PresetBackend
}

type PresetBackend interface {
	Read(key string) (data []byte, err error)
	ReadAll() (data [][]byte, err error)
	Write(key string, data []byte) error
}

// does this need to return an error? ping during init or something?
func PresetsWithBackend(b PresetBackend) (*Presets, error) {
	return &Presets{b}, nil
}

func (p *Presets) Load() ([]*Station, error) {
	st := []*Station{}

	data, err := p.Backend.ReadAll()
	if err != nil {
		return st, errors.Wrap(err, "failed to read stations")
	}

	for _, d := range data {
		station := &Station{}
		if err = json.Unmarshal(d, station); err != nil {
			return st, errors.Wrap(err, "failed to unmarshal station")
		}
		st = append(st, station)
	}
	return st, nil
}

func (p *Presets) Add(s *Station) error {
	data, err := json.Marshal(s)
	if err != nil {
		return errors.Wrap(err, "failed to marshal station")
	}

	key := fmt.Sprintf("preset-%s", s.Name)
	if err = p.Backend.Write(key, data); err != nil {
		return errors.Wrap(err, "failed to write station")
	}

	return nil
}
