package abi

import (
	"bytes"
	"github.com/vitelabs/go-vite/common/helper"
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/vm/abi"
	"math/big"
	"strings"
)

const (
	jsonTimer = `
	[
		{"type":"function","name":"NewTask", "inputs":[{"name":"taskType","type":"uint64"},{"name":"start","type":"uint64"},{"name":"window","type":"uint64"},{"name":"gap","type":"uint64"},{"name":"endCondition","type":"uint64"},{"name":"receiverAddress","type":"address"},{"name":"refundAddress","type":"address"}]},
		{"type":"function","name":"DeleteTask", "inputs":[{"name":"taskId","type":"bytes32"},{"name":"refundAddress","type":"address"}]},
		{"type":"function","name":"Recharge", "inputs":[{"name":"taskId","type":"bytes32"}]},
		{"type":"function","name":"UpdateOwner", "inputs":[{"name":"owner","type":"address"}]},
		{"type":"variable","name":"test", "inputs":[{"name":"taskId","type":"bytes32"}]},
		{"type":"variable","name":"taskInfo","inputs":[{"name":"taskId","type":"bytes32"},{"name":"taskType","type":"uint64"},{"name":"window","type":"uint64"},{"name":"gap","type":"uint64"},{"name":"endCondition","type":"uint64"},{"name":"receiverAddress","type":"address"},{"name":"refundAddress","type":"address"}]},
		{"type":"variable","name":"taskTriggerInfo","inputs":[{"name":"balance","type":"uint256"},{"name":"triggerTimes","type":"uint64"},{"name":"next","type":"uint64"},{"name":"delete","type":"uint64"}]},
		{"type":"variable","name":"lastTriggerInfo","inputs":[{"name":"timestamp","type":"uint64"},{"name":"height","type":"uint64"}]}
	]`

	jsonTimerNotify = `
	[
		{"type":"function","name":"__notify", "inputs":[{"name":"current","type":"uint64"},{"name":"taskId","type":"bytes32"}]}
	]
	`

	MethodNameTimerNewTask           = "NewTask"
	MethodNameTimerDeleteTask        = "DeleteTask"
	MethodNameTimerRecharge          = "Recharge"
	MethodNameTimerUpdateOwner       = "UpdateOwner"
	MethodNameTimerNotify            = "__notify"
	VariableNameTimerTaskInfo        = "taskInfo"
	VariableNameTimerTaskTriggerInfo = "taskTriggerInfo"
	VariableNameTimerLastTriggerInfo = "lastTriggerInfo"
)
const (
	TimerTimeHeightTime uint8 = iota + 1
	TimerTimeHeightHeight
)
const (
	TimerEndTypeEndTimeHeight uint8 = iota + 1
	TimerEndTypeTimes
	TimerEndTypePermanent
)
const (
	TimerGapTypePostpone uint8 = iota + 1
	TimerGapTypeFixed
)
const (
	TimerChargeTypeFree uint8 = iota + 1
	TimerChargeTypeCharge
)
const (
	timerIdLen              = types.AddressSize + 8
	timerQueueKeyLen        = 2 + 1 + 8 + 8
	timerStoppedQueueKeyLen = 2 + 8 + 8
)

/*
 taskInfo: 31 byte, 2 byte prefix + 21 byte address + 8 byte id
 taskTriggerInfo: 31 byte, 2 byte prefix + 21 byte address + 8 byte id
 queue: 19 byte, 2 byte prefix + 1 byte time height type + 8 byte next time height + 8 byte id
 stoppedQueue: 18 byte, 2 byte prefix + 8 byte delete height + 8 byte id
 taskId: 32 byte request hash
 nextId: 3 byte
 lastTriggerInfo: 3 byte
 fee: 3 byte
*/
var (
	ABITimer, _                   = abi.JSONToABIContract(strings.NewReader(jsonTimer))
	ABITimerNotify, _             = abi.JSONToABIContract(strings.NewReader(jsonTimerNotify))
	timerNextIdKey                = []byte("nid")
	timerLastTriggerInfoKey       = []byte("lti")
	timerFeeKey                   = []byte("fee")
	timerOwnerKey                 = []byte("own")
	timerTaskInfoKeyPrefix        = []byte{0, 0}
	timerTaskTriggerInfoKeyPrefix = []byte{0, 1}
	timerQueueKeyPrefix           = []byte{1, 0}
	timerStoppedQueueKeyPrefix    = []byte{1, 1}
)

type ParamTimerNewTask struct {
	TaskType        uint64
	Start           uint64
	EndCondition    uint64
	Window          uint64
	Gap             uint64
	ReceiverAddress types.Address
	RefundAddress   types.Address
}

type ParamTimerDeleteTask struct {
	TaskId        types.Hash
	RefundAddress types.Address
}

type TimerTaskInfo struct {
	TaskId          types.Hash
	TaskType        uint64
	Window          uint64
	Gap             uint64
	EndCondition    uint64
	ReceiverAddress types.Address
	RefundAddress   types.Address
}

type TimerTaskTriggerInfo struct {
	Balance      *big.Int
	TriggerTimes uint64
	Next         uint64
	Delete       uint64
}

func (t *TimerTaskTriggerInfo) IsStopped() bool {
	return t.Delete > 0
}

func (t *TimerTaskTriggerInfo) IsFinish() bool {
	return t.Delete == 0 && t.Next == 0
}

type TimerLastTriggerInfo struct {
	Timestamp uint64
	Height    uint64
}

func GetTimerId(address types.Address, id uint64) []byte {
	return helper.JoinBytes(address.Bytes(), helper.LeftPadBytes(new(big.Int).SetUint64(id).Bytes(), 8))
}
func GetOwnerFromTimerId(timerId []byte) types.Address {
	addr, _ := types.BytesToAddress(timerId[:types.AddressSize])
	return addr
}
func getIdFromTimerId(timerId []byte) []byte {
	return timerId[types.AddressSize:]
}
func IsTimerIdKey(k []byte) bool {
	return len(k) == timerIdLen
}
func GetTimerNextIdKey() []byte {
	return timerNextIdKey
}
func GetTimerTaskTypeDetail(taskType uint64) (timeHeight, endType, gapType uint8) {
	ct := taskType / 1000
	th := (taskType - ct*1000) / 100
	et := (taskType - ct*1000 - th*100) / 10
	gt := taskType - ct*1000 - th*100 - et*10
	return uint8(th), uint8(et), uint8(gt)
}
func GetChargeTypeFromTaskType(taskType uint64) uint8 {
	return uint8(taskType / 1000)
}
func GetVariableTaskTypeByParamTaskType(taskType uint64, isOwner bool) uint64 {
	if isOwner {
		return uint64(TimerChargeTypeFree)*1000 + taskType
	}
	return uint64(TimerChargeTypeCharge)*1000 + taskType
}
func GenerateTimerTaskType(timeHeight, endType, gapType uint8) uint64 {
	return uint64(gapType) + uint64(endType)*10 + uint64(timeHeight)*100
}

func GetTimerQueueKey(timeHeight uint8, timerId []byte, next uint64) []byte {
	return helper.JoinBytes(timerQueueKeyPrefix, []byte{byte(timeHeight)}, helper.LeftPadBytes(new(big.Int).SetUint64(next).Bytes(), 8), getIdFromTimerId(timerId))
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
func GetTimerStoppedQueueKey(timerId []byte, delete uint64) []byte {
	return helper.JoinBytes(timerStoppedQueueKeyPrefix, helper.LeftPadBytes(new(big.Int).SetUint64(delete).Bytes(), 8), getIdFromTimerId(timerId))
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
func GetTimerTaskInfoKey(timerId []byte) []byte {
	return helper.JoinBytes(timerTaskInfoKeyPrefix, timerId)
}
func GetTimerTaskTriggerInfoKey(timerId []byte) []byte {
	return helper.JoinBytes(timerTaskTriggerInfoKeyPrefix, timerId)
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

func GetTaskInfoByTimerId(db StorageDatabase, timerId []byte) ([]byte, *TimerTaskInfo, error) {
	taskInfoKey := GetTimerTaskInfoKey(timerId)
	taskInfoValue, err := db.GetValue(taskInfoKey)
	if err != nil {
		return nil, nil, err
	}
	taskInfo := new(TimerTaskInfo)
	ABITimer.UnpackVariable(taskInfo, VariableNameTimerTaskInfo, taskInfoValue)
	return taskInfoKey, taskInfo, nil
}

func GetTaskTriggerInfoByTimerId(db StorageDatabase, timerId []byte) ([]byte, *TimerTaskTriggerInfo, error) {
	taskTriggerInfoKey := GetTimerTaskTriggerInfoKey(timerId)
	taskTriggerInfoValue, err := db.GetValue(taskTriggerInfoKey)
	if err != nil {
		return nil, nil, err
	}
	taskTriggerInfo := new(TimerTaskTriggerInfo)
	ABITimer.UnpackVariable(taskTriggerInfo, VariableNameTimerTaskTriggerInfo, taskTriggerInfoValue)
	return taskTriggerInfoKey, taskTriggerInfo, nil
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
