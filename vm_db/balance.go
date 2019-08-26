package vm_db

import (
	"github.com/vitelabs/go-vite/common/db/xleveldb/errors"
	"github.com/vitelabs/go-vite/common/types"
	"math/big"
)

func (vdb *vmDb) GetBalance(tokenTypeId *types.TokenTypeId) (*big.Int, error) {
	if vdb.uns != nil {
		if balance, ok := vdb.unsaved().GetBalance(tokenTypeId); ok {
			return new(big.Int).Set(balance), nil
		}
	}

	return vdb.chain.GetBalance(*vdb.address, *tokenTypeId)
}

func (vdb *vmDb) SetBalance(tokenTypeId *types.TokenTypeId, amount *big.Int) {
	vdb.unsaved().SetBalance(tokenTypeId, amount)
}

func (vdb *vmDb) GetBalanceMap() (map[types.TokenTypeId]*big.Int, error) {
	address := vdb.Address()
	if address == nil {
		return nil, errors.New("no address")
	}

	balanceMap, err := vdb.chain.GetBalanceMap(*address)
	if err != nil {
		return nil, err
	}

	if balanceMap == nil {
		balanceMap = make(map[types.TokenTypeId]*big.Int)
	}
	unsavedBalanceMap := vdb.GetUnsavedBalanceMap()
	for tokenId, balance := range unsavedBalanceMap {
		balanceMap[tokenId] = balance
	}

	return balanceMap, nil

}

func (vdb *vmDb) GetUnsavedBalanceMap() map[types.TokenTypeId]*big.Int {
	if vdb.uns == nil {
		return make(map[types.TokenTypeId]*big.Int)
	}
	return vdb.unsaved().GetBalanceMap()
}
