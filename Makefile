BINARY   := radiotimemachine
VERSION  ?= latest
IMAGE    := ${BINARY}:${VERSION}

.PHONY: clean all test build

all: test

test:
	go test

cover:
	go test -coverprofile .cover.out
	go tool cover -html=.cover.out

clean:
	rm -f ${BINARY} ${BINARY}-linux
	go clean

build:
	go build

linux-static:
	GOOS=linux go build -a --ldflags '-extldflags "-static"' -tags netgo -installsuffix netgo -o ${BINARY}-linux

docker:
	docker build -t ${IMAGE} .

