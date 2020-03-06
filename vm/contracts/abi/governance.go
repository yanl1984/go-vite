package abi

import (
	"encoding/json"
	"github.com/vitelabs/go-vite/config"
	"github.com/vitelabs/go-vite/interfaces"
	"regexp"

	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/vm/util"
	"github.com/vitelabs/go-vite/vm_db"
)

var VotingInterval = uint64(60 * 60 * 24 * 3)
var registrationNameLengthMax = 40

type SbpInfo struct {
	Name              string // sbp name
	ProducingAddress  types.Address
	OwnerAddress      types.Address
	Status            uint8 // 0/1/2 =>  voting/working/deleted
	VoteType          uint8 // 0/1 => in/out
	VoteExpiredHeight uint64
	Approval          []string // [sbp_name...]
	DisApproval       []string // [sbp_name...]
}

func (info SbpInfo) Registration() *types.Registration {
	registration := &types.Registration{
		Name:                  info.Name,
		BlockProducingAddress: info.ProducingAddress,
		RewardWithdrawAddress: info.OwnerAddress,
		StakeAddress:          info.OwnerAddress,
		Amount:                nil,
		ExpirationHeight:      0,
		RewardTime:            0,
		RevokeTime:            0,
		Status:                info.Status,
		HisAddrList:           nil,
	}
	return registration
}

type Governance struct {
	db       vm_db.VmDb
	selector *GovernanceSelector
}

func NewGovernance(db vm_db.VmDb) (*Governance, error) {
	address := db.Address()
	if *address != types.AddressGovernance {
		return nil, util.ErrAddressNotMatch
	}
	selector, err := NewGovernanceSelector(db)
	if err != nil {
		return nil, err
	}
	return &Governance{db: db, selector: selector}, nil
}

// 检查是否是一个sbp
func (g Governance) CheckPermission(sbpName string) error {
	info, err := g.selector.selectBySbpName(sbpName)
	util.AssertNil(err)
	if info == nil {
		return util.ErrInvalidMethodParam
	}
	if info.Status != 1 {
		return util.ErrInvalidMethodParam
	}
	return nil
}

func (g Governance) CheckSbpExistForOwner(sbpName string, ownerAddress types.Address) error {
	info, err := g.selector.selectBySbpName(sbpName)
	util.AssertNil(err)
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

func (g Governance) CheckSbpExistForRegister(sbpName string, producerAddress types.Address, curHeight uint64) (*SbpInfo, error) {
	if len(sbpName) == 0 ||
		len(sbpName) > registrationNameLengthMax {
		return nil, util.ErrInvalidMethodParam
	}
	if ok, _ := regexp.MatchString("^([0-9a-zA-Z_.]+[ ]?)*[0-9a-zA-Z_.]$", sbpName); !ok {
		return nil, util.ErrInvalidMethodParam
	}

	oldName, err := g.selector.selectByProducingAddress(producerAddress)
	util.AssertNil(err)
	if oldName != nil && *oldName != sbpName {
		return nil, util.ErrInvalidMethodParam
	}

	info, err := g.selector.selectBySbpName(sbpName)
	util.AssertNil(err)
	// 从来都没有注册过
	if info != nil {
		// 已经注册过了，并正在工作
		if info.Status == 1 || info.Status == 2 {
			return nil, util.ErrInvalidMethodParam
		}
		// 规定时间内重复注册
		if curHeight != uint64(0) && curHeight <= info.VoteExpiredHeight {
			return nil, util.ErrInvalidMethodParam
		}
	}
	{ // 当前工作节点数检查
		working, err := g.selector.CountWorking()
		if err != nil {
			return nil, err
		}
		groupInfo, err := g.selector.SelectGroupInfo(types.SNAPSHOT_GID)
		if groupInfo == nil {
			return nil, util.ErrInvalidMethodParam
		}
		if err != nil {
			return nil, err
		}
		if working >= int(groupInfo.NodeCount) {
			return nil, util.ErrInvalidMethodParam
		}
	}
	if info == nil && oldName == nil {
		return nil, nil
	}
	return info, nil
}

func (g Governance) CheckSbpExistForVoting(sbpName string, voteType uint8, curHeight uint64, proposerSbpName string) (*SbpInfo, error) {
	info, err := g.selector.selectBySbpName(sbpName)
	util.AssertNil(err)
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
	if info.VoteType == 0 {
		// 当前工作节点数检查
		working, err := g.selector.CountWorking()
		if err != nil {
			return nil, err
		}
		groupInfo, err := g.selector.SelectGroupInfo(types.SNAPSHOT_GID)
		if groupInfo == nil {
			return nil, util.ErrInvalidMethodParam
		}
		if err != nil {
			return nil, err
		}
		if working >= int(groupInfo.NodeCount) {
			return nil, util.ErrInvalidMethodParam
		}
	}
	return info, nil
}

func (g Governance) CheckSbpExistForRevoke(sbpName string) (*SbpInfo, error) {
	info, err := g.selector.selectBySbpName(sbpName)
	util.AssertNil(err)
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

func (g Governance) CheckSbpExistForUpdateProducingAddress(sbpName string, producingAddress types.Address) (*SbpInfo, error) {
	info, err := g.selector.selectBySbpName(sbpName)
	util.AssertNil(err)
	if info == nil || info.Status != 1 {
		return nil, util.ErrInvalidMethodParam
	}
	if info.Name != sbpName {
		return nil, util.ErrInvalidMethodParam
	}
	address, err := g.selector.selectByProducingAddress(producingAddress)
	util.AssertNil(err)
	if address != nil {
		return nil, util.ErrInvalidMethodParam
	}
	return info, nil
}

func (g *Governance) addSbpInfo(sbpName string, producingAddress types.Address, ownerAddress types.Address, curHeight uint64) error {
	insert := &SbpInfo{
		Name:              sbpName,
		ProducingAddress:  producingAddress,
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

func (g *Governance) updateSbpInfoForRegister(sbpName string, producingAddress types.Address, ownerAddress types.Address, info *SbpInfo, curHeight uint64) error {
	update := &SbpInfo{
		Name:              info.Name,
		ProducingAddress:  producingAddress,
		OwnerAddress:      ownerAddress,
		Status:            0,
		VoteType:          0,
		VoteExpiredHeight: curHeight + VotingInterval,
		Approval:          []string{},
		DisApproval:       []string{},
	}
	return g.updateSbpInfo(sbpName, update)
}

func (g *Governance) updateSbpInfoForVoting(sbpName string, voteType uint8, approval bool, proposerSbpName string, curHeight uint64, info *SbpInfo) error {
	if approval {
		info.Approval = append(info.Approval, proposerSbpName)
	} else {
		info.DisApproval = append(info.DisApproval, proposerSbpName)
	}
	workingCnt, err := g.selector.CountWorking()
	util.AssertNil(err)
	if len(info.Approval)*3 > workingCnt*2 {
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

func (g *Governance) updateSbpInfoForRevoke(sbpName string, proposerSbpName string, curHeight uint64, info *SbpInfo) error {
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
func (g *Governance) updateSbpInfoForProducingAddress(sbpName string, producingAddress types.Address, info *SbpInfo) error {
	info.ProducingAddress = producingAddress
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
	if err != nil {
		return err
	}

	sbpInfo, err := g.CheckSbpExistForRegister(sbpName, producingAddress, curHeight)
	if err != nil {
		return err
	}

	if sbpInfo == nil {
		// 添加
		err := g.addSbpInfo(sbpName, producingAddress, ownerAddress, curHeight)
		util.AssertNil(err)
	} else {
		// 更新
		err := g.updateSbpInfoForRegister(sbpName, producingAddress, ownerAddress, sbpInfo, curHeight)
		util.AssertNil(err)
	}
	return nil
}

func (g *Governance) Voting(sbpName string, voteType uint8, approval bool, proposerSbpName string, curHeight uint64) error {
	err := g.CheckPermission(proposerSbpName)
	if err != nil {
		return err
	}

	info, err := g.CheckSbpExistForVoting(sbpName, voteType, curHeight, proposerSbpName)
	if err != nil {
		return err
	}

	return g.updateSbpInfoForVoting(sbpName, voteType, approval, proposerSbpName, curHeight, info)
}

func (g *Governance) Revoke(sbpName string, proposerSbpName string, curHeight uint64) error {
	err := g.CheckPermission(proposerSbpName)
	if err != nil {
		return err
	}

	info, err := g.CheckSbpExistForRevoke(sbpName)
	if err != nil {
		return err
	}

	return g.updateSbpInfoForRevoke(sbpName, proposerSbpName, curHeight, info)
}

func (g *Governance) UpdateProducingAddress(producingAddress types.Address, proposerSbpName string) error {
	info, err := g.CheckSbpExistForUpdateProducingAddress(proposerSbpName, producingAddress)
	if err != nil {
		return err
	}

	return g.updateSbpInfoForProducingAddress(proposerSbpName, producingAddress, info)
}

func (g *Governance) updateSbpInfo(name string, info *SbpInfo) error {
	return g.insertSbpInfo(name, info)
}

func (g *Governance) insertSbpInfo(name string, info *SbpInfo) error {
	// insert sbp_name => sbp_info
	data, err := json.Marshal(info)
	if err != nil {
		return err
	}
	return g.db.SetValue(GetSbpInfoKey(name), data)
}
func (g *Governance) insertProducingAddress(producingAddress types.Address, sbpName string) error {
	// insert producer_address => sbp_name
	return g.db.SetValue(GetProducingAddressKey(producingAddress), []byte(sbpName))
}

func (g *Governance) InitGroupInfo(gid types.Gid, cfg *config.ConsensusGroupInfo) error {
	info := types.ConsensusGroupInfo{
		Gid:              gid,
		NodeCount:        cfg.NodeCount,
		Interval:         cfg.Interval,
		PerCount:         cfg.PerCount,
		RandCount:        cfg.RandCount,
		RandRank:         cfg.RandRank,
		Repeat:           cfg.Repeat,
		CheckLevel:       cfg.CheckLevel,
		CountingTokenId:  cfg.CountingTokenId,
		Owner:            cfg.Owner,
		StakeAmount:      cfg.StakeAmount,
		ExpirationHeight: cfg.ExpirationHeight,
	}
	data, err := json.Marshal(info)
	if err != nil {
		return err
	}
	return g.db.SetValue(GetGroupInfoKey(gid), data)
}

func (g *Governance) InitRegistration(name string, cfg *config.RegistrationInfo) error {
	err := g.insertSbpInfo(name, &SbpInfo{
		Name:              name,
		ProducingAddress:  *cfg.BlockProducingAddress,
		OwnerAddress:      *cfg.StakeAddress,
		Status:            1,
		VoteType:          0,
		VoteExpiredHeight: 0,
		Approval:          []string{},
		DisApproval:       []string{},
	})
	return err
}

type GovernanceSelector struct {
	db StorageDatabase
}

func NewGovernanceSelector(db StorageDatabase) (*GovernanceSelector, error) {
	address := db.Address()
	if *address != types.AddressGovernance {
		return nil, util.ErrAddressNotMatch
	}
	return &GovernanceSelector{db: db}, nil
}

func (g GovernanceSelector) selectByProducingAddress(producingAddress types.Address) (*string, error) {
	value, err := g.db.GetValue(GetProducingAddressKey(producingAddress))
	if err != nil {
		return nil, err
	}
	if len(value) <= 0 {
		return nil, nil
	}
	name := string(value)
	return &name, nil
}

func (g GovernanceSelector) selectBySbpName(sbpName string) (*SbpInfo, error) {
	value, err := g.db.GetValue(GetSbpInfoKey(sbpName))
	if err != nil {
		return nil, err
	}
	return g.unpackSbpInfo(value)
}

func (g GovernanceSelector) unpackSbpInfo(value []byte) (*SbpInfo, error) {
	if len(value) <= 0 {
		return nil, nil
	}
	info := &SbpInfo{}
	err := json.Unmarshal(value, info)
	if err != nil {
		return nil, err
	}
	return info, nil
}

func (g GovernanceSelector) unpackGroupInfo(value []byte) (*types.ConsensusGroupInfo, error) {
	if len(value) <= 0 {
		return nil, nil
	}
	info := &types.ConsensusGroupInfo{}
	err := json.Unmarshal(value, info)
	if err != nil {
		return nil, err
	}
	return info, nil
}

func (g GovernanceSelector) SelectRegistration(sbpName string) (*types.Registration, error) {
	info, err := g.selectBySbpName(sbpName)
	if err != nil {
		return nil, err
	}
	if info == nil {
		return nil, nil
	}
	return info.Registration(), nil
}

func (g GovernanceSelector) selectSbpList(workingFilter bool) ([]*SbpInfo, error) {
	var iterator interfaces.StorageIterator
	var err error

	iterator, err = g.db.NewStorageIterator(sbpInfoKey.prefix)
	if err != nil {
		return nil, err
	}
	defer iterator.Release()

	result := make([]*SbpInfo, 0)
	for {
		if !iterator.Next() {
			if iterator.Error() != nil {
				return nil, iterator.Error()
			}
			break
		}
		value := iterator.Value()
		key := iterator.Key()
		if !filterKeyValue(key, value, isSbpInfoKey) {
			continue
		}
		sbpInfo, err := g.unpackSbpInfo(value)
		if err != nil {
			return nil, err
		}
		if sbpInfo == nil {
			continue
		}

		if workingFilter {
			if sbpInfo.Status == 1 {
				result = append(result, sbpInfo)
			}
		} else {
			result = append(result, sbpInfo)
		}
	}
	return result, nil
}

func (g GovernanceSelector) CountWorking() (int, error) {
	list, err := g.SelectRegistrationList(true)
	return len(list), err
}

func (g GovernanceSelector) SelectRegistrationList(workingFilter bool) ([]*types.Registration, error) {
	list, err := g.selectSbpList(workingFilter)
	if err != nil {
		return nil, err
	}
	var result []*types.Registration
	for _, info := range list {
		result = append(result, info.Registration())
	}
	return result, nil
}

func (g GovernanceSelector) SelectGroupInfo(gid types.Gid) (*types.ConsensusGroupInfo, error) {
	value, err := g.db.GetValue(GetGroupInfoKey(gid))
	if err != nil {
		return nil, err
	}
	return g.unpackGroupInfo(value)
}

func (g GovernanceSelector) SelectGroupList() ([]*types.ConsensusGroupInfo, error) {
	var iterator interfaces.StorageIterator
	var err error

	iterator, err = g.db.NewStorageIterator(groupInfoKey.prefix)
	if err != nil {
		return nil, err
	}
	defer iterator.Release()

	result := make([]*types.ConsensusGroupInfo, 0)
	for {
		if !iterator.Next() {
			if iterator.Error() != nil {
				return nil, iterator.Error()
			}
			break
		}
		value := iterator.Value()
		key := iterator.Key()
		if !filterKeyValue(key, value, isGroupInfoKey) {
			continue
		}
		info, err := g.unpackGroupInfo(value)
		if err != nil {
			return nil, err
		}
		if info == nil {
			continue
		}
		result = append(result, info)
	}
	return result, nil
}

func (g GovernanceSelector) SelectState() ([]*SbpInfo, []*types.ConsensusGroupInfo, error) {
	registrationList, _ := g.SelectGroupList()
	sbpList, _ := g.selectSbpList(false)
	return sbpList, registrationList, nil
}
