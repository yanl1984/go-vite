package quota

import (
	"math/big"

	"github.com/vitelabs/go-vite/common/helper"

	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/ledger"
)

type quotaConfigParams struct {
	difficultyList  []*big.Int
	stakeAmountList []*big.Int
	qcIndexMin      uint64
	qcIndexMax      uint64
	qcMap           map[uint64]*big.Int
	calcQuotaFunc   func(db quotaDb, addr types.Address, stakeAmount *big.Int, difficulty *big.Int, sbHeight uint64) (quotaTotal, quotaStake, quotaAddition, snapshotCurrentQuota, quotaAvg uint64, blocked bool, blockReleaseHeight uint64, err error)
}

var quotaConfig quotaConfigParams

type quotaDb interface {
	GetGlobalQuota() types.QuotaInfo
	GetQuotaUsedList(address types.Address) []types.QuotaInfo
	GetUnconfirmedBlocks(address types.Address) []*ledger.AccountBlock
	GetLatestAccountBlock(addr types.Address) (*ledger.AccountBlock, error)
	GetConfirmedTimes(blockHash types.Hash) (uint64, error)
}

// CalcBlockQuotaUsed recalculate quotaUsed field of an account block
func CalcBlockQuotaUsed(db quotaDb, block *ledger.AccountBlock, sbHeight uint64) (uint64, error) {
	return block.Quota, nil // todo
	panic("CalcBlockQuotaUsed error")
}

// GetQuotaForBlock calculate available quota for a block
func GetQuotaForBlock(db quotaDb, addr types.Address, sbHeight uint64) (quotaTotal uint64, err error) {
	return helper.MaxUint64, nil // todo
	panic("GetQuotaForBlock error")
}

// CheckQuota check whether current quota of a contract account is enough to receive a new block
func CheckQuota(db quotaDb, q uint64, addr types.Address) (bool, uint64) {
	return true, 0
	panic("CheckQuota error")
}
