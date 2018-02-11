package main

import (
	"fmt"

	"github.com/coreos/etcd/client"
	"golang.org/x/net/context"
)

// A EtcdBackend implements Backend and connects to etcd
// with a 24h expiration on stored entries
type EtcdBackend struct {
	host   string
	port   int
	client client.Client
}

// Implements Backend
func (b *EtcdBackend) Init() error {
	ep := fmt.Sprintf("http://%s:%d", b.host, b.port)
	cfg := client.Config{
		Endpoints: []string{ep},
		Transport: client.DefaultTransport,
	}
	c, err := client.New(cfg)
	if err != nil {
		return err
	}
	b.client = c
	return nil
}

// Implements RecordedTape
func (b *EtcdBackend) RecordedTape(ctx context.Context, name string, i Incrementer) (*RecordedTape, error) {
	return &RecordedTape{
		tape: &EtcdTape{
			name:   name,
			i:      i,
			client: b.client,
		},
	}, nil
}

// Implements BlankTape
func (b *EtcdBackend) BlankTape(ctx context.Context, name string, i Incrementer) (*BlankTape, error) {
	return &BlankTape{
		tape: &EtcdTape{
			name:   name,
			i:      i,
			client: b.client,
		},
	}, nil
}

// A EtcdTape implements BlankTape and RecordedTape
// and stores entries with an expiration according to TTL
type EtcdTape struct {
	name   string
	i      Incrementer
	client client.Client
}

func (t *EtcdTape) Write(data []byte) error {
	kAPI := client.NewKeysAPI(t.client)
	k := fmt.Sprintf("/chunk/%s/%s", t.name, t.i.Key())
	_, err := kAPI.Set(context.Background(), k, string(data), &client.SetOptions{TTL: TTL})
	return err
}

func (t *EtcdTape) Read() ([]byte, error) {
	kAPI := client.NewKeysAPI(t.client)
	k := fmt.Sprintf("/chunk/%s/%s", t.name, t.i.Key())
	r, err := kAPI.Get(context.Background(), k, nil)
	if err != nil {
		return []byte{}, err
	}
	return []byte(r.Node.Value), nil
}

// Implements PresetBackend
func (b *EtcdBackend) ReadPreset(name string) (data []byte, err error) {
	kAPI := client.NewKeysAPI(b.client)
	k := fmt.Sprintf("/preset/%s", name)
	r, err := kAPI.Get(context.Background(), k, nil)
	if err != nil {
		return []byte{}, err
	}
	return []byte(r.Node.Value), nil
}

func (b *EtcdBackend) ReadAllPresets() (data [][]byte, err error) {
	kAPI := client.NewKeysAPI(b.client)
	r, e := kAPI.Get(context.Background(), "/preset", &client.GetOptions{Recursive: true})
	if e != nil {
		err = e
		return
	}

	for _, n := range r.Node.Nodes {
		data = append(data, []byte(n.Value))
	}

	return
}

func (b *EtcdBackend) WritePreset(name string, data []byte) error {
	kAPI := client.NewKeysAPI(b.client)
	k := fmt.Sprintf("/preset/%s", name)
	_, err := kAPI.Create(context.Background(), k, string(data))
	return err
}
