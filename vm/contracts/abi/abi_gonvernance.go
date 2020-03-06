package abi

import (
	"strings"

	"github.com/vitelabs/go-vite/common/helper"
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/vm/abi"
	"github.com/vitelabs/go-vite/vm/util"
)

const (
	jsonGovernance = `
	[
		{"type":"function","name":"Register","inputs":[{"name":"sbpName","type":"string"},{"name":"blockProducingAddress","type":"address"},{"name":"ownerAddress","type":"address"},{"name":"proposerSbpName","type":"string"}]},
		{"type":"function","name":"Vote","inputs":[{"name":"sbpName","type":"string"},{"name":"voteType","type":"uint8"},{"name":"approval","type":"bool"},{"name":"proposerSbpName","type":"string"}]},
		{"type":"function","name":"Revoke","inputs":[{"name":"sbpName","type":"string"},{"name":"proposerSbpName","type":"string"}]},
		{"type":"function","name":"UpdateSBPBlockProducingAddress", "inputs":[{"name":"sbpName","type":"string"},{"name":"blockProducingAddress","type":"address"}]}
	]`

	MethodNameRegister                    = "Register"
	MethodNameVote                        = "Vote"
	MethodNameRevoke                      = "Revoke"
	MethodNameUpdateBlockProducingAddress = "UpdateSBPBlockProducingAddress"
)

var (
	// ABIGovernance is abi definition of governance contract
	ABIGovernance, err = abi.JSONToABIContract(strings.NewReader(jsonGovernance))

	groupInfoKey        = key{prefix: []byte{0}, prefixSize: 1, size: 1 + types.GidSize}
	sbpInfoKey          = key{prefix: []byte{1}, prefixSize: 1, size: 30}
	producingAddressKey = key{prefix: []byte{2}, prefixSize: 1, size: 1 + types.AddressSize}
)

func init() {
	if err != nil {
		panic(err)
	}
}

type key struct {
	prefix     []byte
	prefixSize int
	size       int
}

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

type ParamVote struct {
	SbpName  string
	VoteType uint8
	Approval bool

	ProposerSbpName string
}

func (p ParamVote) Serialize() []interface{} {
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

func isGroupInfoKey(key []byte) bool {
	return len(key) == groupInfoKey.size
}

// GetGroupInfoKey generate db key for vote info
func GetGroupInfoKey(gid types.Gid) []byte {
	k := groupInfoKey
	return helper.JoinBytes(k.prefix, gid.Bytes())
}

// GetSbpInfoKey generate db key for vote info
func GetSbpInfoKey(sbpName string) []byte {
	k := sbpInfoKey
	return helper.JoinBytes(k.prefix, types.DataHash([]byte(sbpName)).Bytes()[:k.size-k.prefixSize])
}

func isSbpInfoKey(key []byte) bool {
	return len(key) == sbpInfoKey.size
}

func GetProducingAddressKey(addr types.Address) []byte {
	return helper.JoinBytes(producingAddressKey.prefix, addr.Bytes())
}

// GetConsensusGroupList query all consensus group info list
func GetConsensusGroupList(db StorageDatabase) ([]*types.ConsensusGroupInfo, error) {
	selector, err := NewGovernanceSelector(db)
	if err != nil {
		return nil, err
	}
	return selector.SelectGroupList()
}

// GetConsensusGroup query consensus group info by id
func GetConsensusGroup(db StorageDatabase, gid types.Gid) (*types.ConsensusGroupInfo, error) {
	selector, err := NewGovernanceSelector(db)
	if err != nil {
		return nil, err
	}
	return selector.SelectGroupInfo(gid)
}

// GetAllRegistrationList query all registration info
func GetAllRegistrationList(db StorageDatabase, gid types.Gid) ([]*types.Registration, error) {
	selector, err := NewGovernanceSelector(db)
	if err != nil {
		return nil, err
	}
	return selector.SelectRegistrationList(false)
}

// GetCandidateList query all registration info which is not canceled
func GetCandidateList(db StorageDatabase, gid types.Gid) ([]*types.Registration, error) {
	selector, err := NewGovernanceSelector(db)
	if err != nil {
		return nil, err
	}
	return selector.SelectRegistrationList(true)
}

// GetRegistrationList query registration info list staked by certain address
func GetRegistrationList(db StorageDatabase, gid types.Gid, stakeAddr types.Address) ([]*types.Registration, error) {
	selector, err := NewGovernanceSelector(db)
	if err != nil {
		return nil, err
	}
	list, err := selector.SelectRegistrationList(false)
	if err != nil {
		return nil, err
	}
	var result []*types.Registration
	for _, registration := range list {
		if registration.StakeAddress == stakeAddr {
			result = append(result, registration)
			break
		}
	}
	return list, err
}

// GetRegistration query registration info by consensus group id and sbp name
func GetRegistration(db StorageDatabase, gid types.Gid, name string) (*types.Registration, error) {
	selector, err := NewGovernanceSelector(db)
	if err != nil {
		return nil, err
	}
	return selector.SelectRegistration(name)
}
