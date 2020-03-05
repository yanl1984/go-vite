package abi

import (
	"bytes"
	"strings"

	"github.com/vitelabs/go-vite/common/helper"
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/interfaces"
	"github.com/vitelabs/go-vite/vm/abi"
	"github.com/vitelabs/go-vite/vm/util"
)

const (
	jsonGovernance = `
	[
		{"type":"variable","name":"consensusGroupInfo","inputs":[{"name":"nodeCount","type":"uint8"},{"name":"interval","type":"int64"},{"name":"perCount","type":"int64"},{"name":"randCount","type":"uint8"},{"name":"randRank","type":"uint8"},{"name":"repeat","type":"uint16"},{"name":"checkLevel","type":"uint8"},{"name":"countingTokenId","type":"tokenId"},{"name":"registerConditionId","type":"uint8"},{"name":"registerConditionParam","type":"bytes"},{"name":"voteConditionId","type":"uint8"},{"name":"voteConditionParam","type":"bytes"},{"name":"owner","type":"address"},{"name":"stakeAmount","type":"uint256"},{"name":"expirationHeight","type":"uint64"}]},
		{"type":"variable","name":"registerStakeParam","inputs":[{"name":"stakeAmount","type":"uint256"},{"name":"stakeToken","type":"tokenId"},{"name":"stakeHeight","type":"uint64"}]},
		
		{"type":"function","name":"Register", "inputs":[{"name":"gid","type":"gid"},{"name":"sbpName","type":"string"},{"name":"blockProducingAddress","type":"address"}]},
		{"type":"function","name":"RegisterSBP", "inputs":[{"name":"sbpName","type":"string"},{"name":"blockProducingAddress","type":"address"},{"name":"rewardWithdrawAddress","type":"address"}]},
		
		{"type":"function","name":"UpdateRegistration", "inputs":[{"name":"gid","type":"gid"},{"name":"sbpName","type":"string"},{"name":"blockProducingAddress","type":"address"}]},
		{"type":"function","name":"UpdateBlockProducingAddress", "inputs":[{"name":"gid","type":"gid"},{"name":"sbpName","type":"string"},{"name":"blockProducingAddress","type":"address"}]},
		{"type":"function","name":"UpdateSBPBlockProducingAddress", "inputs":[{"name":"sbpName","type":"string"},{"name":"blockProducingAddress","type":"address"}]},
    	
		{"type":"function","name":"UpdateSBPRewardWithdrawAddress", "inputs":[{"name":"sbpName","type":"string"},{"name":"rewardWithdrawAddress","type":"address"}]},
    
		{"type":"function","name":"CancelRegister","inputs":[{"name":"gid","type":"gid"}, {"name":"sbpName","type":"string"}]},
		{"type":"function","name":"Revoke","inputs":[{"name":"gid","type":"gid"}, {"name":"sbpName","type":"string"}]},
		{"type":"function","name":"RevokeSBP","inputs":[{"name":"sbpName","type":"string"}]},

		{"type":"function","name":"Reward","inputs":[{"name":"gid","type":"gid"},{"name":"sbpName","type":"string"},{"name":"receiveAddress","type":"address"}]},
		{"type":"function","name":"WithdrawReward","inputs":[{"name":"gid","type":"gid"},{"name":"sbpName","type":"string"},{"name":"receiveAddress","type":"address"}]},
		{"type":"function","name":"WithdrawSBPReward","inputs":[{"name":"sbpName","type":"string"},{"name":"receiveAddress","type":"address"}]},
		
		{"type":"variable","name":"registrationInfo","inputs":[{"name":"name","type":"string"},{"name":"blockProducingAddress","type":"address"},{"name":"stakeAddress","type":"address"},{"name":"amount","type":"uint256"},{"name":"expirationHeight","type":"uint64"},{"name":"rewardTime","type":"int64"},{"name":"revokeTime","type":"int64"},{"name":"hisAddrList","type":"address[]"}]},
		{"type":"variable","name":"registrationInfoV2","inputs":[{"name":"name","type":"string"},{"name":"blockProducingAddress","type":"address"},{"name":"rewardWithdrawAddress","type":"address"},{"name":"stakeAddress","type":"address"},{"name":"amount","type":"uint256"},{"name":"expirationHeight","type":"uint64"},{"name":"rewardTime","type":"int64"},{"name":"revokeTime","type":"int64"},{"name":"hisAddrList","type":"address[]"}]},
		{"type":"variable","name":"registeredHisName","inputs":[{"name":"name","type":"string"}]},
		
		{"type":"function","name":"Vote", "inputs":[{"name":"gid","type":"gid"},{"name":"sbpName","type":"string"}]},
		{"type":"function","name":"VoteForSBP", "inputs":[{"name":"sbpName","type":"string"}]},
		{"type":"function","name":"CancelVote","inputs":[{"name":"gid","type":"gid"}]},
		{"type":"function","name":"CancelSBPVoting","inputs":[]},

		{"type":"variable","name":"voteInfo","inputs":[{"name":"sbpName","type":"string"}]}
	]`

	VariableNameConsensusGroupInfo = "consensusGroupInfo"
	VariableNameRegisterStakeParam = "registerStakeParam"

	MethodNameRegister                       = "Register"
	MethodNameRegisterV3                     = "RegisterSBP"
	MethodNameRevoke                         = "CancelRegister"
	MethodNameRevokeV2                       = "Revoke"
	MethodNameRevokeV3                       = "RevokeSBP"
	MethodNameWithdrawReward                 = "Reward"
	MethodNameWithdrawRewardV2               = "WithdrawReward"
	MethodNameWithdrawRewardV3               = "WithdrawSBPReward"
	MethodNameUpdateBlockProducingAddress    = "UpdateRegistration"
	MethodNameUpdateBlockProducintAddressV2  = "UpdateBlockProducingAddress"
	MethodNameUpdateBlockProducintAddressV3  = "UpdateSBPBlockProducingAddress"
	MethodNameUpdateSBPRewardWithdrawAddress = "UpdateSBPRewardWithdrawAddress"
	VariableNameRegistrationInfo             = "registrationInfo"
	VariableNameRegistrationInfoV2           = "registrationInfoV2"
	VariableNameRegisteredHisName            = "registeredHisName"

	MethodNameVote         = "Vote"
	MethodNameVoteV3       = "VoteForSBP"
	MethodNameCancelVote   = "CancelVote"
	MethodNameCancelVoteV3 = "CancelSBPVoting"
	VariableNameVoteInfo   = "voteInfo"

	groupInfoKeyPrefixSize    = 1
	voteInfoKeyPrefixSize     = 1
	consensusGroupInfoKeySize = groupInfoKeyPrefixSize + types.GidSize                    // 11byte, 1 + 10byte gid
	registrationInfoKeySize   = 30                                                        //30byte, 10byte gid + 20byte name hash
	voteInfoKeySize           = voteInfoKeyPrefixSize + types.GidSize + types.AddressSize //32byte, 0 + 10byte gid + 21 byte address

	WithdrawRewardAddressSeparation = ","
)

var (
	// ABIGovernance is abi definition of governance contract
	ABIGovernance, _ = abi.JSONToABIContract(strings.NewReader(jsonGovernance))

	groupInfoKeyPrefix = []byte{1}
	voteInfoKeyPrefix  = []byte{0}
)

type Serializable interface {
	Serialize() []interface{}
}

type ParamRegister struct {
	SbpName               string
	BlockProducingAddress types.Address
	OwnerAddress          types.Address

	ProposerSbpName string
}

func (p ParamRegister) Serialize() []interface{} {
	return util.Serialize(p.SbpName, p.BlockProducingAddress, p.OwnerAddress, p.ProposerSbpName)
}

type ParamVoting struct {
	SbpName  string
	VoteType uint8
	Approval bool

	ProposerSbpName string
}

func (p ParamVoting) Serialize() []interface{} {
	return util.Serialize(p.SbpName, p.VoteType, p.Approval, p.ProposerSbpName)
}

type ParamRevoke struct {
	SbpName         string
	ProposerSbpName string
}

func (p ParamRevoke) Serialize() []interface{} {
	return util.Serialize(p.SbpName, p.ProposerSbpName)
}

type ParamUpdateProducingAddress struct {
	SbpName               string
	BlockProducingAddress types.Address
}

func (p ParamUpdateProducingAddress) Serialize() []interface{} {
	return util.Serialize(p.SbpName, p.BlockProducingAddress)
}

type ParamCancelRegister struct {
	Gid     types.Gid
	SbpName string
}

// GetConsensusGroupInfoKey generate db key for consensus group info
func GetConsensusGroupInfoKey(gid types.Gid) []byte {
	return append(groupInfoKeyPrefix, gid.Bytes()...)
}
func getGidFromConsensusGroupInfoKey(key []byte) types.Gid {
	gid, _ := types.BytesToGid(key[groupInfoKeyPrefixSize:])
	return gid
}
func isConsensusGroupInfoKey(key []byte) bool {
	return len(key) == consensusGroupInfoKeySize
}

// GetRegistrationInfoKey generate db key for registration info
func GetRegistrationInfoKey(name string, gid types.Gid) []byte {
	return append(gid.Bytes(), types.DataHash([]byte(name)).Bytes()[:registrationInfoKeySize-types.GidSize]...)
}

func isRegistrationInfoKey(key []byte) bool {
	return len(key) == registrationInfoKeySize
}

// GetHisNameKey generate db key for registered history name of block producing address
func GetHisNameKey(addr types.Address, gid types.Gid) []byte {
	return append(addr.Bytes(), gid.Bytes()...)
}

// GetVoteInfoKey generate db key for vote info
func GetVoteInfoKey(addr types.Address, gid types.Gid) []byte {
	return helper.JoinBytes(voteInfoKeyPrefix, gid.Bytes(), addr.Bytes())
}

func GetVoteKey(addr types.Address) []byte {
	return nil
}

func getVoteInfoKeyPerfixByGid(gid types.Gid) []byte {
	return append(voteInfoKeyPrefix, gid.Bytes()...)
}

func isVoteInfoKey(key []byte) bool {
	return len(key) == voteInfoKeySize
}

func getAddrFromVoteInfoKey(key []byte) types.Address {
	addr, _ := types.BytesToAddress(key[1+types.GidSize:])
	return addr
}

// GetConsensusGroupList query all consensus group info list
func GetConsensusGroupList(db StorageDatabase) ([]*types.ConsensusGroupInfo, error) {
	if *db.Address() != types.AddressGovernance {
		return nil, util.ErrAddressNotMatch
	}
	iterator, err := db.NewStorageIterator(groupInfoKeyPrefix)
	if err != nil {
		return nil, err
	}
	defer iterator.Release()
	consensusGroupInfoList := make([]*types.ConsensusGroupInfo, 0)
	for {
		if !iterator.Next() {
			if iterator.Error() != nil {
				return nil, iterator.Error()
			}
			break
		}
		if !filterKeyValue(iterator.Key(), iterator.Value(), isConsensusGroupInfoKey) {
			continue
		}
		if info, err := parseConsensusGroup(iterator.Value(), getGidFromConsensusGroupInfoKey(iterator.Key())); err == nil && info != nil && info.IsActive() {
			consensusGroupInfoList = append(consensusGroupInfoList, info)
		} else {
			return nil, err
		}
	}
	return consensusGroupInfoList, nil
}

// GetConsensusGroup query consensus group info by id
func GetConsensusGroup(db StorageDatabase, gid types.Gid) (*types.ConsensusGroupInfo, error) {
	if *db.Address() != types.AddressGovernance {
		return nil, util.ErrAddressNotMatch
	}
	data, err := db.GetValue(GetConsensusGroupInfoKey(gid))
	if err != nil {
		return nil, err
	}
	if len(data) > 0 {
		return parseConsensusGroup(data, gid)
	}
	return nil, nil
}

func parseConsensusGroup(data []byte, gid types.Gid) (*types.ConsensusGroupInfo, error) {
	consensusGroupInfo := new(types.ConsensusGroupInfo)
	err := ABIGovernance.UnpackVariable(consensusGroupInfo, VariableNameConsensusGroupInfo, data)
	if err == nil {
		consensusGroupInfo.Gid = gid
		return consensusGroupInfo, nil
	}
	return nil, err
}

// GetRegisterStakeParamOfConsensusGroup decode stake param of register sbp
func GetRegisterStakeParamOfConsensusGroup(data []byte) (*VariableRegisterStakeParam, error) {
	stakeParam := new(VariableRegisterStakeParam)
	err := ABIGovernance.UnpackVariable(stakeParam, VariableNameRegisterStakeParam, data)
	stakeParam.StakeAmount = nil
	return stakeParam, err
}

// GetAllRegistrationList query all registration info
func GetAllRegistrationList(db StorageDatabase, gid types.Gid) ([]*types.Registration, error) {
	return getRegistrationList(db, gid, false)
}

// GetCandidateList query all registration info which is not canceled
func GetCandidateList(db StorageDatabase, gid types.Gid) ([]*types.Registration, error) {
	return getRegistrationList(db, gid, true)
}

func getRegistrationList(db StorageDatabase, gid types.Gid, filter bool) ([]*types.Registration, error) {
	if *db.Address() != types.AddressGovernance {
		return nil, util.ErrAddressNotMatch
	}
	var iterator interfaces.StorageIterator
	var err error
	if gid == types.DELEGATE_GID {
		iterator, err = db.NewStorageIterator(types.SNAPSHOT_GID.Bytes())
	} else {
		iterator, err = db.NewStorageIterator(gid.Bytes())
	}
	if err != nil {
		return nil, err
	}
	defer iterator.Release()
	registerList := make([]*types.Registration, 0)
	for {
		if !iterator.Next() {
			if iterator.Error() != nil {
				return nil, iterator.Error()
			}
			break
		}
		if !filterKeyValue(iterator.Key(), iterator.Value(), isRegistrationInfoKey) {
			continue
		}

		if registration, err := UnpackRegistration(iterator.Value()); err == nil {
			if filter {
				if registration.IsActive() {
					registerList = append(registerList, registration)
				}
			} else {
				registerList = append(registerList, registration)
			}
		}
	}
	return registerList, nil
}

// GetRegistrationList query registration info list staked by certain address
func GetRegistrationList(db StorageDatabase, gid types.Gid, stakeAddr types.Address) ([]*types.Registration, error) {
	if *db.Address() != types.AddressGovernance {
		return nil, util.ErrAddressNotMatch
	}
	var iterator interfaces.StorageIterator
	var err error
	if gid == types.DELEGATE_GID {
		iterator, err = db.NewStorageIterator(types.SNAPSHOT_GID.Bytes())
	} else {
		iterator, err = db.NewStorageIterator(gid.Bytes())
	}
	if err != nil {
		return nil, err
	}
	defer iterator.Release()
	registrationList := make([]*types.Registration, 0)
	for {
		if !iterator.Next() {
			if iterator.Error() != nil {
				return nil, iterator.Error()
			}
			break
		}
		if !filterKeyValue(iterator.Key(), iterator.Value(), isRegistrationInfoKey) {
			continue
		}
		if registration, err := UnpackRegistration(iterator.Value()); err == nil && registration.StakeAddress == stakeAddr {
			registrationList = append(registrationList, registration)
		}
	}
	return registrationList, nil
}

var registerInfoValuePrefix = []byte{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 1, 0}

// GetRegistration query registration info by consensus group id and sbp name
func GetRegistration(db StorageDatabase, gid types.Gid, name string) (*types.Registration, error) {
	if *db.Address() != types.AddressGovernance {
		return nil, util.ErrAddressNotMatch
	}
	value, err := db.GetValue(GetRegistrationInfoKey(name, gid))
	if err != nil {
		return nil, err
	}
	if len(value) == 0 {
		return nil, nil
	}

	return UnpackRegistration(value)
}

func UnpackRegistration(value []byte) (*types.Registration, error) {
	registration := new(types.Registration)
	if bytes.Equal(value[:32], registerInfoValuePrefix) {
		if err := ABIGovernance.UnpackVariable(registration, VariableNameRegistrationInfo, value); err == nil {
			registration.RewardWithdrawAddress = registration.StakeAddress
			registration.IsActive()
			return registration, nil
		}
	} else {
		if err := ABIGovernance.UnpackVariable(registration, VariableNameRegistrationInfoV2, value); err == nil {
			return registration, nil
		}
	}
	return nil, nil
}

func PackRegistration(registration types.Registration) ([]byte, error) {
	// todo
	// save withdraw reward address -> sbp name
	//registerInfo, _ := ABIGovernance.PackVariable(
	//	VariableNameRegistrationInfoV2,
	//	registration.Name,
	//	param.BlockProducingAddress,
	//	ownerAddress,
	//	int64(0),
	//	hisAddrList)
	return nil, nil
}
