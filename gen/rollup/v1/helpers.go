package rollupv1

import (
	"errors"

	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	sdktypes "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/msgservice"
)

var _ sdktypes.Msg = (*MsgApplyL1TxsRequest)(nil)

func (*MsgApplyL1TxsRequest) GetSigners() []sdktypes.AccAddress {
	return nil
}

func (m *MsgApplyL1TxsRequest) ValidateBasic() error {
	if len(m.TxBytes) < 1 {
		return errors.New("expected TxBytes to contain at least one deposit transaction")
	}
	return nil
}

func (*MsgApplyL1TxsRequest) Type() string {
	return "l1txs"
}

func (*MsgApplyL1TxsRequest) Route() string {
	return "rollup"
}

func RegisterInterfaces(registry codectypes.InterfaceRegistry) {
	msgservice.RegisterMsgServiceDesc(registry, &_MsgService_serviceDesc)
}

// TODO: add MsgWithdraw helpers
