package proof

import "context"

// GenerateStateProofFromRPC opens one RPC connection for the duration of generation.
func GenerateStateProofFromRPC(ctx context.Context, rpcURL string, req StateProofRequest) (*StateProofPackage, error) {
	source, err := openRPCSource(ctx, rpcURL)
	if err != nil {
		return nil, err
	}
	defer source.Close()
	return GenerateStateProof(ctx, source, req)
}

// GenerateReceiptProofFromRPC opens one RPC connection for the duration of generation.
func GenerateReceiptProofFromRPC(ctx context.Context, rpcURL string, req ReceiptProofRequest) (*ReceiptProofPackage, error) {
	source, err := openRPCSource(ctx, rpcURL)
	if err != nil {
		return nil, err
	}
	defer source.Close()
	return GenerateReceiptProof(ctx, source, req)
}

// GenerateTransactionProofFromRPC opens one RPC connection for the duration of generation.
func GenerateTransactionProofFromRPC(ctx context.Context, rpcURL string, req TransactionProofRequest) (*TransactionProofPackage, error) {
	source, err := openRPCSource(ctx, rpcURL)
	if err != nil {
		return nil, err
	}
	defer source.Close()
	return GenerateTransactionProof(ctx, source, req)
}
