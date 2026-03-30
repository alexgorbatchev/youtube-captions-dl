binary := "youtube-captions-dl"

default:
    just --list

build:
    mkdir -p bin
    go build -o bin/{{binary}} .

test:
    go test ./...

test-race:
    go test -race ./...

fmt:
    go fmt ./...

vet:
    go vet ./...

lint:
    golangci-lint run ./...

verify:
    go mod tidy -diff
    go build ./...
    go vet ./...
    go test -race ./...
    golangci-lint run ./...

run url: build
    ./bin/{{binary}} '{{url}}'
