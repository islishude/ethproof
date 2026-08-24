# Ethereum Proof Generator/Verifier in Go

This project generates and verifies three Ethereum Merkle Patricia Trie proof types:

- `state`: account proof + one or more storage proofs against `stateRoot`
- `receipt(event)`: receipt inclusion proof against `receiptsRoot`, then event matching inside the receipt
- `transaction`: transaction inclusion proof against `transactionsRoot`

It ships with deterministic offline fixtures under [proof/testdata](./proof/testdata) plus a local Anvil-backed e2e path that deploys demo contracts, generates all three proof types from a real local transaction, and validates them through both the Go API and the CLI.

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
transaction ⊂ transactions trie -> transactionsRoot
receipt index == transaction index
```

Generation fetches the containing block and all receipts, rebuilds both tries locally, and proves that the claimed transaction and event-bearing receipt occupy the same index. Receipt proofs generated before `transactionProofNodes` was added are intentionally rejected and must be regenerated.

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

The default minimum is `3` distinct configured RPC sources. This can be overridden per request with `--min-rpcs`; the local Anvil e2e uses `--min-rpcs 1` explicitly. The code can enforce unique configured URLs or source names, but deployment across independent operators remains the caller’s responsibility.

## Commands

The CLI is now primarily config-driven. Start from [config.example.json](./config.example.json) and pass `--config`; explicit flags still override the matching config fields.

Addresses, transaction hashes, topics, event data, and mapping byte keys are parsed strictly. Storage slots may use 1–64 hexadecimal digits and are left-padded to 32 bytes; malformed or oversized values are rejected instead of truncated.

The CLI keeps runtime output minimal. Successful `generate`, `verify`, and `mkfixtures` runs print a short status line to `stderr`; `resolve slot` writes JSON to `stdout` unless `--out` is set. Help text still prints to `stdout`, and usage/runtime errors still print to `stderr`. The `proof` package itself is silent by default and does not emit runtime logs.

## Build and install

```bash
make build
```

This writes both binaries to `./bin/`:

- `./bin/ethproof`
- `./bin/mkfixtures`

Install `ethproof` into your `$GOPATH/bin` with:

```bash
make install
```

### Generate state proof

```bash
ethproof generate state --config ./config.example.json
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

`resolve slot` resolves a Solidity variable path into concrete storage slot metadata from compiler output. Its `slots` array describes logical fields and may repeat a packed physical slot; `proofSlots` is deduplicated in first-seen order and can be passed directly into `generate state --slot`.

The command supports three input shapes:

- raw Solidity `storageLayout` JSON
- Foundry artifacts that include `storageLayout`
- Hardhat build-info JSON via `output.contracts`

`--contract` is required and validated for Foundry artifact and build-info inputs. Raw `storageLayout` input already represents one selected layout, so it must not be combined with `--contract`.

Dynamic containers must be indexed explicitly. Examples:

- `value`: a top-level storage variable.
- `config.owner`: a struct member path.
- `balances[0x1111111111111111111111111111111111111111]`: a mapping entry keyed by the given address.
- `data[4][9].b`: nested array or mapping indexing followed by a struct field selection.
- `blob@word(1)`: the second 32-byte data word of a dynamic `bytes` or `string` value. `@word(n)` is only valid as the final suffix on a `bytes` / `string` query and addresses the storage slot at `keccak256(headSlot) + n`.

For dynamic `bytes` / `string` values, the bare variable path such as `blob` resolves the head slot, while `blob@word(0)`, `blob@word(1)`, and so on resolve individual 32-byte words from the value's hashed data area.

Without `--out`, the resolved JSON is written to `stdout`. With `--out`, it is saved to the requested file.

Foundry artifact example:

```bash
ethproof resolve slot \
  --compiler-output ./out/ProofComplexDemo.sol/ProofComplexDemo.json \
  --contract ProofComplexDemo \
  --var 'balances[0x1111111111111111111111111111111111111111]' \
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
  --min-rpcs 3
```

`verify receipt` proves both receipt and transaction inclusion at the same index and validates the embedded event. `--expect-*` flags add assertions on top of the package’s claims, and CLI verify re-fetches the block header from the independent verify RPC set to authenticate the included roots.

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

`SourceName()` values must be non-empty and unique, but are never persisted or added by the framework to errors. Generated `block.sourceConsensus.rpcs` contains only opaque values such as `source[0]`. `SourceConsensus` is unauthenticated generation metadata, not evidence that independent operators agreed.

For trusted offline verification, construct a `BlockAnchor` from a header obtained through your own trust/finality mechanism and use the `...AgainstBlockAnchor` APIs:

```go
anchor, err := proof.BlockAnchorFromHeader(chainID, trustedHeader)
if err == nil {
    err = proof.VerifyTransactionProofPackageAgainstBlockAnchor(pkg, anchor)
}
```

The `...AgainstEmbeddedRoots` APIs verify cryptographic consistency against roots carried inside the package, but do not authenticate those roots. The older offline `Verify*ProofPackage` names remain deprecated wrappers for source compatibility.

## Offline fixtures

The repository includes three generated offline proof fixtures:

- [transaction_fixture.json](./proof/testdata/transaction_fixture.json)
- [receipt_fixture.json](./proof/testdata/receipt_fixture.json)
- [state_fixture.json](./proof/testdata/state_fixture.json)

These are deterministic synthetic Ethereum examples built with real Ethereum encodings and trie rules, so tests do not depend on network access.

Regenerate them with:

```bash
go run ./cmd/mkfixtures --out-dir ./proof/testdata
```

## Testing

Testing is split between an offline-stable path and a local e2e path:

- `make unit-test` runs `go test -v -race ./...`. `TestAnvilE2E` is skipped unless `ETH_PROOF_REQUIRE_E2E=1`, so this path remains offline-stable.
- `make e2e-test` starts Anvil with `docker compose` and runs only `./proof -run TestAnvilE2E`.
- `make test` runs the full suite: `make unit-test` followed by `make e2e-test`.
- `make ci` runs formatting, lint, build, race tests, binding regeneration with a scoped drift check, and Anvil E2E.

### Local Anvil E2E

The local e2e flow uses the checked-in [docker-compose.yml](./docker-compose.yml) plus [contracts/ProofComplexDemo.sol](./contracts/ProofComplexDemo.sol):

- `api_mainline` deploys `ProofComplexDemo`, then generates and verifies `transaction`, `receipt`, and `state` proofs from its `applyUpdate` transaction.
- `cli_mainline` resolves the `balances[caller]` slot from the `ProofComplexDemo` artifact, runs the CLI `generate` / `verify` flow, then runs a focused `resolve slot` regression for:
  - `balances[caller]`
  - `history[caller][2]`
  - `positions[caller][positionId].lastPrice`
  - `note@word(0)`
  - `payload@word(0)`
  - `basicUint256`, `basicAddress`, `basicBytes32`, and `basicBool`
  - packed struct fields, mixed struct fields, and packed fixed-array elements
- the complex resolver stage compares each resolved slot, offset, byte length, and type against the contract's actual storage word at the mined block, rather than generating a proof.

The e2e test expects Anvil on `http://127.0.0.1:8545` with chain ID `31337`. You can override the RPC URL with `ETH_PROOF_E2E_RPC`.

## Contract bindings

Go contract bindings are generated with geth `abigen v1`, not `--v2`.

The Foundry profile is pinned for deterministic output across platforms and toolchain updates:

- `solc_version = "0.8.35"`
- `evm_version = "osaka"`
- `bytecode_hash = "none"`
- `cbor_metadata = false`
- `via_ir = true`
- `extra_output = ["storageLayout"]`

This keeps `forge inspect ... bytecode` stable for binding generation instead of depending on Foundry defaults or metadata hashes embedded at the end of the bytecode.

The extra compiler output also makes repo-local Foundry artifacts usable with `ethproof resolve slot` directly from `out/*.json`.

Regenerate them with:

```bash
make bindings
```

The target runs `forge build`, then emits bindings for the local e2e contracts into [internal/e2e/bindings](./internal/e2e/bindings):

```bash
internal/e2e/bindings/proofcomplexdemo.go
internal/e2e/bindings/erc7201customlayoutdemo.go
```

## Notes

- `state` proofs use `eth_getProof`; `receipt` and `transaction` proofs are rebuilt locally from canonical block data.
- The library `...AgainstEmbeddedRoots` APIs remain fully offline but do not establish block authenticity. Use a caller-trusted `BlockAnchor` or the source/RPC-aware APIs for that boundary.
- CLI `verify` never reuses `block.sourceConsensus.rpcs`; the opaque identifiers and digests in that unauthenticated field are generation metadata only.
- Even with independent RPC anchoring, if you need bridge-grade security, you must separately verify that the block header itself is finalized and trusted.
- The code targets `github.com/ethereum/go-ethereum` `v1.17.x`.
