package main

type Backend interface {
	Initable
}

// Init will test the validity of the backend configuration
type Initable interface {
	Init() error
}
