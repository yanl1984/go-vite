package chain

import (
	"fmt"
	"github.com/vitelabs/go-vite/interfaces"
	"github.com/vitelabs/go-vite/monitor"
	"sync/atomic"
	"time"
)

const (
	InsertAccountBlockFunc                 = 0
	InsertSnapshotBlockFunc                = 1
	DeleteAccountBlocksFunc                = 2
	DeleteAccountBlocksToHeightFunc        = 3
	DeleteSnapshotBlocksFunc               = 4
	DeleteSnapshotBlocksToHeightFunc       = 5
	IsGenesisAccountBlockFunc              = 6
	IsAccountBlockExistedFunc              = 7
	GetAccountBlockByHeightFunc            = 8
	GetAccountBlockByHashFunc              = 9
	GetReceiveAbBySendAbFunc               = 10
	IsReceivedFunc                         = 11
	GetAccountBlocksFunc                   = 12
	GetCompleteBlockByHashFunc             = 13
	GetAccountBlocksByHeightFunc           = 14
	GetCallDepthFunc                       = 15
	GetConfirmedTimesFunc                  = 16
	GetLatestAccountBlockFunc              = 17
	GetLatestAccountHeightFunc             = 18
	IsGenesisSnapshotBlockFunc             = 19
	IsSnapshotBlockExistedFunc             = 20
	GetGenesisSnapshotBlockFunc            = 21
	GetLatestSnapshotBlockFunc             = 22
	GetSnapshotHeightByHashFunc            = 23
	GetSnapshotHeaderByHeightFunc          = 24
	GetSnapshotBlockByHeightFunc           = 25
	GetSnapshotHeaderByHashFunc            = 26
	GetSnapshotBlockByHashFunc             = 27
	GetRangeSnapshotHeadersFunc            = 28
	GetRangeSnapshotBlocksFunc             = 29
	GetSnapshotHeadersFunc                 = 30
	GetSnapshotBlocksFunc                  = 31
	GetSnapshotHeadersByHeightFunc         = 32
	GetSnapshotBlocksByHeightFunc          = 33
	GetConfirmSnapshotHeaderByAbHashFunc   = 34
	GetConfirmSnapshotBlockByAbHashFunc    = 35
	GetSnapshotHeaderBeforeTimeFunc        = 36
	GetSnapshotHeadersAfterOrEqualTimeFunc = 37
	GetLastSeedSnapshotHeaderFunc          = 38
	GetRandomSeedFunc                      = 39
	GetSnapshotBlockByContractMetaFunc     = 40
	GetSeedFunc                            = 41
	GetSubLedgerFunc                       = 42
	GetSubLedgerAfterHeightFunc            = 43
	GetAllUnconfirmedBlocksFunc            = 44
	GetUnconfirmedBlocksFunc               = 45
	GetContentNeedSnapshotFunc             = 46
	IsContractAccountFunc                  = 47
	IterateContractsFunc                   = 48
	IterateAccountsFunc                    = 49
	GetBalanceFunc                         = 50
	GetBalanceMapFunc                      = 51
	GetConfirmedBalanceListFunc            = 52
	GetContractCodeFunc                    = 53
	GetContractMetaFunc                    = 54
	GetContractMetaInSnapshotFunc          = 55
	GetContractListFunc                    = 56
	GetQuotaUnusedFunc                     = 57
	GetGlobalQuotaFunc                     = 58
	GetQuotaUsedListFunc                   = 59
	GetStorageIteratorFunc                 = 60
	GetValueFunc                           = 61
	GetVmLogListFunc                       = 62
	GetRegisterListFunc                    = 63
	GetAllRegisterListFunc                 = 64
	GetConsensusGroupListFunc              = 65
	GetVoteListFunc                        = 66
	GetPledgeBeneficialAmountFunc          = 67
	GetPledgeQuotaFunc                     = 68
	GetPledgeQuotasFunc                    = 69
	GetTokenInfoByIdFunc                   = 70
	GetAllTokenInfoFunc                    = 71
	GetLedgerReaderByHeightFunc            = 72
	GetSyncCacheFunc                       = 73
	LoadOnRoadFunc                         = 74
	DeleteOnRoadFunc                       = 75
	GetAccountOnRoadInfoFunc               = 76
	GetOnRoadBlocksByAddrFunc              = 77
	LoadAllOnRoadFunc                      = 78
	MaxFunc                                = 79
)

type Statistic struct {
	startTimeUnixNano int64
	list              []*interfaces.StatisticInfo
}

func NewStatistic() *Statistic {
	s := &Statistic{
		startTimeUnixNano: time.Now().UnixNano(),
		list: []*interfaces.StatisticInfo{{
			Name: "InsertAccountBlock",
		}, {
			Name: "InsertSnapshotBlock",
		}, {
			Name: "DeleteAccountBlocksFunc",
		}, {
			Name: "DeleteAccountBlocksToHeightFunc",
		}, {
			Name: "DeleteSnapshotBlocksFunc",
		}, {
			Name: "DeleteSnapshotBlocksToHeightFunc",
		}, {
			Name: "IsGenesisAccountBlockFunc",
		}, {
			Name: "IsAccountBlockExistedFunc",
		}, {
			Name: "GetAccountBlockByHeightFunc",
		}, {
			Name: "GetAccountBlockByHashFunc",
		}, {
			Name: "GetReceiveAbBySendAbFunc",
		}, {
			Name: "IsReceivedFunc",
		}, {
			Name: "GetAccountBlocksFunc",
		}, {
			Name: "GetCompleteBlockByHashFunc",
		}, {
			Name: "GetAccountBlocksByHeightFunc",
		}, {
			Name: "GetCallDepthFunc",
		}, {
			Name: "GetConfirmedTimesFunc",
		}, {
			Name: "GetLatestAccountBlockFunc",
		}, {
			Name: "GetLatestAccountHeightFunc",
		}, {
			Name: "IsGenesisSnapshotBlockFunc",
		}, {
			Name: "IsSnapshotBlockExistedFunc",
		}, {
			Name: "GetGenesisSnapshotBlockFunc",
		}, {
			Name: "GetLatestSnapshotBlockFunc",
		}, {
			Name: "GetSnapshotHeightByHashFunc",
		}, {
			Name: "GetSnapshotHeaderByHeightFunc",
		}, {
			Name: "GetSnapshotBlockByHeightFunc",
		}, {
			Name: "GetSnapshotHeaderByHashFunc",
		}, {
			Name: "GetSnapshotBlockByHashFunc",
		}, {
			Name: "GetRangeSnapshotHeadersFunc",
		}, {
			Name: "GetRangeSnapshotBlocksFunc",
		}, {
			Name: "GetSnapshotHeadersFunc",
		}, {
			Name: "GetSnapshotBlocksFunc",
		}, {
			Name: "GetSnapshotHeadersByHeightFunc",
		}, {
			Name: "GetSnapshotBlocksByHeightFunc",
		}, {
			Name: "GetConfirmSnapshotHeaderByAbHashFunc",
		}, {
			Name: "GetConfirmSnapshotBlockByAbHashFunc",
		}, {
			Name: "GetSnapshotHeaderBeforeTimeFunc",
		}, {
			Name: "GetSnapshotHeadersAfterOrEqualTimeFunc",
		}, {
			Name: "GetLastSeedSnapshotHeaderFunc",
		}, {
			Name: "GetRandomSeedFunc",
		}, {
			Name: "GetSnapshotBlockByContractMetaFunc",
		}, {
			Name: "GetSeedFunc",
		}, {
			Name: "GetSubLedgerFunc",
		}, {
			Name: "GetSubLedgerAfterHeightFunc",
		}, {
			Name: "GetAllUnconfirmedBlocksFunc",
		}, {
			Name: "GetUnconfirmedBlocksFunc",
		}, {
			Name: "GetContentNeedSnapshotFunc",
		}, {
			Name: "IsContractAccountFunc",
		}, {
			Name: "IterateContractsFunc",
		}, {
			Name: "IterateAccountsFunc",
		}, {
			Name: "GetBalanceFunc",
		}, {
			Name: "GetBalanceMapFunc",
		}, {
			Name: "GetConfirmedBalanceListFunc",
		}, {
			Name: "GetContractCodeFunc",
		}, {
			Name: "GetContractMetaFunc",
		}, {
			Name: "GetContractMetaInSnapshotFunc",
		}, {
			Name: "GetContractListFunc",
		}, {
			Name: "GetQuotaUnusedFunc",
		}, {
			Name: "GetGlobalQuotaFunc",
		}, {
			Name: "GetQuotaUsedListFunc",
		}, {
			Name: "GetStorageIteratorFunc",
		}, {
			Name: "GetValueFunc",
		}, {
			Name: "GetVmLogListFunc",
		}, {
			Name: "GetRegisterListFunc",
		}, {
			Name: "GetAllRegisterListFunc",
		}, {
			Name: "GetConsensusGroupListFunc",
		}, {
			Name: "GetVoteListFunc",
		}, {
			Name: "GetPledgeBeneficialAmountFunc",
		}, {
			Name: "GetPledgeQuotaFunc",
		}, {
			Name: "GetPledgeQuotasFunc",
		}, {
			Name: "GetTokenInfoByIdFunc",
		}, {
			Name: "GetAllTokenInfoFunc",
		}, {
			Name: "GetLedgerReaderByHeightFunc",
		}, {
			Name: "GetSyncCacheFunc",
		}, {
			Name: "LoadOnRoadFunc",
		}, {
			Name: "DeleteOnRoadFunc",
		}, {
			Name: "GetAccountOnRoadInfoFunc",
		}, {
			Name: "GetOnRoadBlocksByAddrFunc",
		}, {
			Name: "LoadAllOnRoadFunc",
		}},
	}

	if len(s.list) != MaxFunc {
		panic(fmt.Sprintf("len(s.list) is %d, MaxFunc is %d", len(s.list), MaxFunc))
	}

	return s
}

func (s *Statistic) Add(flag int) {
	atomic.AddUint64(&s.list[flag].Count, 1)
	// duration is nanoseconds
	monitor.LogDuration("chain", s.list[flag].Name, 1000)
	//monitor.LogTimerConsuming([]string{"chain", s.list[flag].Name}, s)
}

func (s *Statistic) GetSummary() []*interfaces.StatisticInfo {
	now := time.Now()
	diff := now.UnixNano() - s.startTimeUnixNano
	total := make([]*interfaces.StatisticInfo, 3)
	for index := range total {
		total[index] = &interfaces.StatisticInfo{}
	}
	for _, item := range s.list {
		item.Tps = float64(item.Count*1000*1000) / float64(diff/1000)
	}
	return s.list
}
