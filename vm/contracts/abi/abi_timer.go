package abi

import (
	"bytes"
	"github.com/vitelabs/go-vite/common/helper"
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/vm/abi"
	"github.com/vitelabs/go-vite/vm/util"
	"math/big"
	"strings"
)

const (
	jsonTimer = `
	[
		{"type":"function","name":"NewTimer", "inputs":[{"name":"timerType","type":"uint64"},{"name":"start","type":"uint64"},{"name":"window","type":"uint64"},{"name":"interval","type":"uint64"},{"name":"expiringCondition","type":"uint64"},{"name":"invokingAddr","type":"address"},{"name":"refundAddr","type":"address"}]},
		{"type":"function","name":"DeleteTimer", "inputs":[{"name":"timerId","type":"bytes32"},{"name":"refundAddr","type":"address"}]},
		{"type":"function","name":"Deposit", "inputs":[{"name":"timerId","type":"bytes32"}]},
		{"type":"function","name":"TransferOwnership", "inputs":[{"name":"newOwnerAddr","type":"address"}]},
		{"type":"variable","name":"timerInfo","inputs":[{"name":"timerId","type":"bytes32"},{"name":"timerType","type":"uint64"},{"name":"window","type":"uint64"},{"name":"interval","type":"uint64"},{"name":"expiringCondition","type":"uint64"},{"name":"invokingAddr","type":"address"},{"name":"refundAddr","type":"address"}]},
		{"type":"variable","name":"timerTriggerInfo","inputs":[{"name":"amount","type":"uint256"},{"name":"triggerTimes","type":"uint64"},{"name":"next","type":"uint64"},{"name":"delete","type":"uint64"}]},
		{"type":"variable","name":"lastTriggerInfo","inputs":[{"name":"timestamp","type":"uint64"},{"name":"height","type":"uint64"}]}
	]`

	jsonTimerNotify = `
	[
		{"type":"function","name":"__notify", "inputs":[{"name":"current","type":"uint64"},{"name":"timerId","type":"bytes32"}]}
	]
	`

	MethodNameTimerNewTimer          = "NewTimer"
	MethodNameTimerDeleteTimer       = "DeleteTimer"
	MethodNameTimerDeposit           = "Deposit"
	MethodNameTimerTransferOwnership = "TransferOwnership"
	MethodNameTimerNotify            = "__notify"
	VariableNameTimerInfo            = "timerInfo"
	VariableNameTimerTriggerInfo     = "timerTriggerInfo"
	VariableNameTimerLastTriggerInfo = "lastTriggerInfo"
)
const (
	TimerTimeHeightTime uint8 = iota + 1
	TimerTimeHeightHeight
)
const (
	TimerExpiringTypeTimeHeight uint8 = iota + 1
	TimerExpiringTypeTimes
	TimerExpiringTypePermanent
)
const (
	TimerIntervalTypePostpone uint8 = iota + 1
	TimerIntervalTypeFixed
)
const (
	TimerChargeTypeFree uint8 = iota + 1
	TimerChargeTypeCharge
)
const (
	timerInfoKeyLen         = 2 + types.AddressSize + 8
	timerQueueKeyLen        = 2 + 1 + 8 + 8
	timerStoppedQueueKeyLen = 2 + 8 + 8
)

/*
 timerInfo: 31 byte, 2 byte prefix + 21 byte address + 8 byte id
 timerTriggerInfo: 31 byte, 2 byte prefix + 21 byte address + 8 byte id
 queue: 19 byte, 2 byte prefix + 1 byte time height type + 8 byte next time height + 8 byte id
 stoppedQueue: 18 byte, 2 byte prefix + 8 byte delete height + 8 byte id
 timerId: 32 byte request hash
 nextId: 3 byte
 lastTriggerInfo: 3 byte
 fee: 3 byte
 owner: 3 byte
*/
var (
	ABITimer, _                = abi.JSONToABIContract(strings.NewReader(jsonTimer))
	ABITimerNotify, _          = abi.JSONToABIContract(strings.NewReader(jsonTimerNotify))
	timerNextIndexKey          = []byte("nid")
	timerLastTriggerInfoKey    = []byte("lti")
	timerFeeKey                = []byte("fee")
	timerOwnerKey              = []byte("own")
	timerInfoKeyPrefix         = []byte{0, 0}
	timerTriggerInfoKeyPrefix  = []byte{0, 1}
	timerQueueKeyPrefix        = []byte{1, 0}
	timerStoppedQueueKeyPrefix = []byte{1, 1}
)

type ParamNewTimer struct {
	TimerType         uint64
	Start             uint64
	ExpiringCondition uint64
	Window            uint64
	Interval          uint64
	InvokingAddr      types.Address
	RefundAddr        types.Address
}

type ParamDeleteTimer struct {
	TimerId    types.Hash
	RefundAddr types.Address
}

type TimerInfo struct {
	TimerId           types.Hash
	TimerType         uint64
	Window            uint64
	Interval          uint64
	ExpiringCondition uint64
	InvokingAddr      types.Address
	RefundAddr        types.Address
	InnerId           []byte
}

type TimerTriggerInfo struct {
	Amount       *big.Int
	TriggerTimes uint64
	Next         uint64
	Delete       uint64
}

func (t *TimerTriggerInfo) IsStopped() bool {
	return t.Delete > 0
}

func (t *TimerTriggerInfo) IsFinish() bool {
	return t.Delete == 0 && t.Next == 0
}

type TimerLastTriggerInfo struct {
	Timestamp uint64
	Height    uint64
}

func GetInnerId(address types.Address, id uint64) []byte {
	return helper.JoinBytes(address.Bytes(), helper.LeftPadBytes(new(big.Int).SetUint64(id).Bytes(), 8))
}
func GetOwnerFromInnerId(innerId []byte) types.Address {
	addr, _ := types.BytesToAddress(innerId[:types.AddressSize])
	return addr
}
func GetIndexFromInnerId(innerId []byte) []byte {
	return innerId[types.AddressSize:]
}
func GetTimerNextIndexKey() []byte {
	return timerNextIndexKey
}
func GetTimerTypeDetail(timerType uint64) (timeHeight, expiringType, intervalType uint8) {
	ct := timerType / 1000
	th := (timerType - ct*1000) / 100
	et := (timerType - ct*1000 - th*100) / 10
	it := timerType - ct*1000 - th*100 - et*10
	return uint8(th), uint8(et), uint8(it)
}
func GetChargeTypeFromTimerType(timerType uint64) uint8 {
	return uint8(timerType / 1000)
}
func GetVariableTimerTypeByParamTimerType(timerType uint64, isOwner bool) uint64 {
	if isOwner {
		return uint64(TimerChargeTypeFree)*1000 + timerType
	}
	return uint64(TimerChargeTypeCharge)*1000 + timerType
}
func GenerateTimerType(timeHeight, expiringType, intervalType uint8) uint64 {
	return uint64(intervalType) + uint64(expiringType)*10 + uint64(timeHeight)*100
}

func GetTimerQueueKey(timeHeight uint8, innerId []byte, next uint64) []byte {
	return helper.JoinBytes(timerQueueKeyPrefix, []byte{byte(timeHeight)}, helper.LeftPadBytes(new(big.Int).SetUint64(next).Bytes(), 8), GetIndexFromInnerId(innerId))
}
func GetTimerNewQueueKey(oldKey []byte, next uint64) []byte {
	return helper.JoinBytes(oldKey[:3], helper.LeftPadBytes(new(big.Int).SetUint64(next).Bytes(), 8), oldKey[11:])
}
func GetTimerQueueKeyPrefix(timeHeight uint8) []byte {
	return helper.JoinBytes(timerQueueKeyPrefix, []byte{byte(timeHeight)})
}
func IsTimerQueueKey(k []byte) bool {
	return len(k) == timerQueueKeyLen && bytes.Equal(k[:2], timerQueueKeyPrefix)
}
func GetNextTriggerFromTimerQueueKey(k []byte) uint64 {
	return new(big.Int).SetBytes(k[3:11]).Uint64()
}
func GetTimerStoppedQueueKey(innerId []byte, delete uint64) []byte {
	return helper.JoinBytes(timerStoppedQueueKeyPrefix, helper.LeftPadBytes(new(big.Int).SetUint64(delete).Bytes(), 8), GetIndexFromInnerId(innerId))
}
func IsTimerStoppedQueueKey(k []byte) bool {
	return len(k) == timerStoppedQueueKeyLen && bytes.Equal(k[:2], timerStoppedQueueKeyPrefix)
}
func GetTimerStoppedQueueKeyPrefix() []byte {
	return timerStoppedQueueKeyPrefix
}
func GetDeleteHeightFromTimerStoppedQueueKey(key []byte) uint64 {
	return new(big.Int).SetBytes(key[2:10]).Uint64()
}
func GetTimerInfoKey(innerId []byte) []byte {
	return helper.JoinBytes(timerInfoKeyPrefix, innerId)
}
func GetTimerInfoKeyPrefix(owner types.Address) []byte {
	return helper.JoinBytes(timerInfoKeyPrefix, owner.Bytes())
}
func GetInnerIdFromTimerInfoKey(k []byte) []byte {
	return k[2:]
}
func IsTimerInfoKey(k []byte) bool {
	return len(k) == timerInfoKeyLen && bytes.Equal(k[:2], timerInfoKeyPrefix)
}
func GetTimerTriggerInfoKey(innerId []byte) []byte {
	return helper.JoinBytes(timerTriggerInfoKeyPrefix, innerId)
}
func GetTimerFeeKey() []byte {
	return timerFeeKey
}
func GetTimerLastTriggerInfoKey() []byte {
	return timerLastTriggerInfoKey
}
func GetTimerOwnerKey() []byte {
	return timerOwnerKey
}
func GetTimerIdKey(timerId types.Hash) []byte {
	return timerId.Bytes()
}

func GetTimerFee(db StorageDatabase) ([]byte, *big.Int, error) {
	key := GetTimerFeeKey()
	value, err := db.GetValue(key)
	if err != nil {
		return nil, nil, err
	}
	return key, new(big.Int).SetBytes(value), err
}

func GetTimerNextIndex(db StorageDatabase) ([]byte, uint64, error) {
	timerNextIndexKey := GetTimerNextIndexKey()
	value, err := db.GetValue(timerNextIndexKey)
	if err != nil {
		return nil, 0, err
	}
	nextIndex := uint64(1)
	if len(value) > 0 {
		nextIndex = new(big.Int).SetBytes(value).Uint64()
	}
	return timerNextIndexKey, nextIndex, nil
}

func GetTimerOwner(db StorageDatabase) (*types.Address, error) {
	value, err := db.GetValue(GetTimerOwnerKey())
	if err != nil {
		return nil, err
	}
	if len(value) > 0 {
		owner, _ := types.BytesToAddress(value)
		return &owner, nil
	}
	return nil, nil
}

func GetInnerIdByTimerId(db StorageDatabase, timerId types.Hash) ([]byte, error) {
	return db.GetValue(GetTimerIdKey(timerId))
}

func GetTimerInfoByInnerId(db StorageDatabase, innerId []byte) ([]byte, *TimerInfo, error) {
	timerInfoKey := GetTimerInfoKey(innerId)
	timerInfoValue, err := db.GetValue(timerInfoKey)
	if err != nil {
		return nil, nil, err
	}
	timerInfo := new(TimerInfo)
	ABITimer.UnpackVariable(timerInfo, VariableNameTimerInfo, timerInfoValue)
	return timerInfoKey, timerInfo, nil
}

func GetTimerTriggerInfoByInnerId(db StorageDatabase, innerId []byte) ([]byte, *TimerTriggerInfo, error) {
	timerTriggerInfoKey := GetTimerTriggerInfoKey(innerId)
	timerTriggerInfoValue, err := db.GetValue(timerTriggerInfoKey)
	if err != nil {
		return nil, nil, err
	}
	timerTriggerInfo := new(TimerTriggerInfo)
	ABITimer.UnpackVariable(timerTriggerInfo, VariableNameTimerTriggerInfo, timerTriggerInfoValue)
	return timerTriggerInfoKey, timerTriggerInfo, nil
}

func GetTimerLastTriggerInfo(db StorageDatabase) (*TimerLastTriggerInfo, error) {
	lastTriggerInfoValue, err := db.GetValue(GetTimerLastTriggerInfoKey())
	if err != nil {
		return nil, err
	}
	if len(lastTriggerInfoValue) > 0 {
		lastTriggerInfo := new(TimerLastTriggerInfo)
		ABITimer.UnpackVariable(lastTriggerInfo, VariableNameTimerLastTriggerInfo, lastTriggerInfoValue)
		return lastTriggerInfo, nil
	}
	return nil, nil
}

func GetTimerInfoListByOwner(db StorageDatabase, owner types.Address) ([]*TimerInfo, error) {
	if *db.Address() != types.AddressTimer {
		return nil, util.ErrAddressNotMatch
	}
	iterator, err := db.NewStorageIterator(GetTimerInfoKeyPrefix(owner))
	if err != nil {
		return nil, err
	}
	defer iterator.Release()
	timerInfoList := make([]*TimerInfo, 0)
	for {
		if !iterator.Next() {
			if iterator.Error() != nil {
				return nil, iterator.Error()
			}
			break
		}
		if !filterKeyValue(iterator.Key(), iterator.Value(), IsTimerInfoKey) {
			continue
		}
		timerInfo := new(TimerInfo)
		if err := ABITimer.UnpackVariable(timerInfo, VariableNameTimerInfo, iterator.Value()); err == nil {
			timerInfo.InnerId = GetInnerIdFromTimerInfoKey(iterator.Key())
			timerInfoList = append(timerInfoList, timerInfo)
		}
	}
	return timerInfoList, nil
}

func GetTimerSummary(db StorageDatabase) (uint64, uint64, error) {
	queueCount := uint64(0)
	stoppedQueueCount := uint64(0)
	iterator, err := db.NewStorageIterator([]byte{1})
	if err != nil {
		return 0, 0, err
	}
	defer iterator.Release()
	for {
		if !iterator.Next() {
			if iterator.Error() != nil {
				return 0, 0, iterator.Error()
			}
			break
		}
		if filterKeyValue(iterator.Key(), iterator.Value(), IsTimerQueueKey) {
			queueCount = queueCount + 1
		} else if filterKeyValue(iterator.Key(), iterator.Value(), IsTimerStoppedQueueKey) {
			stoppedQueueCount = stoppedQueueCount + 1
		}
	}
	return queueCount, stoppedQueueCount, nil
}
