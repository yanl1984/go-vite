package config

import (
	"math/big"

	"github.com/vitelabs/go-vite/common/types"
)

type Genesis struct {
	GenesisAccountAddress *types.Address
	ForkPoints            *ForkPoints
	ConsensusGroupInfo    *ConsensusGroupContractInfo
	MintageInfo           *MintageContractInfo
	PledgeInfo            *PledgeContractInfo
	AccountBalanceMap     map[string]map[string]*big.Int // address - tokenId - balanceAmount
	DexFundInfo           *DexFundContractInfo
	DexTradeInfo          *DexTradeContractInfo
}

func IsCompleteGenesisConfig(genesisConfig *Genesis) bool {
	if genesisConfig == nil || genesisConfig.GenesisAccountAddress == nil ||
		genesisConfig.ConsensusGroupInfo == nil || len(genesisConfig.ConsensusGroupInfo.ConsensusGroupInfoMap) == 0 ||
		len(genesisConfig.ConsensusGroupInfo.RegistrationInfoMap) == 0 ||
		genesisConfig.MintageInfo == nil || len(genesisConfig.MintageInfo.TokenInfoMap) == 0 ||
		len(genesisConfig.AccountBalanceMap) == 0 {
		return false
	}
	return true
}

type ForkPoint struct {
	Height  uint64
	Version uint32
}

type ForkPoints struct {
	SeedFork *ForkPoint
}

type GenesisVmLog struct {
	Data   string
	Topics []types.Hash
}

type ConsensusGroupContractInfo struct {
	ConsensusGroupInfoMap map[string]ConsensusGroupInfo          // consensus group info, gid - info
	RegistrationInfoMap   map[string]map[string]RegistrationInfo // registration info, gid - nodeName - info
	HisNameMap            map[string]map[string]string           // used node name for node addr, gid - nodeAddr - nodeName
	VoteStatusMap         map[string]map[string]string           // vote info, gid - voteAddr - nodeName
}

type MintageContractInfo struct {
	TokenInfoMap map[string]TokenInfo // tokenId - info
	LogList      []GenesisVmLog       // mint events
}

type PledgeContractInfo struct {
	PledgeInfoMap       map[string][]PledgeInfo
	PledgeBeneficialMap map[string]*big.Int
}

type ConsensusGroupInfo struct {
	NodeCount              uint8
	Interval               int64
	PerCount               int64
	RandCount              uint8
	RandRank               uint8
	Repeat                 uint16
	CheckLevel             uint8
	CountingTokenId        types.TokenTypeId
	RegisterConditionId    uint8
	RegisterConditionParam RegisterConditionParam
	VoteConditionId        uint8
	VoteConditionParam     VoteConditionParam
	Owner                  types.Address
	PledgeAmount           *big.Int
	WithdrawHeight         uint64
}
type RegisterConditionParam struct {
	PledgeAmount *big.Int
	PledgeToken  types.TokenTypeId
	PledgeHeight uint64
}
type VoteConditionParam struct {
}
type RegistrationInfo struct {
	NodeAddr       types.Address
	PledgeAddr     types.Address
	Amount         *big.Int
	WithdrawHeight uint64
	RewardTime     int64
	CancelTime     int64
	HisAddrList    []types.Address
}
type TokenInfo struct {
	TokenName      string
	TokenSymbol    string
	TotalSupply    *big.Int
	Decimals       uint8
	Owner          types.Address
	PledgeAmount   *big.Int
	PledgeAddr     types.Address
	WithdrawHeight uint64
	MaxSupply      *big.Int
	OwnerBurnOnly  bool
	IsReIssuable   bool
}
type PledgeInfo struct {
	Amount         *big.Int
	WithdrawHeight uint64
	BeneficialAddr types.Address
}

type DexFundContractInfo struct {
	Owner                 types.Address
	Timer                 types.Address
	Trigger               types.Address
	Maintainer            types.Address
	MakerMineProxy        types.Address
	NotifiedTimestamp     int64
	EndorseVxAmount       *big.Int
	Tokens                []DexTokenInfo
	PendingTransferTokens []DexPendingTransferToken
	Markets               []DexMarketInfo
	UserFunds             []DexUserFund
	PledgeVxs             map[types.Address]*big.Int
	PledgeVips            []types.Address
	MakerMinedVxs         map[string]*big.Int
	Inviters              map[types.Address]uint32
}
type DexTokenInfo struct {
	TokenId        types.TokenTypeId
	Decimals       int32
	Symbol         string
	Index          int32
	Owner          types.Address
	QuoteTokenType int32
}
type DexPendingTransferToken struct {
	TokenId types.TokenTypeId
	Origin  types.Address
	New     types.Address
}
type DexMarketInfo struct {
	MarketId           int32
	MarketSymbol       string
	TradeToken         types.TokenTypeId
	QuoteToken         types.TokenTypeId
	QuoteTokenType     int32
	TradeTokenDecimals int32
	QuoteTokenDecimals int32
	TakerBrokerFeeRate int32
	MakerBrokerFeeRate int32
	AllowMine          bool
	Valid              bool
	Owner              types.Address
	Creator            types.Address
	Stopped            bool
	Timestamp          int64
}
type DexUserFund struct {
	Address  types.Address
	Accounts []DexFundUserAcc
}
type DexFundUserAcc struct {
	Token     types.TokenTypeId
	Available *big.Int
	Locked    *big.Int
}

type DexTradeContractInfo struct {
	Markets   []DexMarketInfo
	Orders    []DexTradeOrder
	Timestamp int64
}
type DexTradeOrder struct {
	Id                 string
	Address            types.Address
	MarketId           int32
	Side               bool
	Type               int32
	Price              string
	TakerFeeRate       int32
	MakerFeeRate       int32
	TakerBrokerFeeRate int32
	MakerBrokerFeeRate int32
	Quantity           *big.Int
	Amount             *big.Int
	LockedBuyFee       *big.Int
	Status             int32
	ExecutedQuantity   *big.Int
	ExecutedAmount     *big.Int
	ExecutedBaseFee    *big.Int
	ExecutedBrokerFee  *big.Int
	Timestamp          int64
}
