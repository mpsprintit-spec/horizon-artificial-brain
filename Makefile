.PHONY: help fmt test validate build

help:
	@echo "Targets: fmt test validate build"

fmt:
	go fmt ./...

test:
	go test ./...

validate: fmt test
	go vet ./...

build:
	go build ./...
