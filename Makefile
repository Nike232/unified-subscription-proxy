SHELL := /bin/zsh

.PHONY: test build run-control run-proxy

test:
	go test ./...

build:
	mkdir -p bin
	go build -o bin/control-plane ./apps/control-plane
	go build -o bin/proxy-core ./apps/proxy-core

run-control:
	go run ./apps/control-plane

run-proxy:
	go run ./apps/proxy-core

