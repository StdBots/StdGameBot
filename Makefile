.PHONY: all build run clean docker-build

all: build

build:
	go mod tidy
	CGO_ENABLED=0 go build -ldflags="-w -s" -o bin/bot ./cmd/bot

run: build
	./bin/bot

clean:
	rm -rf bin/

docker-build:
	docker build -t stdgamebot .
