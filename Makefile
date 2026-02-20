.PHONY: test build
test:
	go test -v ./...

build:
	go build -o epp ./cmd/epp
