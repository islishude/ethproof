# eth-proof

Generate and verify Ethereum Merkle Patricia Trie proofs in Go, with a CLI and Solidity storage-slot resolver.

| Proof | Claim |
| --- | --- |
| State | An account and its storage values are included under `stateRoot`. |
| Transaction | A transaction is included under `transactionsRoot`. |
| Receipt / event | A transaction and receipt occupy the same index under their respective roots, and the receipt contains the claimed event. |

Each live operation uses one RPC. Proofs are checked locally; trusting the block header is a separate requirement. See [verification modes](#verification-modes) and [internal design](design.md).

## Quick start

Use the Go version specified in [go.mod](go.mod). Build the CLI and fixture generator:

```bash
make build
# Or install ethproof into your Go bin directory:
make install
```

Copy [config.example.json](config.example.json) to `config.json` and replace the example RPC URLs, proof inputs, and event expectations. Then run:

```bash
./bin/ethproof generate state --config config.json
./bin/ethproof generate receipt --config config.json
./bin/ethproof generate tx --config config.json

./bin/ethproof verify state --config config.json
./bin/ethproof verify receipt --config config.json
./bin/ethproof verify tx --config config.json
```

Each section uses a single `rpc` string. Verification reads `verify.<kind>.rpc`; it never falls back to generation configuration. The same URL may be explicitly configured for both.

Flags override configuration. `--rpc` may appear once and must be non-empty; `--slot` and `--expect-topic` are repeatable. Storage slots accept 1–64 hexadecimal digits with a `0x` prefix. Receipt assertions use `--expect-emitter`, `--expect-topic`, and `--expect-data`.

```bash
./bin/ethproof verify tx --config config.json --rpc https://verify-rpc.example
./bin/ethproof --help
```

Generation writes proof JSON to the configured output path. Generation and verification report status on `stderr`; the library emits no runtime logs.

## Verification modes

| Go API suffix | What it establishes |
| --- | --- |
| `AgainstEmbeddedRoots` | Cryptographic consistency with the roots inside the package; no network access or header authentication. |
| `AgainstBlockAnchor` | Local verification plus a match with a caller-trusted block anchor. |
| `AgainstRPC` / `AgainstSource` | Local verification plus a match with a freshly fetched header from the configured endpoint. Used by CLI verification. |

The caller establishes trust in an anchor or RPC. None of these APIs establishes Ethereum consensus finality. The deprecated `Verify*ProofPackage` names perform embedded-root verification only.

## Go API

Generation requests contain only proof inputs. Pass a URL separately; the helper owns the connection:

```go
req := proof.TransactionProofRequest{TxHash: txHash}
pkg, err := proof.GenerateTransactionProofFromRPC(ctx, rpcURL, req)
if err != nil {
    return err
}
return proof.VerifyTransactionProofPackageAgainstRPC(ctx, pkg, verifyRPCURL)
```

For custom transport, authentication, or caching, use `GenerateTransactionProof(ctx, source, req)` and `VerifyTransactionProofPackageAgainstSource(ctx, pkg, source)`. State and receipt flows follow the same pattern; receipt verification also accepts optional `ReceiptExpectations`.

Implement the relevant interface in [proof/source.go](proof/source.go). Injected sources remain caller-owned and must honor context cancellation. Receipt sources must support concurrent `TransactionReceipt` calls. Request and proof types are in [proof/types.go](proof/types.go).

For trusted offline verification:

```go
anchor, err := proof.BlockAnchorFromHeader(chainID, trustedHeader)
if err != nil {
    return err
}
return proof.VerifyTransactionProofPackageAgainstBlockAnchor(pkg, anchor)
```

## Resolve storage slots

`resolve slot` accepts raw Solidity `storageLayout`, Foundry artifacts, or Hardhat build-info. `--contract` is required for artifact/build-info inputs and must be omitted for raw layouts.

```bash
./bin/ethproof resolve slot \
  --compiler-output out/ProofComplexDemo.sol/ProofComplexDemo.json \
  --contract ProofComplexDemo \
  --var 'balances[0x1111111111111111111111111111111111111111]' \
  --format artifact
```

Queries support fields (`config.owner`), indexed mappings/arrays (`data[4][9].b`), and dynamic bytes/string words (`blob@word(1)`). Dynamic containers require explicit indices. A bare bytes/string path returns its head slot; `@word(n)` returns the slot at `keccak256(headSlot) + n`.

Output contains logical `slots` and deduplicated `proofSlots` for `generate state --slot`. Packed fields can share a physical slot. JSON goes to `stdout`, or to the file specified by `--out`.

## Development

[Makefile](Makefile) defines the supported commands. Go is required for builds; lint uses `golangci-lint`. Formatting and contract builds use Foundry; local E2E also requires Docker Compose.

| Command | Purpose |
| --- | --- |
| `make build` | Build `bin/ethproof` and `bin/mkfixtures`. |
| `make lint` / `make fmt-check` | Static analysis and formatting checks. |
| `make fmt` | Apply Go formatting/fixes and Solidity formatting. |
| `make unit-test` | Offline-stable tests with the race detector. |
| `make e2e-test` | Start Anvil and test API, CLI, and storage resolution. |
| `make test` | Unit tests and E2E. |
| `make fixtures` | Regenerate deterministic [proof fixtures](proof/testdata). |
| `make bindings` | Rebuild contracts and generated Go bindings. |
| `make ci` | All checks, including binding drift and E2E. |

E2E uses [docker-compose.yml](docker-compose.yml) and chain ID `31337`. The default endpoint is `http://127.0.0.1:8545`; override it with `ETH_PROOF_E2E_RPC`. Direct Go test runs enable E2E only when `ETH_PROOF_REQUIRE_E2E=1`.

Bindings use geth `abigen v1` through [scripts/generate_bindings.sh](scripts/generate_bindings.sh); do not hand-edit generated files or use `--v2`. Compiler settings live in [foundry.toml](foundry.toml).
