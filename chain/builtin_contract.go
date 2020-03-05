package chain

import (
	"fmt"

	"github.com/vitelabs/go-vite/ledger"

	"github.com/vitelabs/go-vite/common/helper"

	"github.com/pkg/errors"
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/vm/contracts/abi"
)

// sb height
func (c *chain) GetRegisterList(snapshotHash types.Hash, gid types.Gid) ([]*types.Registration, error) {
	sd, err := c.stateDB.NewStorageDatabase(snapshotHash, types.AddressGovernance)
	if err != nil {
		cErr := errors.New(fmt.Sprintf("c.stateDB.NewStorageDatabase failed, snapshotHash is %s",
			snapshotHash))
		c.log.Error(cErr.Error(), "method", "GetRegisterList")
		return nil, cErr
	}

	// do something
	return abi.GetCandidateList(sd, gid)
}

func (c *chain) GetAllRegisterList(snapshotHash types.Hash, gid types.Gid) ([]*types.Registration, error) {
	sd, err := c.stateDB.NewStorageDatabase(snapshotHash, types.AddressGovernance)
	if err != nil {
		cErr := errors.New(fmt.Sprintf("c.stateDB.NewStorageDatabase failed, snapshotHash is %s",
			snapshotHash))
		c.log.Error(cErr.Error(), "method", "GetAllRegisterList")
		return nil, cErr
	}

	// do something
	return abi.GetAllRegistrationList(sd, gid)
}

func (c *chain) GetConsensusGroup(snapshotHash types.Hash, gid types.Gid) (*types.ConsensusGroupInfo, error) {
	sd, err := c.stateDB.NewStorageDatabase(snapshotHash, types.AddressGovernance)
	if err != nil {
		cErr := errors.New(fmt.Sprintf("c.stateDB.NewStorageDatabase failed, snapshotHash is %s",
			snapshotHash))
		c.log.Error(cErr.Error(), "method", "GetConsensusGroup")
		return nil, cErr
	}

	return abi.GetConsensusGroup(sd, gid)
}

func (c *chain) GetConsensusGroupList(snapshotHash types.Hash) ([]*types.ConsensusGroupInfo, error) {
	sd, err := c.stateDB.NewStorageDatabase(snapshotHash, types.AddressGovernance)
	if err != nil {
		cErr := errors.New(fmt.Sprintf("c.stateDB.NewStorageDatabase failed, snapshotHash is %s",
			snapshotHash))
		c.log.Error(cErr.Error(), "method", "GetConsensusGroupList")
		return nil, cErr
	}

	// do something
	return abi.GetConsensusGroupList(sd)
}

// total
func (c *chain) GetCurrentStakeQuota(addr types.Address) (uint64, error) {
	return helper.MaxUint64, nil
}

func (c *chain) GetCurrentStakeQuotas(addrList []types.Address) (map[types.Address]uint64, error) {
	result := make(map[types.Address]uint64, len(addrList))
	for _, v := range addrList {
		result[v] = helper.MaxUint64
	}
	return result, nil
}

func (c *chain) GetTokenInfoById(tokenId types.TokenTypeId) (*types.TokenInfo, error) {
	if tokenId.Hex() == ledger.ViteTokenId.Hex() {
		return &types.TokenInfo{
			TokenName:     "Vite ",
			TokenSymbol:   "VITE",
			TotalSupply:   nil,
			Decimals:      18,
			Owner:         types.Address{},
			MaxSupply:     nil,
			OwnerBurnOnly: false,
			IsReIssuable:  false,
			Index:         0,
		}, nil
	}
	panic("GetTokenInfoById error")
}
