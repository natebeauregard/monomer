package keeper

import (
	"context"
	"github.com/polymerdao/monomer/gen/rollup/v1"
)

var _ rollupv1.QueryServiceServer = (*Keeper)(nil)

// L1BlockInfo implements the Query/L1BlockInfo gRPC method
// It returns the L1 block info. L2 clients should directly get L1 block info from x/rollup keeper
func (k *Keeper) L1BlockInfo(ctx context.Context, request *rollupv1.QueryL1BlockInfoRequest) (*rollupv1.QueryL1BlockInfoResponse, error) {
	// TODO: make a ticket to retrieve L1BlockInfo at the requested height instead of always using the latest one
	info, err := k.GetL1BlockInfo(ctx)
	if err != nil {
		return nil, err
	}
	return &rollupv1.QueryL1BlockInfoResponse{
		Number:         info.Number,
		Time:           info.Time,
		BaseFee:        info.BaseFee.Bytes(),
		BlockHash:      info.BlockHash.Bytes(),
		SequenceNumber: info.SequenceNumber,
		BatcherAddr:    info.BatcherAddr.Bytes(),
		L1FeeOverhead:  info.L1FeeOverhead[:],
		L1FeeScalar:    info.L1FeeScalar[:],
	}, nil
}

// OutputAtBlock implements the Query/OutputAtBlock gRPC method
// Retrieves the L2 output at a given block height to prove withdrawal transactions on L1.
func (k *Keeper) OutputAtBlock(goCtx context.Context, request *rollupv1.QueryOutputAtBlockRequest) (*rollupv1.QueryOutputAtBlockResponse, error) {
	//ctx := sdk.UnwrapSDKContext(goCtx)

	// TODO: implement after the L2ToL1MessagePasser is implemented in x/rollup

	return &rollupv1.QueryOutputAtBlockResponse{
		// TODO: fill in the response fields
		Version:               make([]byte, 32),
		OutputRoot:            nil,
		BlockRef:              nil,
		WithdrawalStorageRoot: nil,
		StateRoot:             nil,
		Status:                nil,
	}, nil
}
