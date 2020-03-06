package api

import (
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/vm/contracts/abi"
)

func (api *ContractApi) GetGovernanceState() (map[string]interface{}, error) {
	db, err := getVmDb(api.chain, types.AddressGovernance)
	if err != nil {
		return nil, err
	}

	selector, err := abi.NewGovernanceSelector(db)
	if err != nil {
		return nil, err
	}
	sbpInfos, groupInfos, err := selector.SelectState()
	if err != nil {
		return nil, err
	}
	return map[string]interface{}{
		"SbpInfos":   sbpInfos,
		"GroupInfos": groupInfos,
	}, err
}
