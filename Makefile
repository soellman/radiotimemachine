BINARY := radiotimemachine

.PHONY: clean all test build

all: test

test:
	go test

build:
	go build

linux-static:
	GOOS=linux go build -a --ldflags '-extldflags "-static"' -tags netgo -installsuffix netgo -o ${BINARY}-linux

cover:
	go test -coverprofile .cover.out
	go tool cover -html=.cover.out

clean:
	go clean
