.PHONY: build test fixtures live-test bindings e2e-up e2e-down e2e-test e2e fmt-check lint ci

all: build test fmt-check lint

install:
	go install ./cmd/ethproof

build:
	mkdir -p bin && go build -o ./bin ./cmd/...

test:
	go test -v -race ./...

fmt-check:
	gofmt -d .
	forge fmt --check

lint:
	golangci-lint run ./...

fmt: 
	gofmt -w .
	forge fmt

fixtures:
	go run ./cmd/mkfixtures --out-dir ./proof/testdata

bindings:
	sh ./scripts/generate_bindings.sh

e2e-up:
	docker compose up -d --wait anvil

e2e-down:
	docker compose down -v

e2e-test:
	set -e; \
		docker compose down; \
		docker compose up -d --wait; \
		trap 'docker compose down' EXIT; \
	ETH_PROOF_REQUIRE_E2E=1 go test -v -race -count=1 ./...

e2e: bindings e2e-test

live-test:
	@test -n "$(ETH_PROOF_RPCS)" || (echo "ETH_PROOF_RPCS is required"; exit 1)
	@test -n "$(ETH_PROOF_LIVE_TX)" || (echo "ETH_PROOF_LIVE_TX is required"; exit 1)
	@test -n "$(ETH_PROOF_LIVE_LOG_INDEX)" || (echo "ETH_PROOF_LIVE_LOG_INDEX is required"; exit 1)
	@test -n "$(ETH_PROOF_LIVE_STATE_BLOCK)" || (echo "ETH_PROOF_LIVE_STATE_BLOCK is required"; exit 1)
	@test -n "$(ETH_PROOF_LIVE_ACCOUNT)" || (echo "ETH_PROOF_LIVE_ACCOUNT is required"; exit 1)
	@test -n "$(ETH_PROOF_LIVE_SLOT)" || (echo "ETH_PROOF_LIVE_SLOT is required"; exit 1)
	go test ./proof -run TestLiveGenerateAndVerify -count=1
