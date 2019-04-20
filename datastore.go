package main

import (
	"context"

	"cloud.google.com/go/datastore"
)

// DatastoreBackend implements PresetBackend
type DatastoreBackend struct {
	client *datastore.Client
}

// Implements Backend
func (b *DatastoreBackend) Init() error {
	ctx := context.Background()
	client, err := datastore.NewClient(ctx, "") // inject via DATASTORE_PROJECT_ID
	if err != nil {
		return err
	}
	b.client = client
	return nil
}

type PresetEntity struct {
	Value []byte
}

// Implements PresetBackend
func (b DatastoreBackend) ReadPreset(name string) (data []byte, err error) {
	k := datastore.NameKey("Preset", name, nil)
	p := new(PresetEntity)
	b.client.Get(context.Background(), k, p)
	if err != nil {
		return
	}
	copy(data, p.Value)
	return
}

func (b DatastoreBackend) ReadAllPresets() (data [][]byte, err error) {
	presets := []PresetEntity{}
	if _, err = b.client.GetAll(context.Background(), datastore.NewQuery("Preset"), &presets); err != nil {
		return
	}

	for _, p := range presets {
		data = append(data, p.Value)
	}

	return
}

func (b DatastoreBackend) WritePreset(name string, data []byte) error {
	k := datastore.NameKey("Preset", name, nil)
	if _, err := b.client.Put(context.Background(), k, &PresetEntity{Value: data}); err != nil {
		return err
	}
	return nil
}
