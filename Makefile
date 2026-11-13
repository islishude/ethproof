.PHONY: all install build unit-test test fixtures bindings e2e-up e2e-down e2e-test e2e fmt-check fmt lint ci

all: ci

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
	docker compose up --quiet-pull -d --wait anvil

e2e-down:
	docker compose down -v

unit-test:
	go test -v -race ./...

e2e-test:
	set -e; \
		docker compose down; \
		docker compose up --quiet-pull -d --wait; \
		trap 'docker compose down' EXIT; \
	ETH_PROOF_REQUIRE_E2E=1 go test -v -race -count=1 ./proof -run TestAnvilE2E$$

e2e: bindings e2e-test

test: unit-test e2e-test

ci: fmt-check lint build unit-test
	$(MAKE) bindings
	git diff --exit-code -- internal/e2e/bindings
	$(MAKE) e2e-test

# Real PoS finality tests are separate from the fast Anvil suite.
.PHONY: pos-e2e-up pos-e2e-down pos-e2e-test
pos-e2e-up:
	sh scripts/pos_e2e.sh up

pos-e2e-down:
	sh scripts/pos_e2e.sh down

pos-e2e-test:
	sh scripts/pos_e2e.sh test
