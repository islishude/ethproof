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

The caller establishes trust in an anchor or RPC. These modes do not establish Ethereum consensus finality; the optional finality mode below adds that authentication. The deprecated `Verify*ProofPackage` names perform embedded-root verification only.

## Consensus finality (optional)

`--finality` adds a finalized-only Beacon light client inside the Go process:

```bash
./bin/ethproof verify tx --config config.json --finality
./bin/ethproof verify receipt --config config.json --finality
./bin/ethproof verify state --config config.json --finality
```

Configure the top-level `finality` object in [config.example.json](config.example.json):

- `network`: `mainnet` or `sepolia`; it must match the proof and verification RPC chain ID.
- `beaconRpc`: one Beacon API supporting light-client bootstrap, updates, and finality updates.
- `checkpointRpcs`: 3–16 explicitly trusted, independently operated Beacon API endpoints on distinct hosts. All must answer genesis, finality-checkpoint, and header queries and confirm the selected checkpoint as canonical, finalized, and not execution-optimistic. No majority fallback is used. Distinct hosts alone do not guarantee independent operators.
- `cacheDir`: optional local trusted-state directory; defaults to `os.UserCacheDir()/ethproof/finality/<network>/`.
- `maxAncestry`: maximum number of execution parent headers to traverse, default 8192. This limits distance from the available finalized anchor, not a fixed wall-clock age.
- `timeoutSeconds`: total finality-mode verification timeout, default 300 and maximum 86400.

Set `verify.<kind>.finality` to `true` to enable it in configuration; `--finality=false` overrides it. Verification still requires its separately configured single execution RPC. Generation and proof JSON are unchanged. State proofs describe the finalized block's state, not necessarily the latest state.

The client authenticates bootstrap, sync committees, BLS aggregate signatures, finality branches, and execution inclusion. It then checks execution-header hashes and ancestry before applying the existing MPT verification. It does not run an EVM, P2P node, RPC server, optimistic-head tracker, or forced-update path. The accepted finality has Ethereum sync-committee light-client security assumptions, not full-node consensus re-execution.

**Checkpoint providers are an explicit external trust boundary.** On first use, the client chooses the earliest reported finalized checkpoint and asks every configured provider to confirm that exact root. The latest finalized roots need not match. The cached finalized header must be less than 24 hours old at startup; this is a conservative application policy, not a computed weak-subjectivity period. An expired cache automatically triggers a new multi-provider trust acquisition. Every provider must also confirm the old finalized root remains canonical/finalized; unavailable old history or any conflict stops recovery without overwriting the cache. The CLI reports each new trust acquisition. Initial and latest acquisition records are retained in the cache.

The cache is locally trusted, locked, atomically replaced, and bound to the network/fork schedule and checkpoint-provider configuration. Do not import another party's cache as a proof. Corrupt caches or changed trust configurations fail explicitly: use a new `cacheDir` to deliberately establish a new trust context. A clock rollback that puts cached state in the future is rejected. Provider URLs and credentials are not stored in the cache or emitted in transport errors.

Fork support is explicit: Capella, Deneb, Electra and Fulu are supported, including transitions. Other forks, including Gloas, fail closed; a network whose configured active fork has advanced beyond Fulu requires a future implementation update. Network fork schedules come from the pinned geth dependency. Historical execution blocks are supported only through the bounded parent-header chain. There is no portable offline finality-evidence format in this version.

Library users can construct `NewFinalityClient(config, executionSource)` and call the three `Verify*ProofPackageAgainstFinality` APIs (receipt also has a `WithExpectations` variant). `FinalizedAnchor` exposes the authenticated target `BlockAnchor` for reuse. `NewFinalityClientWithDependencies` accepts custom Beacon/checkpoint sources, a local store, a clock and a trust-change callback; injected components remain caller-owned and must honor cancellation. The library is silent by default.

An opt-in live smoke test uses the same configuration and a recent transaction proof:

```bash
ETH_PROOF_FINALITY_LIVE_CONFIG=/absolute/path/config.json go test -v ./proof -run '^TestFinalityLive$'
```

The default test suite uses pinned official SSZ proof vectors, historical Helios responses with real signatures, deterministic signed state-machine scenarios, and local HTTP servers. Anvil tests exercise execution-layer compatibility; they do not simulate PoS finality.

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
