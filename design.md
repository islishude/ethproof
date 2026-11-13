# Internal design

For commands and integration examples, see [README.md](README.md). This document describes the implementation boundaries and correctness checks.

## Code ownership

| Area | Responsibility |
| --- | --- |
| [proof](proof) | Generation, verification, RPC response validation, JSON I/O, and storage resolution. |
| [cmd/ethproof](cmd/ethproof) | Parse configuration and flags, set timeouts, dispatch calls, and format output. |
| [internal/proofutil](internal/proofutil) | Ethereum encoding, trie construction, and proof-node helpers. |
| [cmd/mkfixtures](cmd/mkfixtures) | Deterministic fixtures using the public proof types. |

The three proof packages and their requests are defined in [proof/types.go](proof/types.go). Storage resolution is an auxiliary tool, not another proof type.

## Transport and block context

`Generate*Proof(ctx, source, req)` accepts a caller-owned source. `Generate*ProofFromRPC(ctx, rpcURL, req)` opens one connection, delegates to the same generator, and closes it on return. Requests contain only proof inputs.

[Source interfaces](proof/source.go) require only the methods each flow uses. Nil and typed-nil sources are rejected. Implementations honor cancellation; URL-backed errors redact credentials while preserving error unwrapping. Execution operations do not compare multiple nodes or fall back to another execution endpoint. The separate finality checkpoint-acquisition policy is described below.

`BlockContext` stores the chain ID, block number/hash, parent hash, and state/transaction/receipt roots. `BlockContextFromHeader` validates and projects a header without establishing trust. `BlockAnchor` is a distinct type with the same fields, supplied through a caller's trust mechanism.

Proof JSON contains neither RPC endpoints nor consensus metadata. `LoadJSON` ignores unknown fields, including legacy `sourceConsensus`. Receipt packages require transaction inclusion proof nodes; packages missing them are rejected.

## Generation

All three flows construct a proof package directly and locally verify it before returning.

### State

[State generation](proof/state_generate.go) fetches a header at the requested height and calls `eth_getProof` for the account and slots. It checks the returned height, sorts proof nodes for deterministic output, and restores storage proofs to requested-slot order.

Account proof verification derives the account RLP and checks nonce, balance, storage root, and code hash against the claim. Each storage proof is checked under that account's storage root. An empty storage trie requires zero values and no storage proof nodes. Empty or duplicate requested slots, unexpected response slots, and malformed values are rejected.

### Transaction

[Shared transaction loading](proof/transaction_data.go) fetches the transaction, its receipt, and the containing block. The receipt supplies the block hash and transaction index; both are checked against the block. The target transaction's hash and encoded bytes must match the entry at that index.

The block already includes its header, so generation makes no separate header-by-hash request. [Transaction generation](proof/transaction.go) rebuilds the transaction trie, checks `transactionsRoot`, and exports the target inclusion proof.

### Receipt / event

[Receipt generation](proof/receipt_generate.go) reuses the transaction data and builds both transaction and receipt proofs at the same index. The target receipt must match the corresponding block receipt, and both reconstructed trie roots must match the header.

[Receipt retrieval](proof/receipt_fetch.go) prefers `eth_getBlockReceipts`. Only a method-not-found error triggers per-transaction fallback, with at most 8 concurrent requests to the same node. Fallback reuses the target receipt, preserves transaction order, and cancels on failure.

Verification binds the transaction hash to the receipt index, checks matching transaction/receipt types, and compares the selected log's address, topics, and data. Caller expectations may additionally constrain emitter, topic prefix, or data.

## Verification and trust

Embedded-root verification checks proof nodes, canonical encodings, and claims locally. It does not authenticate the roots against the advertised block hash.

[Anchor verification](proof/block_anchor.go) additionally compares every block-context field with a caller-trusted anchor. [Online verification](proof/verify_rpc.go) performs local checks and event expectations first, then obtains chain ID and header from the configured source and applies the same comparison.

CLI verification uses only its own configured RPC or explicit flag. A matching header is trusted only to the extent the caller trusts that endpoint; neither RPC verification nor anchor construction establishes consensus finality.

## Finality light client

The optional finality path lives in `proof/`, sharing the existing `BlockAnchor` boundary. It adds no fourth proof type and does not change proof JSON. CLI verification first runs the existing local/RPC checks, then authenticates the block using the separately configured finality client. The library's finality wrappers compose local MPT verification and the same complete block-context checks through the client.

The client separates three responsibilities:

1. **External trust acquisition:** at least three configured checkpoint providers must match built-in genesis parameters and confirm one candidate root. The earliest advertised finalized epoch is selected; equal-epoch conflicting roots fail. Each provider must return matching header bytes and affirmative canonical/finalized flags. Those flags establish an externally accepted trust statement; they are not a locally verified consensus proof.
2. **Pure consensus verification:** bootstrap binds the committee to that root. Full/finality updates validate header execution inclusion, normalized state branches, time/period ordering, committee identity and BLS signatures. Signing domains use `signature_slot - 1`. Only supermajority updates can learn/rotate committees or advance finalized state. No optimistic state, ranking fallback or force-update is retained. The own update model avoids geth's full synchronizer's additional period restrictions while reusing geth's SSZ/Merkle/BLS primitives.
3. **Execution anchoring:** extract the execution block hash, hash-check its execution header, and traverse a bounded parent chain to the target. Comparing the resulting complete `BlockAnchor` also binds all three MPT roots. Beacon SSZ transactions roots are never substituted for execution MPT transaction roots.

On-demand calls use a context deadline and serialize cache read/verify/write transactions using an advisory file lock. The cache holds current/next committees, finalized header and execution proof, network/trust-policy fingerprint, and initial/latest acquisition records. It is trusted local state, not an independently verifiable exported proof. Files are written with restricted permissions through a synced temporary file and atomic rename. Credential-bearing URLs enter only the configuration fingerprint, never persisted records.

A cached finalized slot older than 24 hours triggers the configured external trust acquisition again. All checkpoint providers must confirm the prior finalized root too, and the new checkpoint cannot move backward. This is explicitly a fresh trust decision, not a claim that an arbitrarily old committee chain was cryptographically caught up. Cache corruption/configuration mismatch never triggers silent deletion. Failed refresh leaves existing state intact; successfully verified incremental progress may be saved even if the requested target has not yet finalized.

HTTP transports reject redirects, cap responses at 16 MiB, apply request deadlines, and redact transport/body errors while preserving cancellation through error unwrapping. No remote response may change the network parameters or trusted-provider list. Unknown fork versions fail closed. The v1 supported fork set is Capella through Fulu; pre-Capella and Gloas light-client data are deliberately unsupported.

Consensus behavior is based on the Altair light-client sync protocol with Capella/Deneb/Electra amendments. Reference material and immutable test-vector revisions are recorded in `proof/testdata/finality/`. Official mainnet vectors cover execution SSZ inclusion; real Helios response fixtures exercise bootstrap and committee progression, supplemented by signed boundary/negative cases. The default test suite never contacts public nodes.

## Storage resolution

[Layout loading](proof/storage_layout.go) accepts raw layouts, Foundry artifacts, and Hardhat build-info. Artifact/build-info inputs require a validated contract selector; raw layouts reject one.

[Path resolution](proof/storage_resolver.go) walks fields, mappings, arrays, and bytes/string `@word(n)` suffixes. It returns logical slot/offset/size entries plus first-seen, deduplicated `proofSlots`. Expansion rejects cyclic types and is bounded to 4096 fields and 64 recursive levels; larger arrays require explicit indexing.

## Test coverage

- Offline proof fixtures and golden tests check deterministic encoding and serialization.
- Generation tests cover malformed node responses, wrong block heights, root mismatches, and receipt fallback.
- Verification tests cover tampered proofs, event expectations, every anchor field, cancellation, and credential redaction.
- CLI tests cover configuration isolation, flag precedence, strict inputs, and output behavior.
- Anvil E2E exercises all three proof flows through Go and CLI, plus storage resolution checked against deployed contracts.

Use the commands in [README.md](README.md#development). Maintenance rules are in [AGENTS.md](AGENTS.md).
