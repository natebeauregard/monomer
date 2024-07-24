package types

const (
	AttributeKeyL1DepositTxType   = "l1_deposit_tx_type"
	AttributeKeyL2WithdrawalType  = "l2_withdrawal_type"
	AttributeKeyBridgedTokenType  = "bridged_token_type"
	AttributeKeyFromEvmAddress    = "from_evm_address"
	AttributeKeyToEvmAddress      = "to_evm_address"
	AttributeKeyFromCosmosAddress = "from_cosmos_address"
	AttributeKeyToCosmosAddress   = "to_cosmos_address"
	AttributeKeyAmount            = "amount"
	AttributeKeySender            = "sender"
	AttributeKeyTarget            = "target"

	L1SystemDepositTxType = "l1_system_deposit"
	L1UserDepositTxType   = "l1_user_deposit"
	L2WithdrawalType      = "l2_withdrawal"

	EventTypeMintETH             = "mint_eth"
	EventTypeBurnETH             = "burn_eth"
	EventTypeWithdrawalInitiated = "withdrawal_initiated"
)
