package main

import (
	"fmt"
	"io/ioutil"

	"cloud.google.com/go/storage"
	"golang.org/x/net/context"
)

// A GCSBackend implements Backend and connects to google cloud storage
type GCSBackend struct {
	bucket string
}

// Implements Backend
func (b *GCSBackend) Init() error {
	client, err := storage.NewClient(context.Background())
	if err != nil {
		return err
	}

	handle := client.Bucket(b.bucket)
	_, err = handle.Attrs(context.Background())

	return err
}

// Implements RecordedTape
func (b GCSBackend) RecordedTape(ctx context.Context, name string, i Incrementer) (*RecordedTape, error) {
	client, err := storage.NewClient(ctx)
	handle := client.Bucket(b.bucket)

	return &RecordedTape{
		tape: &GCSTape{
			ctx:    ctx,
			handle: handle,
			name:   name,
			i:      i,
		},
	}, err
}

// Implements BlankTape
func (b GCSBackend) BlankTape(ctx context.Context, name string, i Incrementer) (*BlankTape, error) {
	client, err := storage.NewClient(ctx)
	handle := client.Bucket(b.bucket)

	return &BlankTape{
		tape: &GCSTape{
			handle: handle,
			name:   name,
			i:      i,
		},
	}, err
}

// A GCSTape implements BlankTape and RecordedTape
// and stores entries with an expiration according to TTL
type GCSTape struct {
	name   string
	i      Incrementer
	handle *storage.BucketHandle
	ctx    context.Context
}

func (t *GCSTape) Write(data []byte) error {
	name := fmt.Sprintf("%s/%s.chunk", t.name, t.i.Key())
	w := t.handle.Object(name).NewWriter(t.ctx)
	defer w.Close()

	if _, err := w.Write(data); err != nil {
		return err
	}

	return nil
}

func (t *GCSTape) Read() ([]byte, error) {
	name := fmt.Sprintf("%s/%s.chunk", t.name, t.i.Key())

	r, err := t.handle.Object(name).NewReader(t.ctx)
	if err != nil {
		return []byte{}, err
	}
	defer r.Close()

	return ioutil.ReadAll(r)
}
