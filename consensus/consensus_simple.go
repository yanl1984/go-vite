package consensus

import (
	"time"

	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/consensus/core"
	"github.com/vitelabs/go-vite/ledger"
	"github.com/vitelabs/go-vite/log15"
)

var simpleGenesis = time.Unix(1553849738, 0)

var simpleAddrs = genSimpleAddrs()

func genSimpleAddrs() []types.Address {
	var simpleAddrs []types.Address
	addrs := []string{"vite_360232b0378111b122685a15e612143dc9a89cfa7e803f4b5a",
		"vite_ce18b99b46c70c8e6bf34177d0c5db956a8c3ea7040a1c1e25",
		"vite_40996a2ba285ad38930e09a43ee1bd0d84f756f65318e8073a",
		"vite_c8c70c248536117c54d5ffd9724428c58c7fc57f3183508b3d",
		"vite_50f3ede3d3098ae7236f65bb320578ca13dd52516aafc1a10c"}

	for _, v := range addrs {
		addr, err := types.HexToAddress(v)
		if err != nil {
			panic(err)
		}
		simpleAddrs = append(simpleAddrs, addr)
	}
	return simpleAddrs
}

func genSimpleInfo() *core.GroupInfo {
	group := types.ConsensusGroupInfo{
		Gid:                    types.SNAPSHOT_GID,
		NodeCount:              5,
		Interval:               1,
		PerCount:               3,
		RandCount:              1,
		RandRank:               100,
		CountingTokenId:        types.CreateTokenTypeId(),
		RegisterConditionId:    0,
		RegisterConditionParam: nil,
		VoteConditionId:        0,
		VoteConditionParam:     nil,
		Owner:                  types.Address{},
		PledgeAmount:           nil,
		WithdrawHeight:         0,
	}

	info := core.NewGroupInfo(simpleGenesis, group)
	return info
}

type simpleCs struct {
	consensusDpos
	algo core.Algo

	log log15.Logger
}

func newSimpleCs(log log15.Logger) *simpleCs {
	cs := &simpleCs{}
	cs.log = log.New("gid", "snapshot")

	cs.info = genSimpleInfo()
	cs.algo = core.NewAlgo(cs.info)
	return cs
}

func (self *simpleCs) GenVoteTime(h uint64) time.Time {
	_, end := self.info.Index2Time(h)
	return end
}

func (self *simpleCs) ElectionTime(t time.Time) (*electionResult, error) {
	index := self.info.Time2Index(t)
	return self.ElectionIndex(index)
}

func (self *simpleCs) ElectionIndex(index uint64) (*electionResult, error) {
	plans := genElectionResult(self.info, index, simpleAddrs)
	return plans, nil
}

func (self *simpleCs) VerifySnapshotProducer(header *ledger.SnapshotBlock) (bool, error) {
	electionResult, err := self.ElectionTime(*header.Timestamp)
	if err != nil {
		return false, err
	}

	return self.verifyProducer(*header.Timestamp, header.Producer(), electionResult), nil
}

func (self *simpleCs) verifyProducer(t time.Time, address types.Address, result *electionResult) bool {
	if result == nil {
		return false
	}
	for _, plan := range result.Plans {
		if plan.Member == address {
			if plan.STime == t {
				return true
			}
		}
	}
	return false
}