package contracts

import (
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/vm/util"
	"math/big"
)

const (
	cgNodeCountMin   uint8 = 3       // Minimum node count of consensus group
	cgNodeCountMax   uint8 = 101     // Maximum node count of consensus group
	cgIntervalMin    int64 = 1       // Minimum interval of consensus group in second
	cgIntervalMax    int64 = 10 * 60 // Maximum interval of consensus group in second
	cgPerCountMin    int64 = 1
	cgPerCountMax    int64 = 10 * 60
	cgPerIntervalMin int64 = 1
	cgPerIntervalMax int64 = 10 * 60

	registrationNameLengthMax int = 40

	tokenNameLengthMax   int = 40 // Maximum length of a token name(include)
	tokenSymbolLengthMax int = 10 // Maximum length of a token symbol(include)

	tokenNameIndexMax  uint16 = 1000
	GetRewardTimeLimit int64  = 3600 // Cannot get snapshot block reward of current few blocks, for latest snapshot block could be reverted

	PledgeHeightMax uint64 = 3600 * 24 * 365

	TimerTimeWindowMin   uint64 = 75
	TimerHeightWindowMin uint64 = 75
	TimerTimeGapMin      uint64 = 75
	TimerTimeGapMax      uint64 = 90 * 24 * 3600
	TimerHeightGapMin    uint64 = 75
	TimerHeightGapMax    uint64 = 90 * 24 * 3600

	TimerTriggerTasksNumMax    int    = 100
	TimerArrearageDeleteHeight uint64 = 7 * 24 * 3600
)

var (
	rewardPerBlock  = big.NewInt(951293759512937595) // Reward pre snapshot block, rewardPreBlock * blockNumPerYear / viteTotalSupply = 3%
	pledgeAmountMin = new(big.Int).Mul(big.NewInt(134), util.AttovPerVite)
	mintageFee      = new(big.Int).Mul(big.NewInt(1e3), util.AttovPerVite) // Mintage cost choice 1, destroy ViteToken
	// createConsensusGroupPledgeAmount = new(big.Int).Mul(big.NewInt(1000), util.AttovPerVite)
	timerChargeAmountPerTask = new(big.Int).Mul(big.NewInt(10), util.AttovPerVite)
	timerNewTaskFee          = new(big.Int).Mul(big.NewInt(10), util.AttovPerVite)
	timerBurnFeeMin          = new(big.Int).Mul(big.NewInt(1e4), util.AttovPerVite)
)

type ContractsParams struct {
	RegisterMinPledgeHeight          uint64 // Minimum pledge height for register
	PledgeHeight                     uint64 // pledge height for stake
	CreateConsensusGroupPledgeHeight uint64 // Pledge height for registering to be a super node of snapshot group and common delegate group
	MintPledgeHeight                 uint64 // Pledge height for mintage if choose to pledge instead of destroy vite token
	ViteXVipPledgeHeight             uint64 // Pledge height for dex_fund contract, in order to upgrade to viteX vip
	TimerOwnerAddressDefault         types.Address
}

var (
	TimerOwnerAddressDefaultTest, _       = types.HexToAddress("vite_ab24ef68b84e642c0ddca06beec81c9acb1977bbd7da27a87a")
	TimerOwnerAddressDefaultPreMainnet, _ = types.HexToAddress("vite_6f4746c037e3cacff609a01c2ad6ef7dac130a4725ec94b55a")

	ContractsParamsTest = ContractsParams{
		RegisterMinPledgeHeight:          1,
		PledgeHeight:                     1,
		CreateConsensusGroupPledgeHeight: 1,
		MintPledgeHeight:                 1,
		ViteXVipPledgeHeight:             1,
		TimerOwnerAddressDefault:         TimerOwnerAddressDefaultTest,
	}
	ContractsParamsMainNet = ContractsParams{
		RegisterMinPledgeHeight:          3600 * 24 * 3,
		PledgeHeight:                     3600 * 24 * 3,
		CreateConsensusGroupPledgeHeight: 3600 * 24 * 3,
		MintPledgeHeight:                 3600 * 24 * 30 * 3,
		ViteXVipPledgeHeight:             3600 * 24 * 30,
		TimerOwnerAddressDefault:         TimerOwnerAddressDefaultPreMainnet,
	}
)
