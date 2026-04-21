# Ethereum Proof Generator/Verifier in Go

This project generates and verifies three Ethereum Merkle Patricia Trie proof types:

- `state`: account proof + single storage slot proof against `stateRoot`
- `receipt(event)`: receipt inclusion proof against `receiptsRoot`, then event matching inside the receipt
- `transaction`: transaction inclusion proof against `transactionsRoot`

It ships with deterministic offline fixtures under [proof/testdata](/Users/sudoless/codespace/coding/eth-proof/proof/testdata) and can optionally generate live proofs from Ethereum mainnet RPCs.

It also includes a local Anvil-backed e2e path that deploys a minimal test contract, generates all three proof types from a real local transaction, and validates them through both the Go API and the CLI.

## Proof model

### State proof

```text
account ⊂ state trie -> stateRoot
slot ⊂ storage trie(account.storageRoot) -> storageRoot ⊂ account
```

Generation uses `eth_getProof` for account/storage proofs, but verification is fully local:

1. verify `accountProof` against `stateRoot` and `keccak(address)`
2. decode the verified account RLP and check `nonce/balance/storageRoot/codeHash`
3. verify `storageProof` against `account.storageRoot` and `keccak(slot)`
4. normalize the storage value and compare it with the claimed slot value

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
- storage proof nodes
- account RLP / decoded account fields
- normalized storage slot value

For `receipt` and `transaction` proofs the normalized data includes:

- block header context
- target transaction bytes
- block transaction list
- block receipt list for receipt proofs
- target receipt bytes and target event fields for receipt proofs

The default minimum is `3` distinct RPC sources. This can be overridden per request with `--min-rpcs`; the local Anvil e2e uses `--min-rpcs 1` explicitly.

## Commands

### Generate state proof

```bash
go run ./cmd/ethproof generate state \
  --rpc https://rpc-1.example \
  --rpc https://rpc-2.example \
  --rpc https://rpc-3.example \
  --min-rpcs 3 \
  --block 22000000 \
  --account 0xYourAccount \
  --slot 0x0000000000000000000000000000000000000000000000000000000000000000 \
  --out state.json
```

### Generate receipt / event proof

```bash
go run ./cmd/ethproof generate receipt \
  --rpc https://rpc-1.example \
  --rpc https://rpc-2.example \
  --rpc https://rpc-3.example \
  --min-rpcs 3 \
  --tx 0xYourTxHash \
  --log-index 0 \
  --out receipt.json
```

### Generate transaction proof

```bash
go run ./cmd/ethproof generate tx \
  --rpc https://rpc-1.example \
  --rpc https://rpc-2.example \
  --rpc https://rpc-3.example \
  --min-rpcs 3 \
  --tx 0xYourTxHash \
  --out tx.json
```

### Verify proofs

```bash
go run ./cmd/ethproof verify state --proof state.json

go run ./cmd/ethproof verify receipt \
  --proof receipt.json \
  --expect-emitter 0xYourContract \
  --expect-topic 0xYourTopic0 \
  --expect-data 0xYourEventData

go run ./cmd/ethproof verify tx --proof tx.json
```

`verify receipt` always validates all fields embedded in the proof package. `--expect-*` flags add extra assertions on top of the package’s own claims.

## Offline fixtures

The repository includes three offline fixtures:

- [transaction_fixture.json](/Users/sudoless/codespace/coding/eth-proof/proof/testdata/transaction_fixture.json)
- [receipt_fixture.json](/Users/sudoless/codespace/coding/eth-proof/proof/testdata/receipt_fixture.json)
- [state_fixture.json](/Users/sudoless/codespace/coding/eth-proof/proof/testdata/state_fixture.json)

These are deterministic synthetic Ethereum examples built with real Ethereum encodings and trie rules, so tests do not depend on network access.

Regenerate them with:

```bash
go run ./cmd/mkfixtures --out-dir ./proof/testdata
```

## Make targets

```bash
make build
make test
make fixtures
make bindings
make e2e-up
make e2e-test
make e2e-down
make e2e
make live-test
```

## Local Anvil e2e

The local e2e flow uses the checked-in [docker-compose.yml](/Users/sudoless/codespace/coding/eth-proof/docker-compose.yml) and a tiny Foundry contract [contracts/ProofDemo.sol](/Users/sudoless/codespace/coding/eth-proof/contracts/ProofDemo.sol):

- `uint256 public value` keeps the state proof target fixed at storage `slot 0`
- `setValue(uint256,bytes32)` updates storage and emits `ValueUpdated(address indexed caller, bytes32 indexed marker, uint256 value)`
- the same transaction drives `transaction proof`, `receipt/event proof`, and `state proof`

Start the node and run e2e:

```bash
make e2e-up
make e2e-test
make e2e-down
```

Or run the convenience target:

```bash
make e2e
```

The e2e test expects Anvil on `http://127.0.0.1:8545` with chain ID `1337`. You can override the RPC URL with `ETH_PROOF_E2E_RPC`.

## Contract bindings

Go contract bindings are generated with geth `abigen v1`, not `--v2`.

Regenerate them with:

```bash
make bindings
```

The target runs:

```bash
forge build
forge inspect --json contracts/ProofDemo.sol:ProofDemo abi > ABI.json
forge inspect contracts/ProofDemo.sol:ProofDemo bytecode > BIN.txt
go tool github.com/ethereum/go-ethereum/cmd/abigen \
  --abi ABI.json \
  --bin BIN.txt \
  --pkg bindings \
  --type ProofDemo \
  --out internal/e2e/bindings/proofdemo.go
```

`make live-test` requires these environment variables:

- `ETH_PROOF_RPCS`: comma-separated list of at least 3 archive-capable RPC URLs
- `ETH_PROOF_LIVE_TX`: transaction hash used for `transaction` and `receipt` proofs
- `ETH_PROOF_LIVE_LOG_INDEX`: log index inside that receipt
- `ETH_PROOF_LIVE_STATE_BLOCK`: block number used for the state proof
- `ETH_PROOF_LIVE_ACCOUNT`: account address for the state proof
- `ETH_PROOF_LIVE_SLOT`: 32-byte storage slot key for the state proof

Example:

```bash
ETH_PROOF_RPCS="https://rpc1,https://rpc2,https://rpc3" \
ETH_PROOF_LIVE_TX=0x... \
ETH_PROOF_LIVE_LOG_INDEX=0 \
ETH_PROOF_LIVE_STATE_BLOCK=22000000 \
ETH_PROOF_LIVE_ACCOUNT=0x... \
ETH_PROOF_LIVE_SLOT=0x... \
make live-test
```

## Notes

- `state` proofs use `eth_getProof`; `receipt` and `transaction` proofs are rebuilt locally from canonical block data.
- Verification only checks proof correctness against the included roots. If you need bridge-grade security, you must separately verify that the block header itself is finalized and trusted.
- The code targets `github.com/ethereum/go-ethereum` `v1.17.x`.
