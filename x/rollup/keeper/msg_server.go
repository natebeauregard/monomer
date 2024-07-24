package keeper

import (
	"context"
	"encoding/json"
	"fmt"

	sdkmath "cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/ethereum-optimism/optimism/op-node/rollup/derive"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	ethtypes "github.com/ethereum/go-ethereum/core/types"
	"github.com/polymerdao/monomer/gen/rollup/v1"
	"github.com/polymerdao/monomer/x/rollup/types"
	"github.com/samber/lo"
)

type msgServer struct {
	*Keeper
}

// NewMsgServerImpl returns an implementation of the MsgServer interface
// for the provided Keeper.
func NewMsgServerImpl(keeper *Keeper) rollupv1.MsgServiceServer {
	return &msgServer{Keeper: keeper}
}

var _ rollupv1.MsgServiceServer = msgServer{}

// TODO: create task to move full message/query implementations out of msg_server.go into separate files.

// ApplyL1Txs executes all L1 system and user deposit txs passed through MsgApplyL1TxsRequest.
func (k *Keeper) ApplyL1Txs(goCtx context.Context, msg *rollupv1.MsgApplyL1TxsRequest) (*rollupv1.MsgApplyL1TxsResponse, error) {
	if msg.TxBytes == nil || len(msg.TxBytes) < 1 {
		return nil, types.WrapError(types.ErrInvalidL1Txs, "must have at least one L1 Info Deposit tx")
	}
	ctx := sdk.UnwrapSDKContext(goCtx)

	ctx.Logger().Debug("processing L1 txs", "txCount", len(msg.TxBytes))

	// process L1 system deposit tx
	txBytes := msg.TxBytes[0]
	var tx ethtypes.Transaction
	if err := tx.UnmarshalBinary(txBytes); err != nil {
		ctx.Logger().Error("failed to unmarshal system deposit transaction", "index", 0, "err", err, "txBytes", txBytes)
		return nil, types.WrapError(types.ErrInvalidL1Txs, "failed to unmarshal system deposit transaction: %v", err)
	}
	if !tx.IsDepositTx() {
		ctx.Logger().Error("first L1 tx must be a system deposit tx", "type", tx.Type())
		return nil, types.WrapError(types.ErrInvalidL1Txs, "first L1 tx must be a system deposit tx, but got type %d", tx.Type())
	}
	l1blockInfo, err := derive.L1BlockInfoFromBytes(k.rollupCfg, 0, tx.Data())
	if err != nil {
		ctx.Logger().Error("failed to derive L1 block info from L1 Info Deposit tx", "err", err, "txBytes", txBytes)
		return nil, types.WrapError(types.ErrInvalidL1Txs, "failed to derive L1 block info from L1 Info Deposit tx: %v", err)
	}

	// save L1 block info to AppState
	if err := k.SetL1BlockInfo(&ctx, *l1blockInfo); err != nil {
		ctx.Logger().Error("failed to save L1 block info to AppState", "err", err)
		return nil, types.WrapError(types.ErrL1BlockInfo, "save error: %v", err)
	}

	ctx.Logger().Info("save L1 block info", "l1blockInfo", string(lo.Must(json.Marshal(l1blockInfo))))

	// save L1 block History to AppState
	if err := k.SetL1BlockHistory(&ctx, l1blockInfo); err != nil {
		ctx.Logger().Error("failed to save L1 block history info to AppState", "err", err)
		return nil, types.WrapError(types.ErrL1BlockInfo, "save error: %v", err)
	}

	ctx.Logger().Info("save L1 block history info", "l1blockHistoryInfo", string(lo.Must(json.Marshal(l1blockInfo))))

	// process L1 user deposit txs
	for i := 1; i < len(msg.TxBytes); i++ {
		txBytes := msg.TxBytes[i]
		var tx ethtypes.Transaction
		if err := tx.UnmarshalBinary(txBytes); err != nil {
			ctx.Logger().Error("failed to unmarshal user deposit transaction", "index", i, "err", err, "txBytes", txBytes)
			return nil, types.WrapError(types.ErrInvalidL1Txs, "failed to unmarshal user deposit transaction", "index", i, "err", err)
		}
		if !tx.IsDepositTx() {
			ctx.Logger().Error("L1 tx must be a user deposit tx", "index", i, "type", tx.Type())
			return nil, types.WrapError(types.ErrInvalidL1Txs, "L1 tx must be a user deposit tx, index:%d, type:%d", i, tx.Type())
		}
		if tx.IsSystemTx() {
			ctx.Logger().Error("L1 tx must be a user deposit tx", "type", tx.Type())
			return nil, types.WrapError(types.ErrInvalidL1Txs, "L1 tx must be a user deposit tx, type %d", tx.Type())
		}
		ctx.Logger().Debug("user deposit tx", "index", i, "tx", string(lo.Must(tx.MarshalJSON())))
		to := tx.To()
		// if the receipient is nil, it means the tx is creating a contract which we don't support, so return an error.
		// see https://github.com/ethereum-optimism/op-geth/blob/v1.101301.0-rc.2/core/state_processor.go#L154
		if to == nil {
			ctx.Logger().Error("Contract creation txs are not supported", "index", i)
			return nil, types.WrapError(types.ErrInvalidL1Txs, "Contract creation txs are not supported, index:%d", i)
		}
		cosmAddr := evmToCosmos(*to)
		mintAmount := sdkmath.NewIntFromBigInt(tx.Value())
		err := k.MintETH(&ctx, cosmAddr, mintAmount)
		if err != nil {
			ctx.Logger().Error("failed to mint ETH", "evmAddress", to, "polymerAddress", cosmAddr, "err", err)
			return nil, types.WrapError(types.ErrMintETH, "failed to mint ETH", "polymerAddress", cosmAddr, "err", err)
		}
	}
	return &rollupv1.MsgApplyL1TxsResponse{}, nil
}

// InitiateWithdrawal initiates a withdrawal of L2 assets to L1 and burns the L2 assets.
func (k *Keeper) InitiateWithdrawal(goCtx context.Context, msg *rollupv1.MsgInitiateWithdrawalRequest) (*rollupv1.MsgInitiateWithdrawalResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)
	ctx.Logger().Debug("withdrawing L2 assets", "sender", msg.Sender, "amount", msg.Amount)

	// TODO: add custom error types to errors.go
	accAddress, err := sdk.AccAddressFromBech32(msg.Sender)
	if err != nil {
		ctx.Logger().Error("invalid sender address", "sender", msg.Sender)
		return nil, types.WrapError(err, "invalid sender address", "sender", msg.Sender)
	}

	if err := k.BurnETH(&ctx, accAddress, msg.Amount); err != nil {
		return nil, types.WrapError(types.ErrBurnETH, "failed to burn ETH", "polymerAddress", accAddress, "err", err)
	}

	// Emit an event for relayers to use to build proofs of the withdrawal on L1
	ctx.EventManager().EmitEvents(sdk.Events{
		sdk.NewEvent(
			sdk.EventTypeMessage,
			sdk.NewAttribute(sdk.AttributeKeyModule, types.ModuleName),
		),
		sdk.NewEvent(
			types.EventTypeWithdrawalInitiated,
			// TODO: make sure to add extra info needed for the proof (nonce, data, etc.)
			sdk.NewAttribute(types.AttributeKeySender, msg.Sender),
			sdk.NewAttribute(types.AttributeKeyTarget, msg.Target),
			sdk.NewAttribute(types.AttributeKeyAmount, hexutil.Encode(msg.Amount.BigInt().Bytes())),
		),
	})

	store := k.storeService.OpenKVStore(ctx)
	withdrawaldb, err := store.Get([]byte(types.KeyWithdrawalDB))
	// TODO: store withdrawal messages in a db emulating the L2ToL1MessagePasser contract storage
	if err := store.Set([]byte(types.KeyWithdrawalDB), withdrawaldb); err != nil {
		return nil, types.WrapError(err, "add withdrawal message to db")
	}

	return &rollupv1.MsgInitiateWithdrawalResponse{}, nil
}

// MintETH mints ETH to an account where the amount is in wei, the smallest unit of ETH
func (k *Keeper) MintETH(ctx *sdk.Context, addr sdk.AccAddress, amount sdkmath.Int) error {
	coin := sdk.NewCoin(types.ETH, amount)
	if err := k.mintKeeper.MintCoins(*ctx, sdk.NewCoins(coin)); err != nil {
		return err
	}
	if err := k.bankkeeper.SendCoinsFromModuleToAccount(*ctx, types.MintModule, addr, sdk.NewCoins(coin)); err != nil {
		return err
	}
	ctx.EventManager().EmitEvents(sdk.Events{
		sdk.NewEvent(
			// TODO: we probably don't want the event type to be "message" for this event since it's a call from WithdrawalInitiate
			// Look into why the MintETH event is emitted as a "message" event
			sdk.EventTypeMessage,
			sdk.NewAttribute(sdk.AttributeKeyModule, types.ModuleName),
		),
		sdk.NewEvent(
			types.EventTypeMintETH,
			sdk.NewAttribute(types.AttributeKeyL1DepositTxType, types.L1UserDepositTxType),
			sdk.NewAttribute(types.AttributeKeyToCosmosAddress, addr.String()),
			sdk.NewAttribute(types.AttributeKeyAmount, hexutil.Encode(amount.BigInt().Bytes())),
		),
	})
	return nil
}

// BurnETH burns ETH from an account where the amount is in wei
func (k *Keeper) BurnETH(ctx *sdk.Context, addr sdk.AccAddress, amount sdkmath.Int) error {
	coins := sdk.NewCoins(sdk.NewCoin(types.ETH, amount))

	// Transfer the coins to withdraw from the user account to the mint module
	err := k.bankkeeper.SendCoinsFromAccountToModule(*ctx, addr, types.MintModule, coins)
	if err != nil {
		return fmt.Errorf("failed to send withdrawal coins from user account to mint module: %v", err)
	}

	// Burn the ETH coins from the mint module
	if err := k.bankkeeper.BurnCoins(*ctx, types.MintModule, coins); err != nil {
		return err
	}

	// TODO: mirrored these event attributes from MintETH. ensure that these are the attributes we want to emit here
	ctx.EventManager().EmitEvents(sdk.Events{
		sdk.NewEvent(
			sdk.EventTypeMessage,
			sdk.NewAttribute(sdk.AttributeKeyModule, types.ModuleName),
		),
		sdk.NewEvent(
			types.EventTypeBurnETH,
			sdk.NewAttribute(types.AttributeKeyL2WithdrawalType, types.EventTypeWithdrawalInitiated),
			sdk.NewAttribute(types.AttributeKeyFromCosmosAddress, addr.String()),
			sdk.NewAttribute(types.AttributeKeyAmount, hexutil.Encode(amount.BigInt().Bytes())),
		),
	})
	return nil
}

// SetL1BlockInfo sets the L1 block info to the app state
//
// Persisted data conforms to optimism specs on L1 attributes:
// https://github.com/ethereum-optimism/optimism/blob/develop/specs/deposits.md#l1-attributes-predeployed-contract
func (k *Keeper) SetL1BlockInfo(ctx *sdk.Context, info derive.L1BlockInfo) error { //nolint:gocritic
	infoBytes, err := json.Marshal(info)
	if err != nil {
		return types.WrapError(err, "marshal L1 block info")
	}
	if err := k.storeService.OpenKVStore(ctx).Set([]byte(types.KeyL1BlockInfo), infoBytes); err != nil {
		return types.WrapError(err, "set")
	}
	return nil
}

// GetL1BlockInfo gets the L1 block info from the app state
func (k *Keeper) GetL1BlockInfo(ctx context.Context) (*derive.L1BlockInfo, error) {
	// TODO get/set as protobuf
	infoBytes, err := k.storeService.OpenKVStore(ctx).Get([]byte(types.KeyL1BlockInfo))
	if err != nil {
		return nil, types.WrapError(err, "get")
	}
	if infoBytes == nil {
		return nil, types.WrapError(types.ErrL1BlockInfo, "not found")
	}
	info := new(derive.L1BlockInfo)
	if err := json.Unmarshal(infoBytes, &info); err != nil {
		return nil, types.WrapError(err, "unmarshal L1 block info")
	}
	return info, nil
}

// SetL1BlockHistory sets the L1 block info to the app state, with the key being the blockhash, so we can look it up easily later.
func (k *Keeper) SetL1BlockHistory(ctx context.Context, info *derive.L1BlockInfo) error {
	infoBytes, err := json.Marshal(info)
	if err != nil {
		return types.WrapError(err, "marshal L1 block info")
	}
	if err := k.storeService.OpenKVStore(ctx).Set(info.BlockHash.Bytes(), infoBytes); err != nil {
		return types.WrapError(err, "set L1 block history")
	}
	return nil
}

// evmToCosmos converts an EVM address to a sdk.AccAddress
func evmToCosmos(addr common.Address) sdk.AccAddress {
	return addr.Bytes()
}
