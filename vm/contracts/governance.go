package contracts

import (
	"regexp"

	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/vm/util"
	"github.com/vitelabs/go-vite/vm_db"
)

type sbpInfo struct {
	Name              string // sbp name
	ProducerAddress   types.Address
	OwnerAddress      types.Address
	Status            uint8 // 0/1/2 =>  voting/working/deleted
	VoteType          uint8 // 0/1 => in/out
	VoteExpiredHeight uint64
	Approval          []string // [sbp_name...]
	DisApproval       []string // [sbp_name...]
}

type Governance struct {
	db vm_db.VmDb
}

// 检查是否是一个sbp
func (g Governance) CheckPermission(sbpName string) error {
	info := g.selectBySbpName(sbpName)
	if info == nil {
		return util.ErrInvalidMethodParam
	}
	if info.Status != 1 {
		return util.ErrInvalidMethodParam
	}
	return nil
}

func (g Governance) checkSbpExistForOwner(sbpName string, ownerAddress types.Address) error {
	info := g.selectBySbpName(sbpName)
	if info == nil {
		return util.ErrInvalidMethodParam
	}
	if info.Status != 1 {
		return util.ErrInvalidMethodParam
	}
	if info.OwnerAddress != ownerAddress {
		return util.ErrInvalidMethodParam
	}
	return nil
}

func (g Governance) checkSbpExistForRegister(sbpName string, producerAddress types.Address, curHeight uint64) (*sbpInfo, error) {
	if len(sbpName) == 0 ||
		len(sbpName) > registrationNameLengthMax {
		return nil, util.ErrInvalidMethodParam
	}
	if ok, _ := regexp.MatchString("^([0-9a-zA-Z_.]+[ ]?)*[0-9a-zA-Z_.]$", sbpName); !ok {
		return nil, util.ErrInvalidMethodParam
	}

	oldName := g.selectByProducingAddress(producerAddress)
	if oldName != nil && *oldName != sbpName {
		return nil, util.ErrInvalidMethodParam
	}

	info := g.selectBySbpName(sbpName)
	// 从来都没有注册过
	if info == nil && oldName == nil {
		return nil, nil
	}
	if info == nil {
		return nil, util.ErrInvalidMethodParam
	}
	// 已经注册过了，并正在工作
	if info.Status == 1 || info.Status == 2 {
		return nil, util.ErrInvalidMethodParam
	}
	// 规定时间内重复注册
	if curHeight != uint64(0) && curHeight <= info.VoteExpiredHeight {
		return nil, util.ErrInvalidMethodParam
	}
	return info, nil
}

func (g Governance) checkSbpExistForVoting(sbpName string, voteType uint8, curHeight uint64, proposerSbpName string) (*sbpInfo, error) {
	info := g.selectBySbpName(sbpName)
	// 从来都没有注册过
	if info == nil {
		return nil, util.ErrInvalidMethodParam
	}
	// 已经注销了
	if info.Status == 2 {
		return nil, util.ErrInvalidMethodParam
	}
	if info.Status == 0 {
		if info.VoteType != 0 || voteType != 0 {
			return nil, util.ErrInvalidMethodParam
		}
	}
	if info.Status == 1 {
		if info.VoteType != 1 || voteType != 1 {
			return nil, util.ErrInvalidMethodParam
		}
	}
	// 投票窗口已经关闭
	if curHeight != 0 && info.VoteExpiredHeight < curHeight {
		return nil, util.ErrInvalidMethodParam
	}
	{ // 已经投过票了
		for _, v := range info.Approval {
			if v == proposerSbpName {
				return nil, util.ErrInvalidMethodParam
			}
		}

		for _, v := range info.DisApproval {
			if v == proposerSbpName {
				return nil, util.ErrInvalidMethodParam
			}
		}
	}
	return info, nil
}

func (g Governance) checkSbpExistForRevoke(sbpName string) (*sbpInfo, error) {
	info := g.selectBySbpName(sbpName)
	// 从来都没有注册过
	if info == nil {
		return nil, util.ErrInvalidMethodParam
	}
	// 已经注销了 或者 还没开始工作
	if info.Status == 2 || info.Status == 0 {
		return nil, util.ErrInvalidMethodParam
	}
	return info, nil
}

func (g Governance) checkSbpExistForUpdateProducingAddress(sbpName string, producingAddress types.Address) (*sbpInfo, error) {
	info := g.selectBySbpName(sbpName)
	if info == nil || info.Status != 1 {
		return nil, util.ErrInvalidMethodParam
	}
	if info.Name != sbpName {
		return nil, util.ErrInvalidMethodParam
	}
	address := g.selectByProducingAddress(producingAddress)
	if address != nil {
		return nil, util.ErrInvalidMethodParam
	}
	return info, nil
}

func (g *Governance) addSbpInfo(sbpName string, producingAddress types.Address, ownerAddress types.Address, curHeight uint64) error {
	insert := &sbpInfo{
		Name:              sbpName,
		ProducerAddress:   producingAddress,
		OwnerAddress:      ownerAddress,
		Status:            0,
		VoteType:          0,
		VoteExpiredHeight: curHeight + VotingInterval,
		Approval:          []string{},
		DisApproval:       []string{},
	}
	err := g.insertProducingAddress(producingAddress, sbpName)
	if err != nil {
		return err
	}
	return g.insertSbpInfo(sbpName, insert)
}

var VotingInterval = uint64(60 * 60 * 24 * 3)
var FixSbpNum = 10

func (g Governance) updateSbpInfoForRegister(sbpName string, producingAddress types.Address, ownerAddress types.Address, info *sbpInfo, curHeight uint64) error {
	update := &sbpInfo{
		Name:              info.Name,
		ProducerAddress:   producingAddress,
		OwnerAddress:      ownerAddress,
		Status:            0,
		VoteType:          0,
		VoteExpiredHeight: curHeight + VotingInterval,
		Approval:          []string{},
		DisApproval:       []string{},
	}
	return g.updateSbpInfo(sbpName, update)
}

func (g Governance) updateSbpInfoForVoting(sbpName string, voteType uint8, approval bool, proposerSbpName string, curHeight uint64, info *sbpInfo) error {
	if approval {
		info.Approval = append(info.Approval, proposerSbpName)
	} else {
		info.DisApproval = append(info.DisApproval, proposerSbpName)
	}
	if len(info.Approval)*3 > FixSbpNum*2 {
		if voteType == 0 {
			// 投票进入
			info.Status = 1
		} else {
			// 投票退出
			info.Status = 2
		}
	}
	return g.updateSbpInfo(sbpName, info)
}

func (g Governance) updateSbpInfoForRevoke(sbpName string, proposerSbpName string, curHeight uint64, info *sbpInfo) error {
	if sbpName == proposerSbpName {
		info.Status = 2
	} else {
		info.VoteType = 1
		info.VoteExpiredHeight = curHeight + VotingInterval
		info.Approval = []string{}
		info.DisApproval = []string{}
	}
	return g.updateSbpInfo(sbpName, info)
}
func (g Governance) updateSbpInfoForProducingAddress(sbpName string, producingAddress types.Address, info *sbpInfo) error {
	info.ProducerAddress = producingAddress
	err := g.insertProducingAddress(producingAddress, sbpName)
	if err != nil {
		return err
	}
	return g.updateSbpInfo(sbpName, info)
}

/**
  提议注册一个sbp节点
*/
func (g *Governance) Register(sbpName string, producingAddress types.Address, ownerAddress types.Address, proposerSbpName string, curHeight uint64) error {
	err := g.CheckPermission(proposerSbpName)
	util.AssertNull(err)

	sbpInfo, err := g.checkSbpExistForRegister(sbpName, producingAddress, curHeight)
	util.AssertNull(err)

	if sbpInfo == nil {
		// 添加
		err := g.addSbpInfo(sbpName, producingAddress, ownerAddress, curHeight)
		util.AssertNull(err)
	} else {
		// 更新
		err := g.updateSbpInfoForRegister(sbpName, producingAddress, ownerAddress, sbpInfo, curHeight)
		util.AssertNull(err)
	}
	return nil
}

func (g Governance) Voting(sbpName string, voteType uint8, approval bool, proposerSbpName string, curHeight uint64) error {
	err := g.CheckPermission(proposerSbpName)
	util.AssertNull(err)

	info, err := g.checkSbpExistForVoting(sbpName, voteType, curHeight, proposerSbpName)
	util.AssertNull(info)

	return g.updateSbpInfoForVoting(sbpName, voteType, approval, proposerSbpName, curHeight, info)
}

func (g Governance) Revoke(sbpName string, proposerSbpName string, curHeight uint64) error {
	err := g.CheckPermission(proposerSbpName)
	util.AssertNull(err)

	info, err := g.checkSbpExistForRevoke(sbpName)
	util.AssertNull(err)

	return g.updateSbpInfoForRevoke(sbpName, proposerSbpName, curHeight, info)
}

func (g Governance) UpdateProducingAddress(producingAddress types.Address, proposerSbpName string) error {
	info, err := g.checkSbpExistForUpdateProducingAddress(proposerSbpName, producingAddress)
	util.AssertNull(err)

	return g.updateSbpInfoForProducingAddress(proposerSbpName, producingAddress, info)
}

func (g *Governance) updateSbpInfo(name string, info *sbpInfo) error {

}
func (g *Governance) insertSbpInfo(name string, info *sbpInfo) error {
	// insert sbp_name => sbp_info
}
func (g *Governance) insertProducingAddress(producingAddress types.Address, sbpName string) error {
	// insert producer_address => sbp_name
}
func (g Governance) selectBySbpName(sbpName string) *sbpInfo {

}

func (g Governance) selectByProducingAddress(producingAddress types.Address) *string {

}
