package abi

import (
	"github.com/vitelabs/go-vite/common/helper"
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/vm/abi"
	"math/big"
	"strings"
)

const (
	jsonTimer = `
	[
		{"type":"function","name":"NewTask", "inputs":[{"name":"taskType","type":"uint64"},{"name":"start","type":"uint64"},{"name":"gap","type":"uint64"},{"name":"endCondition","type":"uint64"},{"name":"receiverAddress","type":"address"},{"name":"bid","type":"uint64"}]},
		{"type":"function","name":"DeleteTask", "inputs":[{"name":"taskId","type":"uint256"}]},
		{"type":"function","name":"Recharge", "inputs":[{"name":"taskId","type":"uint256"}]},
		{"type":"variable","name":"taskInfo","inputs":[{"name":"taskType","type":"uint64"},{"name":"gap","type":"uint64"},{"name":"receiverAddress","type":"address"},{"name":"bid","type":"uint64"}]},
		{"type":"variable","name":"taskTriggerInfo","inputs":[{"name":"balance","type":"uint256"},{"name":"next","type":"uint64"},{"name":"endCondition","type":"uint64"}]},
		{"type":"variable","name":"nextId","inputs":[{"name":"id","type":"uint64"}]},
		{"type":"variable","name":"queue","inputs":[{"name":"taskId","type":"uint256"}]},
		{"type":"variable","name":"lastTriggerInfo","inputs":[{"name":"timestamp","type":"uint64"},{"name":"height","type":"uint64"}]},
	]`

	MethodNameTimerNewTask           = "NewTask"
	MethodNameTimerDeleteTask        = "DeleteTask"
	MethodNameTimerRecharge          = "Recharge"
	VariableNameTimerTaskInfo        = "taskInfo"
	VariableNameTimerTaskTriggerInfo = "taskTriggerInfo"
	VariableNameTimerNextId          = "nextId"
	VariableNameTimerQueue           = "queue"
	VariableNameTimerLastTriggerInfo = "lastTriggerInfo"

	TimeHeightTime uint64 = iota + 1
	TimeHeightHeight
	EndTypeOnce uint64 = iota + 1
	EndTypeEndTimeHeight
	EndTypeTimes
	EndTypePermanent
	SkipTypeImmediate uint64 = iota + 1
	SkipTypeSkip
	TriggerTypeFixed uint64 = iota + 1
	TriggerTypeVariable
)

var (
	ABITimer, _    = abi.JSONToABIContract(strings.NewReader(jsonTimer))
	timerTaskIdLen = types.AddressSize + 8
	timerNextIdKey = []byte("nextId")
)

type ParamTimerNewTask struct {
	TaskType        uint64
	Start           uint64
	EndCondition    uint64
	Gap             uint64
	ReceiverAddress types.Address
	Bid             uint64
}

type TimerTaskInfo struct {
	Id              uint64
	TaskType        uint64
	Gap             uint64
	ReceiverAddress types.Address
	Bid             uint64
}

type TimerTaskTriggerInfo struct {
	Balance      *big.Int
	Next         uint64
	EndCondition uint64
}

type TimerLastTriggerInfo struct {
	Timestamp uint64
	Height    uint64
}

func GetTimerTaskId(address types.Address, id uint64) []byte {
	return helper.JoinBytes(address.Bytes(), helper.LeftPadBytes(new(big.Int).SetUint64(id).Bytes(), 8))
}
func GetOwnerFromTimerTaskId(id []byte) types.Address {
	addr, _ := types.BytesToAddress(id[:types.AddressSize])
	return addr
}
func IsTimerTaskIdKey(k []byte) bool {
	return len(k) == timerTaskIdLen
}
func GetTimerNextIdKey() []byte {
	return timerNextIdKey
}
func GetTimerQueueKey(taskType uint64, nextTrigger uint64) []byte {
	timeHeight, _, _, _ := GetTimerTaskTypeDetail(taskType)
	tmp := new(big.Int)
	return helper.JoinBytes(tmp.SetUint64(timeHeight).Bytes(), tmp.SetUint64(nextTrigger).Bytes())
}
func GetTimerTaskTypeDetail(taskType uint64) (timeHeight, endType, skipType, triggerType uint64) {
	timeHeight = taskType / 1000
	endType = (taskType - timeHeight*1000) / 100
	skipType = (taskType - timeHeight*1000 - endType*100) / 10
	triggerType = (taskType - timeHeight*1000 - endType*100 - skipType*10)
	return
}
func GenerateTimerTaskType(timeHeight, endType, skipType, triggerType uint64) uint64 {
	return triggerType + skipType*10 + endType*100 + timeHeight*1000
}
