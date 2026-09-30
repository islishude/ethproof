package proof

import (
	"bytes"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/trie"
	"github.com/islishude/ethproof/internal/proofutil"
)

func TestIndexTrieBuildersValidateEncodingAndRoot(t *testing.T) {
	source, _, _ := mustReceiptSource(t)
	txs := make([]hexutil.Bytes, len(source.block.Transactions()))
	receipts := make([]hexutil.Bytes, len(source.blockReceipts))
	for i, tx := range source.block.Transactions() {
		var err error
		txs[i], err = proofutil.EncodeTransaction(tx)
		if err != nil {
			t.Fatal(err)
		}
	}
	for i, receipt := range source.blockReceipts {
		var err error
		receipts[i], err = proofutil.EncodeReceipt(receipt)
		if err != nil {
			t.Fatal(err)
		}
	}
	for _, tt := range []struct {
		name    string
		entries []hexutil.Bytes
		root    common.Hash
		build   func([]hexutil.Bytes, uint64, common.Hash) (hexutil.Bytes, []hexutil.Bytes, error)
	}{
		{"transaction", txs, source.block.TxHash(), buildTransactionTrieAndProof},
		{"receipt", receipts, source.block.ReceiptHash(), buildReceiptTrieAndProof},
	} {
		t.Run(tt.name, func(t *testing.T) {
			value, nodes, err := tt.build(tt.entries, 1, tt.root)
			if err != nil {
				t.Fatal(err)
			}
			db, err := proofutil.ProofDBFromHexNodes(nodes)
			if err != nil {
				t.Fatal(err)
			}
			verified, err := trie.VerifyProof(tt.root, proofutil.TrieIndexKey(1), db)
			if err != nil || !bytes.Equal(verified, value) || !bytes.Equal(value, tt.entries[1]) {
				t.Fatalf("invalid generated proof: %v", err)
			}
			if _, _, err := tt.build(tt.entries, 1, common.Hash{}); err == nil {
				t.Fatal("accepted wrong root")
			}
			if _, _, err := tt.build(tt.entries, uint64(len(tt.entries)), tt.root); err == nil {
				t.Fatal("accepted out-of-range index")
			}
			// A malformed non-target entry must be rejected even when its raw trie root matches.
			invalid := cloneHexBytesList(tt.entries)
			invalid[0] = hexutil.Bytes{0xff}
			rawTrie := proofutil.MakeProofTrie()
			for i, entry := range invalid {
				if err := rawTrie.Update(proofutil.TrieIndexKey(uint64(i)), entry); err != nil {
					t.Fatal(err)
				}
			}
			if _, _, err := tt.build(invalid, 1, rawTrie.Hash()); err == nil {
				t.Fatal("accepted invalid non-target encoding")
			}
		})
	}
}
