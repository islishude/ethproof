.PHONY: build unit-test test fixtures bindings e2e-up e2e-down e2e-test e2e fmt-check lint ci

all: fmt-check lint build test e2e-test

install:
	go install -trimpath -ldflags="-s -w" ./cmd/ethproof

build:
	mkdir -p bin && go build -o ./bin ./cmd/...

fmt-check:
	gofmt -d .
	go fix -diff ./...
	forge fmt --check

fmt: 
	gofmt -w -s .
	go fix ./...
	forge fmt

lint:
	go vet ./...
	golangci-lint run ./...

fixtures:
	go run ./cmd/mkfixtures --out-dir ./proof/testdata

bindings:
	sh ./scripts/generate_bindings.sh

e2e-up:
	docker compose up -d --wait anvil

e2e-down:
	docker compose down -v

unit-test:
	go test -v -race ./...

e2e-test:
	set -e; \
		docker compose down; \
		docker compose up -d --wait; \
		trap 'docker compose down' EXIT; \
	ETH_PROOF_REQUIRE_E2E=1 go test -v -race -count=1 ./proof -run TestAnvilE2E$$

e2e: bindings e2e-test

test: unit-test e2e-test
