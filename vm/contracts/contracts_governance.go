package contracts

//
//import (
//	"math/big"
//	"regexp"
//	"strings"
//
//	"github.com/vitelabs/go-vite/common/helper"
//	"github.com/vitelabs/go-vite/common/types"
//	"github.com/vitelabs/go-vite/ledger"
//	"github.com/vitelabs/go-vite/vm/contracts/abi"
//	"github.com/vitelabs/go-vite/vm/util"
//	"github.com/vitelabs/go-vite/vm_db"
//)
//
//type MethodRegister struct {
//	MethodName string
//}
//
//func (p *MethodRegister) GetFee(block *ledger.AccountBlock) (*big.Int, error) {
//	return big.NewInt(0), nil
//}
//func (p *MethodRegister) GetRefundData(sendBlock *ledger.AccountBlock, sbHeight uint64) ([]byte, bool) {
//	return []byte{}, false
//}
//func (p *MethodRegister) GetSendQuota(data []byte, gasTable *util.QuotaTable) (uint64, error) {
//	return gasTable.RegisterQuota, nil
//}
//func (p *MethodRegister) GetReceiveQuota(gasTable *util.QuotaTable) uint64 {
//	return 0
//}
//
//func (p *MethodRegister) DoSend(db vm_db.VmDb, block *ledger.AccountBlock) error {
//	param := new(abi.ParamRegister)
//	if err := abi.ABIGovernance.UnpackMethod(param, p.MethodName, block.Data); err != nil {
//		return util.ErrInvalidMethodParam
//	}
//	if p.MethodName == abi.MethodNameRegisterV3 {
//		param.Gid = types.SNAPSHOT_GID
//	}
//	if !checkRegisterAndVoteParam(param.Gid, param.SbpName) {
//		return util.ErrInvalidMethodParam
//	}
//	if p.MethodName == abi.MethodNameRegisterV3 {
//		block.Data, _ = abi.ABIGovernance.PackMethod(p.MethodName, param.SbpName, param.BlockProducingAddress, param.RewardWithdrawAddress)
//	} else {
//		block.Data, _ = abi.ABIGovernance.PackMethod(p.MethodName, param.Gid, param.SbpName, param.BlockProducingAddress)
//	}
//	return nil
//}
//
//func checkRegisterAndVoteParam(gid types.Gid, name string) bool {
//	if util.IsDelegateGid(gid) ||
//		len(name) == 0 ||
//		len(name) > registrationNameLengthMax {
//		return false
//	}
//	if ok, _ := regexp.MatchString("^([0-9a-zA-Z_.]+[ ]?)*[0-9a-zA-Z_.]$", name); !ok {
//		return false
//	}
//	return true
//}
//
//func (p *MethodRegister) DoReceive(db vm_db.VmDb, block *ledger.AccountBlock, sendBlock *ledger.AccountBlock, vm vmEnvironment) ([]*ledger.AccountBlock, error) {
//	// Check param by group info
//	param := new(abi.ParamRegister)
//	abi.ABIGovernance.UnpackMethod(param, p.MethodName, sendBlock.Data)
//	if p.MethodName == abi.MethodNameRegisterV3 {
//		param.Gid = types.SNAPSHOT_GID
//	} else {
//		param.RewardWithdrawAddress = sendBlock.AccountAddress
//	}
//
//	snapshotBlock := vm.GlobalStatus().SnapshotBlock()
//	if sendBlock.Amount.Cmp(SbpStakeAmountMainnet) != 0 || sendBlock.TokenId != ledger.ViteTokenId {
//		return nil, util.ErrInvalidMethodParam
//	}
//
//	var rewardTime = int64(0)
//	if util.IsSnapshotGid(param.Gid) {
//		rewardTime = snapshotBlock.Timestamp.Unix()
//	}
//
//	// Check registration owner
//	old, err := abi.GetRegistration(db, param.Gid, param.SbpName)
//	util.DealWithErr(err)
//	var hisAddrList []types.Address
//	var oldWithdrawRewardAddress *types.Address
//	if old != nil {
//		if old.IsActive() || old.StakeAddress != sendBlock.AccountAddress {
//			return nil, util.ErrInvalidMethodParam
//		}
//		// old is not active, check old reward drained
//		drained, err := checkRewardDrained(vm.ConsensusReader(), db, old, snapshotBlock)
//		util.DealWithErr(err)
//		if !drained {
//			return nil, util.ErrRewardIsNotDrained
//		}
//		hisAddrList = old.HisAddrList
//		oldWithdrawRewardAddress = &old.RewardWithdrawAddress
//	}
//
//	// check node addr belong to one name in a consensus group
//	hisNameKey := abi.GetHisNameKey(param.BlockProducingAddress, param.Gid)
//	hisName := new(string)
//	v := util.GetValue(db, hisNameKey)
//	if len(v) == 0 {
//		// hisName not exist, update hisName
//		hisAddrList = append(hisAddrList, param.BlockProducingAddress)
//		hisNameData, _ := abi.ABIGovernance.PackVariable(abi.VariableNameRegisteredHisName, param.SbpName)
//		util.SetValue(db, hisNameKey, hisNameData)
//	} else {
//		err = abi.ABIGovernance.UnpackVariable(hisName, abi.VariableNameRegisteredHisName, v)
//		if err != nil || (err == nil && *hisName != param.SbpName) {
//			return nil, util.ErrInvalidMethodParam
//		}
//	}
//
//	groupInfo, err := abi.GetConsensusGroup(db, param.Gid)
//	util.DealWithErr(err)
//	if groupInfo == nil {
//		return nil, util.ErrInvalidMethodParam
//	}
//	stakeParam, _ := abi.GetRegisterStakeParamOfConsensusGroup(groupInfo.RegisterConditionParam)
//
//	var registerInfo []byte
//	// save withdraw reward address -> sbp name
//	saveWithdrawRewardAddress(db, oldWithdrawRewardAddress, param.RewardWithdrawAddress, sendBlock.AccountAddress, param.SbpName)
//	registerInfo, _ = abi.ABIGovernance.PackVariable(
//		abi.VariableNameRegistrationInfoV2,
//		param.SbpName,
//		param.BlockProducingAddress,
//		param.RewardWithdrawAddress,
//		sendBlock.AccountAddress,
//		sendBlock.Amount,
//		snapshotBlock.Height+stakeParam.StakeHeight,
//		rewardTime,
//		int64(0),
//		hisAddrList)
//
//	util.SetValue(db, abi.GetRegistrationInfoKey(param.SbpName, param.Gid), registerInfo)
//	return nil, nil
//}
//
//func saveWithdrawRewardAddress(db vm_db.VmDb, oldAddr *types.Address, newAddr, owner types.Address, sbpName string) {
//	if oldAddr == nil {
//		if newAddr == owner {
//			return
//		}
//		addWithdrawRewardAddress(db, newAddr, sbpName)
//		return
//	}
//	if *oldAddr == newAddr {
//		return
//	}
//	if *oldAddr == owner {
//		addWithdrawRewardAddress(db, newAddr, sbpName)
//		return
//	}
//	if newAddr == owner {
//		deleteWithdrawRewardAddress(db, *oldAddr, sbpName)
//		return
//	}
//	deleteWithdrawRewardAddress(db, *oldAddr, sbpName)
//	addWithdrawRewardAddress(db, newAddr, sbpName)
//}
//
//func addWithdrawRewardAddress(db vm_db.VmDb, addr types.Address, sbpName string) {
//	value := util.GetValue(db, addr.Bytes())
//	if len(value) == 0 {
//		value = []byte(sbpName)
//	} else {
//		value = []byte(string(value) + abi.WithdrawRewardAddressSeparation + sbpName)
//	}
//	util.SetValue(db, addr.Bytes(), value)
//}
//func deleteWithdrawRewardAddress(db vm_db.VmDb, addr types.Address, sbpName string) {
//	value := util.GetValue(db, addr.Bytes())
//	valueStr := string(value)
//	// equal, prefix, suffix, middle
//	if valueStr == sbpName {
//		util.SetValue(db, addr.Bytes(), nil)
//	} else if strings.HasPrefix(valueStr, sbpName+abi.WithdrawRewardAddressSeparation) {
//		util.SetValue(db, addr.Bytes(), []byte(valueStr[len(sbpName)+1:]))
//	} else if strings.HasSuffix(valueStr, abi.WithdrawRewardAddressSeparation+sbpName) {
//		util.SetValue(db, addr.Bytes(), []byte(valueStr[:len(valueStr)-len(sbpName)-1]))
//	} else if startIndex := strings.Index(valueStr, abi.WithdrawRewardAddressSeparation+sbpName+abi.WithdrawRewardAddressSeparation); startIndex > 0 {
//		util.SetValue(db, addr.Bytes(), append([]byte(valueStr[:startIndex]+valueStr[startIndex+len(sbpName)+1:])))
//	}
//}
//
//type MethodRevoke struct {
//	MethodName string
//}
//
//func (p *MethodRevoke) GetFee(block *ledger.AccountBlock) (*big.Int, error) {
//	return big.NewInt(0), nil
//}
//func (p *MethodRevoke) GetRefundData(sendBlock *ledger.AccountBlock, sbHeight uint64) ([]byte, bool) {
//	return []byte{}, false
//}
//func (p *MethodRevoke) GetSendQuota(data []byte, gasTable *util.QuotaTable) (uint64, error) {
//	return gasTable.RevokeQuota, nil
//}
//func (p *MethodRevoke) GetReceiveQuota(gasTable *util.QuotaTable) uint64 {
//	return 0
//}
//
//func (p *MethodRevoke) DoSend(db vm_db.VmDb, block *ledger.AccountBlock) error {
//	if block.Amount.Sign() != 0 {
//		return util.ErrInvalidMethodParam
//	}
//	param := new(abi.ParamCancelRegister)
//	if err := abi.ABIGovernance.UnpackMethod(param, p.MethodName, block.Data); err != nil {
//		return util.ErrInvalidMethodParam
//	}
//	if p.MethodName == abi.MethodNameRevokeV3 {
//		param.Gid = types.SNAPSHOT_GID
//	}
//	if !checkRegisterAndVoteParam(param.Gid, param.SbpName) {
//		return util.ErrInvalidMethodParam
//	}
//	if p.MethodName == abi.MethodNameRevokeV3 {
//		block.Data, _ = abi.ABIGovernance.PackMethod(p.MethodName, param.SbpName)
//	} else {
//		block.Data, _ = abi.ABIGovernance.PackMethod(p.MethodName, param.Gid, param.SbpName)
//	}
//	return nil
//}
//func (p *MethodRevoke) DoReceive(db vm_db.VmDb, block *ledger.AccountBlock, sendBlock *ledger.AccountBlock, vm vmEnvironment) ([]*ledger.AccountBlock, error) {
//	param := new(abi.ParamCancelRegister)
//	abi.ABIGovernance.UnpackMethod(param, p.MethodName, sendBlock.Data)
//	if p.MethodName == abi.MethodNameRevokeV3 {
//		param.Gid = types.SNAPSHOT_GID
//	}
//	snapshotBlock := vm.GlobalStatus().SnapshotBlock()
//	old, err := abi.GetRegistration(db, param.Gid, param.SbpName)
//	util.DealWithErr(err)
//	if old == nil || !old.IsActive() || old.StakeAddress != sendBlock.AccountAddress || old.ExpirationHeight > snapshotBlock.Height {
//		return nil, util.ErrInvalidMethodParam
//	}
//
//	rewardTime := old.RewardTime
//	revokeTime := snapshotBlock.Timestamp.Unix()
//	drained, err := checkRewardDrained(vm.ConsensusReader(), db, old, snapshotBlock)
//	util.DealWithErr(err)
//	if drained {
//		rewardTime = -1
//	}
//	var registerInfo []byte
//	registerInfo, _ = abi.ABIGovernance.PackVariable(
//		abi.VariableNameRegistrationInfoV2,
//		old.Name,
//		old.BlockProducingAddress,
//		old.RewardWithdrawAddress,
//		old.StakeAddress,
//		helper.Big0,
//		uint64(0),
//		rewardTime,
//		revokeTime,
//		old.HisAddrList)
//	util.SetValue(db, abi.GetRegistrationInfoKey(param.SbpName, param.Gid), registerInfo)
//	if old.Amount.Sign() > 0 {
//		return []*ledger.AccountBlock{
//			{
//				AccountAddress: block.AccountAddress,
//				ToAddress:      sendBlock.AccountAddress,
//				BlockType:      ledger.BlockTypeSendCall,
//				Amount:         old.Amount,
//				TokenId:        ledger.ViteTokenId,
//				Data:           []byte{},
//			},
//		}, nil
//	}
//	return nil, nil
//}
//
//func checkRewardDrained(reader util.ConsensusReader, db vm_db.VmDb, old *types.Registration, current *ledger.SnapshotBlock) (bool, error) {
//	panic("checkRewardDrained error")
//}
//
//type MethodUpdateBlockProducingAddress struct {
//	MethodName string
//}
//
//func (p *MethodUpdateBlockProducingAddress) GetFee(block *ledger.AccountBlock) (*big.Int, error) {
//	return big.NewInt(0), nil
//}
//
//func (p *MethodUpdateBlockProducingAddress) GetRefundData(sendBlock *ledger.AccountBlock, sbHeight uint64) ([]byte, bool) {
//	return []byte{}, false
//}
//func (p *MethodUpdateBlockProducingAddress) GetSendQuota(data []byte, gasTable *util.QuotaTable) (uint64, error) {
//	return gasTable.UpdateBlockProducingAddressQuota, nil
//}
//func (p *MethodUpdateBlockProducingAddress) GetReceiveQuota(gasTable *util.QuotaTable) uint64 {
//	return 0
//}
//
//func (p *MethodUpdateBlockProducingAddress) DoSend(db vm_db.VmDb, block *ledger.AccountBlock) error {
//	if block.Amount.Sign() != 0 {
//		return util.ErrInvalidMethodParam
//	}
//	param := new(abi.ParamRegister)
//	if err := abi.ABIGovernance.UnpackMethod(param, p.MethodName, block.Data); err != nil {
//		return util.ErrInvalidMethodParam
//	}
//	if p.MethodName == abi.MethodNameUpdateBlockProducintAddressV3 {
//		param.Gid = types.SNAPSHOT_GID
//	}
//	if !checkRegisterAndVoteParam(param.Gid, param.SbpName) {
//		return util.ErrInvalidMethodParam
//	}
//	if p.MethodName == abi.MethodNameUpdateBlockProducintAddressV3 {
//		block.Data, _ = abi.ABIGovernance.PackMethod(p.MethodName, param.SbpName, param.BlockProducingAddress)
//	} else {
//		block.Data, _ = abi.ABIGovernance.PackMethod(p.MethodName, param.Gid, param.SbpName, param.BlockProducingAddress)
//	}
//	return nil
//}
//func (p *MethodUpdateBlockProducingAddress) DoReceive(db vm_db.VmDb, block *ledger.AccountBlock, sendBlock *ledger.AccountBlock, vm vmEnvironment) ([]*ledger.AccountBlock, error) {
//	param := new(abi.ParamRegister)
//	abi.ABIGovernance.UnpackMethod(param, p.MethodName, sendBlock.Data)
//	if p.MethodName == abi.MethodNameUpdateBlockProducintAddressV3 {
//		param.Gid = types.SNAPSHOT_GID
//	}
//	old, err := abi.GetRegistration(db, param.Gid, param.SbpName)
//	util.DealWithErr(err)
//	if old == nil || !old.IsActive() ||
//		old.StakeAddress != sendBlock.AccountAddress ||
//		old.BlockProducingAddress == param.BlockProducingAddress {
//		return nil, util.ErrInvalidMethodParam
//	}
//	// check node addr belong to one name in a consensus group
//	hisNameKey := abi.GetHisNameKey(param.BlockProducingAddress, param.Gid)
//	hisName := new(string)
//	v := util.GetValue(db, hisNameKey)
//	if len(v) == 0 {
//		// hisName not exist, update hisName
//		old.HisAddrList = append(old.HisAddrList, param.BlockProducingAddress)
//		hisNameData, _ := abi.ABIGovernance.PackVariable(abi.VariableNameRegisteredHisName, param.SbpName)
//		util.SetValue(db, hisNameKey, hisNameData)
//	} else {
//		err = abi.ABIGovernance.UnpackVariable(hisName, abi.VariableNameRegisteredHisName, v)
//		if err != nil || (err == nil && *hisName != param.SbpName) {
//			return nil, util.ErrInvalidMethodParam
//		}
//	}
//	var registerInfo []byte
//	registerInfo, _ = abi.ABIGovernance.PackVariable(
//		abi.VariableNameRegistrationInfoV2,
//		old.Name,
//		param.BlockProducingAddress,
//		old.RewardWithdrawAddress,
//		old.StakeAddress,
//		old.Amount,
//		old.ExpirationHeight,
//		old.RewardTime,
//		old.RevokeTime,
//		old.HisAddrList)
//	util.SetValue(db, abi.GetRegistrationInfoKey(param.SbpName, param.Gid), registerInfo)
//	return nil, nil
//}
