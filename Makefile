.PHONY: build test lint clean

build:
	go build -o bin/gh-project-toolkit ./cmd/gh-project-toolkit

test:
	go test -v ./...

lint:
	~/go/bin/golangci-lint run ./...

clean:
	rm -rf bin/

.PHONY: install-tools
install-tools:
	go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest 