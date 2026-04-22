# Ethereum Proof Generator/Verifier in Go

This project generates and verifies three Ethereum Merkle Patricia Trie proof types:

- `state`: account proof + one or more storage proofs against `stateRoot`
- `receipt(event)`: receipt inclusion proof against `receiptsRoot`, then event matching inside the receipt
- `transaction`: transaction inclusion proof against `transactionsRoot`

It ships with deterministic offline fixtures under [proof/testdata](/Users/sudoless/codespace/coding/eth-proof/proof/testdata) plus a local Anvil-backed e2e path that deploys a minimal test contract, generates all three proof types from a real local transaction, and validates them through both the Go API and the CLI.

## Proof model

### State proof

```text
account ⊂ state trie -> stateRoot
slot[i] ⊂ storage trie(account.storageRoot) -> storageRoot ⊂ account
```

Generation uses `eth_getProof` for account/storage proofs, but verification is fully local:

1. verify `accountProof` against `stateRoot` and `keccak(address)`
2. decode the verified account RLP and check `nonce/balance/storageRoot/codeHash`
3. verify each `storageProofs[i]` against `account.storageRoot` and `keccak(slot[i])`
4. normalize each storage value and compare it with the claimed slot value

### Receipt / event proof

```text
log ⊂ receipt ⊂ receipts trie -> receiptsRoot
```

Generation fetches all receipts for the containing block, rebuilds the receipts trie locally, proves receipt inclusion, and stores the target event fields (`address/topics/data`) plus `txHash/logIndex`.

### Transaction proof

```text
transaction ⊂ transactions trie -> transactionsRoot
```

Generation fetches the containing block, rebuilds the transactions trie locally, proves inclusion for the target transaction, and stores the canonical transaction bytes.

## RPC consistency rules

Live generation is intentionally strict:

- you must pass at least 3 RPC sources
- every source must agree after normalization
- comparison is byte-level strict on the normalized data used to build the proof
- any mismatch fails the command immediately; there is no 2-of-3 quorum fallback

For `state` proofs the normalized data includes:

- block header context
- account proof nodes
- storage proofs in requested slot order
- account RLP / decoded account fields
- normalized storage slot values

For `receipt` and `transaction` proofs the normalized data includes:

- block header context
- target transaction bytes
- block transaction list
- block receipt list for receipt proofs
- target receipt bytes and target event fields for receipt proofs

The default minimum is `3` distinct RPC sources. This can be overridden per request with `--min-rpcs`; the local Anvil e2e uses `--min-rpcs 1` explicitly.

## Commands

The CLI is now primarily config-driven. Start from [config.example.json](/Users/sudoless/codespace/coding/eth-proof/config.example.json) and pass `--config`; explicit flags still override the matching config fields.

Runtime logs use the standard library `log/slog`. By default the CLI writes `info`-level text logs to `stderr`; `--log-level` and `--log-format` override the top-level `logging.level` / `logging.format` config values. Help text still prints to `stdout`, and usage errors still print to `stderr` without going through the logger. The `proof` package itself is silent by default and does not emit runtime logs.

## Build and install the CLI with:

```bash
make build
```

You'll have a compiled `ethproof` binary in `./bin/ethproof` that you can run with `--help` for usage info.

You can install it to your `$GOPATH/bin` with:

```bash
make install
```

### Generate state proof

```bash
ethproof generate state --config ./config.example.json

# or override runtime logging for a single invocation
ethproof generate state --config ./config.example.json --log-level debug --log-format json
```

`generate state` accepts repeatable `--slot` flags, and config uses `generate.state.slots`.

### Generate receipt / event proof

```bash
ethproof generate receipt --config ./config.example.json
```

### Generate transaction proof

```bash
ethproof generate tx --config ./config.example.json
```

### Resolve Solidity storage slots

`resolve slot` resolves a Solidity variable path into concrete storage slot metadata from compiler output. This is independent from `generate state`; use it to compute slot keys first, then pass the resolved slot(s) into `generate state --slot`.

The command supports three input shapes:

- raw Solidity `storageLayout` JSON
- Foundry artifacts that include `storageLayout`
- Hardhat build-info JSON via `output.contracts`

Dynamic containers must be indexed explicitly. Examples:

- `value`
- `config.owner`
- `balances[0x1111111111111111111111111111111111111111]`
- `data[4][9].b`
- `blob@word(1)`

Foundry artifact example:

```bash
ethproof resolve slot \
  --compiler-output ./out/ProofDemo.sol/ProofDemo.json \
  --contract ProofDemo \
  --var value \
  --format artifact
```

Hardhat build-info example:

```bash
ethproof resolve slot \
  --compiler-output ./artifacts/build-info/<build-info>.json \
  --contract contracts/MyContract.sol:MyContract \
  --var 'data[4][9].b' \
  --format build-info
```

### Verify proofs

`verify` requires its own independent RPC set in `verify.<kind>.rpcs` or via `--rpc`. It does not reuse the generation RPC list from the proof JSON or from `generate.*.rpcs`.

```bash
ethproof verify state --config ./config.example.json

ethproof verify receipt --config ./config.example.json

ethproof verify tx --config ./config.example.json
```

Example flag override:

```bash
ethproof verify tx \
  --config ./config.example.json \
  --rpc https://verify-rpc-1.example \
  --rpc https://verify-rpc-2.example \
  --rpc https://verify-rpc-3.example \
  --min-rpcs 3 \
  --log-level warn
```

`verify receipt` always validates all fields embedded in the proof package. `--expect-*` flags add extra assertions on top of the package’s own claims, and CLI verify also re-fetches the block header from the independent verify RPC set to anchor the included roots.

## Library Integration

The library now supports two integration styles:

- URL-driven helpers such as `GenerateStateProof` and `VerifyTransactionProofPackageAgainstRPCs`
- source-driven helpers such as `GenerateStateProofFromSources` and `VerifyTransactionProofPackageAgainstSources`

The source-driven APIs are intended for embedders that want to control transport, auth, retries, caching, or a forked `go-ethereum` build.

If your application uses a modified geth module, a normal module replacement is enough:

```go
replace github.com/ethereum/go-ethereum => ../your-go-ethereum-fork
```

Then wrap your client(s) behind the `proof.HeaderSource` / `proof.StateSource` / `proof.TransactionSource` / `proof.ReceiptSource` interfaces:

```go
type myStateSource struct {
	name string
	eth  *ethclient.Client
	geth *gethclient.Client
}

func (s *myStateSource) SourceName() string { return s.name }
func (s *myStateSource) ChainID(ctx context.Context) (*big.Int, error) {
	return s.eth.ChainID(ctx)
}
func (s *myStateSource) HeaderByHash(ctx context.Context, hash common.Hash) (*types.Header, error) {
	return s.eth.HeaderByHash(ctx, hash)
}
func (s *myStateSource) HeaderByNumber(ctx context.Context, num *big.Int) (*types.Header, error) {
	return s.eth.HeaderByNumber(ctx, num)
}
func (s *myStateSource) GetProof(ctx context.Context, account common.Address, slots []string, num *big.Int) (*gethclient.AccountResult, error) {
	return s.geth.GetProof(ctx, account, slots, num)
}

pkg, err := proof.GenerateStateProofFromSources(ctx, proof.StateProofSourcesRequest{
	Sources: []proof.StateSource{
		&myStateSource{name: "rpc-a", eth: ethA, geth: gethA},
		&myStateSource{name: "rpc-b", eth: ethB, geth: gethB},
		&myStateSource{name: "rpc-c", eth: ethC, geth: gethC},
	},
	MinRPCSources: 3,
	BlockNumber:   22_000_000,
	Account:       common.HexToAddress("0x..."),
	Slots: []common.Hash{
		common.HexToHash("0x..."),
		common.HexToHash("0x..."),
	},
})
```

`SourceName()` values are persisted into `block.sourceConsensus.rpcs`, so they must be non-empty and unique within each request.

## Offline fixtures

The repository includes three offline fixtures:

- [transaction_fixture.json](/Users/sudoless/codespace/coding/eth-proof/proof/testdata/transaction_fixture.json)
- [receipt_fixture.json](/Users/sudoless/codespace/coding/eth-proof/proof/testdata/receipt_fixture.json)
- [state_fixture.json](/Users/sudoless/codespace/coding/eth-proof/proof/testdata/state_fixture.json)

These are deterministic synthetic Ethereum examples built with real Ethereum encodings and trie rules, so tests do not depend on network access.

Regenerate them with:

```bash
go run ./cmd/mkfixtures --out-dir ./proof/testdata --log-level info
```

## Testing

The default test flow is offline and deterministic:

- `make test` runs the unit and offline integration suites with `go test -v -race ./...`.
- `make e2e-test` only runs the local Anvil-backed `TestAnvilE2E` mainline path.
- `TestAnvilE2E` is skipped unless `ETH_PROOF_REQUIRE_E2E=1`, so plain `go test ./...` remains offline-stable.

## Local Anvil e2e

The local e2e flow uses the checked-in [docker-compose.yml](/Users/sudoless/codespace/coding/eth-proof/docker-compose.yml) plus the two demo contracts:

- [contracts/ProofDemo.sol](/Users/sudoless/codespace/coding/eth-proof/contracts/ProofDemo.sol) drives the proof-generation mainline.
- [contracts/ProofComplexDemo.sol](/Users/sudoless/codespace/coding/eth-proof/contracts/ProofComplexDemo.sol) provides a narrow resolver compatibility regression against real Foundry artifacts and real chain storage.
- `api_mainline` deploys the simple contract, then generates and verifies `transaction`, `receipt`, and `state` proofs.
- `cli_mainline` first resolves the `value` slot from the simple contract artifact and runs the CLI `generate` / `verify` flow, then runs a focused `ProofComplexDemo` `resolve slot` regression for:
  - `balances[caller]`
  - `history[caller][2]`
  - `positions[caller][positionId].lastPrice`
  - `note@word(0)`
  - `payload@word(0)`
- the complex resolver stage compares each resolved slot against the contract's actual storage word at the mined block, rather than generating a proof.

Start the node and run e2e:

```bash
make e2e-test
```

The e2e test expects Anvil on `http://127.0.0.1:8545` with chain ID `1337`. You can override the RPC URL with `ETH_PROOF_E2E_RPC`.

## Contract bindings

Go contract bindings are generated with geth `abigen v1`, not `--v2`.

The Foundry profile is pinned for deterministic output across platforms and toolchain updates:

- `solc_version = "0.8.28"`
- `evm_version = "prague"`
- `bytecode_hash = "none"`
- `cbor_metadata = false`
- `extra_output = ["storageLayout"]`

This keeps `forge inspect ... bytecode` stable for binding generation instead of depending on Foundry defaults or metadata hashes embedded at the end of the bytecode.

The extra compiler output also makes repo-local Foundry artifacts usable with `ethproof resolve slot` directly from `out/*.json`.

Regenerate them with:

```bash
make bindings
```

The target runs `forge build`, then emits bindings for both demo contracts into [internal/e2e/bindings](/Users/sudoless/codespace/coding/eth-proof/internal/e2e/bindings):

```bash
internal/e2e/bindings/proofdemo.go
internal/e2e/bindings/proofcomplexdemo.go
```

## Notes

- `state` proofs use `eth_getProof`; `receipt` and `transaction` proofs are rebuilt locally from canonical block data.
- The library `Verify*ProofPackage` APIs remain fully offline. The CLI `verify` path now adds an independent RPC block-header check on top of the offline proof verification.
- CLI `verify` never reuses `block.sourceConsensus.rpcs`; that field remains generation metadata only.
- Even with independent RPC anchoring, if you need bridge-grade security, you must separately verify that the block header itself is finalized and trusted.
- The code targets `github.com/ethereum/go-ethereum` `v1.17.x`.
