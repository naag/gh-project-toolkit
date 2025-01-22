.PHONY: build test lint clean

build:
	go build -o bin/gh-project-toolkit ./cmd/gh-project-toolkit

test:
	go test -v ./...

lint:
	go vet ./...
	test -z "$$(gofmt -l .)"

clean:
	rm -rf bin/ 