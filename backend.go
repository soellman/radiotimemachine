package main

type Backend interface {
	Initable
}

// TODO: Init shouldn't take params, params should be fed in at creation time
type Initable interface {
	Init(host string, port int) error
}
