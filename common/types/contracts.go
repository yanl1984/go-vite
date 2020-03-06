package types

import (
	"encoding/binary"
	"math/big"
)

type ConsensusGroupInfo struct {
	Gid                    Gid         // Consensus group id
	NodeCount              uint8       // Active miner count
	Interval               int64       // Timestamp gap between two continuous block
	PerCount               int64       // Continuous block generation interval count
	RandCount              uint8       // Random miner count
	RandRank               uint8       // Chose random miner with chain rank limit of vote
	Repeat                 uint16      // reuse consensus info to produce blocks within repeat times
	CheckLevel             uint8       // consensus check param, 0-check address and sequence, 1-check address only
	CountingTokenId        TokenTypeId // Token id for selecting miner through vote
	RegisterConditionId    uint8
	Owner                  Address
	StakeAmount            *big.Int
	ExpirationHeight       uint64
}

func (groupInfo *ConsensusGroupInfo) IsActive() bool {
	return groupInfo.ExpirationHeight > 0
}

type VoteInfo struct {
	VoteAddr Address
	SbpName  string
}

type Registration struct {
	Name                  string
	BlockProducingAddress Address
	RewardWithdrawAddress Address
	StakeAddress          Address
	Amount                *big.Int
	ExpirationHeight      uint64
	RewardTime            int64
	RevokeTime            int64
	Status                uint8
	HisAddrList           []Address
}

type VotingSbp struct {
	SbpName     string
	StartHeight uint64
	EndHeight   uint64
}

func (v VotingSbp) ID() []byte {
	var buf = make([]byte, 8)
	binary.BigEndian.PutUint64(buf, v.StartHeight)
	return append(DataHash([]byte(v.SbpName)).Bytes(), buf...)
}

func (r *Registration) IsActive() bool {
	return r.RevokeTime == 0
}

type TokenInfo struct {
	TokenName     string   `json:"tokenName"`
	TokenSymbol   string   `json:"tokenSymbol"`
	TotalSupply   *big.Int `json:"totalSupply"`
	Decimals      uint8    `json:"decimals"`
	Owner         Address  `json:"owner"`
	MaxSupply     *big.Int `json:"maxSupply"`
	OwnerBurnOnly bool     `json:"ownerBurnOnly"`
	IsReIssuable  bool     `json:"isReIssuable"`
	Index         uint16   `json:"index"`
}

type StakeInfo struct {
	Amount           *big.Int `json:"amount"`
	ExpirationHeight uint64   `json:"withdrawHeight"`
	Beneficiary      Address  `json:"beneficialAddr"`
	IsDelegated      bool     `json:"agent"`
	DelegateAddress  Address  `json:"agentAddr"`
	Bid              uint8    `json:"bid"`
	StakeAddress     Address  `json:"pledgeAddr"`
	Id               *Hash    `json:"id"`
}
