package dex

import (
	"math/big"

	"github.com/vitelabs/go-vite"
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/interfaces"
	"github.com/vitelabs/go-vite/vm/contracts/dex"
	"github.com/vitelabs/go-vite/vm/util"
)

func GetConsensusReader(vite *vite.Vite) *util.VMConsensusReader {
	return util.NewVMConsensusReader(vite.Consensus().SBPReader())
}

func AmountBytesToString(amt []byte) string {
	return new(big.Int).SetBytes(amt).String()
}

func TokenBytesToString(token []byte) string {
	tk, _ := types.BytesToTokenTypeId(token)
	return tk.String()
}

func InnerGetOrderById(db interfaces.VmDb, orderId []byte) (*RpcOrder, error) {
	matcher := dex.NewRawMatcher(db)
	if order, err := matcher.GetOrderById(orderId); err != nil {
		return nil, err
	} else {
		return OrderToRpc(order), nil
	}
}
