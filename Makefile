.PHONY: clean all test build

all: test

test:
	go test

build:
	go build

cover:
	go test -coverprofile .cover.out
	go tool cover -html=.cover.out

clean:
	go clean
