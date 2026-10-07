.PHONY: help fmt test validate build bridge-smoke

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

bridge-smoke:
	bash scripts/bridge_smoke.sh
