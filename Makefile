.PHONY: clean all test build

all: test

test:
	go test

build:
	go build

clean:
	go clean
