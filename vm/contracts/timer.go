package contracts

import (
	"github.com/vitelabs/go-vite/common/helper"
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/ledger"
	"github.com/vitelabs/go-vite/vm/contracts/abi"
	"github.com/vitelabs/go-vite/vm/util"
	"github.com/vitelabs/go-vite/vm_db"
	"math/big"
)

type MethodTimerNewTask struct{}

func (p *MethodTimerNewTask) GetFee(block *ledger.AccountBlock) (*big.Int, error) {
	return big.NewInt(0), nil
}

func (p *MethodTimerNewTask) GetRefundData(sendBlock *ledger.AccountBlock) ([]byte, bool) {
	return []byte{}, false
}

func (p *MethodTimerNewTask) GetSendQuota(data []byte, gasTable *util.GasTable) (uint64, error) {
	return gasTable.TimerNewTaskGas, nil
}
func (p *MethodTimerNewTask) GetReceiveQuota(gasTable *util.GasTable) uint64 {
	return 0
}

func (p *MethodTimerNewTask) DoSend(db vm_db.VmDb, block *ledger.AccountBlock) error {
	param := new(abi.ParamTimerNewTask)
	err := abi.ABITimer.UnpackMethod(param, abi.MethodNameTimerNewTask, block.Data)
	if err != nil {
		return err
	}

	if block.Amount.Cmp(timerNewTaskFee) < 0 {
		return util.ErrInvalidMethodParam
	}
	chargeAmount := new(big.Int).Sub(block.Amount, timerNewTaskFee)
	if err = checkTimerChargeAmount(chargeAmount, block.TokenId); err != nil {
		return err
	}

	timeHeight, endType, gapType := abi.GetTimerTaskTypeDetail(param.TaskType)
	if generatedTimerTaskType := abi.GenerateTimerTaskType(timeHeight, endType, gapType); generatedTimerTaskType != param.TaskType {
		return util.ErrInvalidMethodParam
	}
	if (timeHeight != abi.TimeHeightTime && timeHeight != abi.TimeHeightHeight) ||
		(endType != abi.EndTypeEndTimeHeight && endType != abi.EndTypeTimes && endType != abi.EndTypePermanent) ||
		(gapType != abi.GapTypePostpone && gapType != abi.GapTypeFixed) {
		return util.ErrInvalidMethodParam
	}

	_, err = util.GetQuotaRatioForS(db, param.ReceiverAddress)
	if err != nil {
		return err
	}

	if (timeHeight == abi.TimeHeightTime && (param.Gap < TimerTimeGapMin || param.Gap > TimerTimeGapMax)) ||
		(timeHeight == abi.TimeHeightHeight && (param.Gap < TimerHeightGapMin || param.Gap > TimerHeightGapMax)) {
		return util.ErrInvalidMethodParam
	}

	if (timeHeight == abi.TimeHeightTime && param.Window < TimerTimeWindowMin) ||
		(timeHeight == abi.TimeHeightHeight && param.Window < TimerHeightWindowMin) ||
		param.Window > param.Gap {
		return util.ErrInvalidMethodParam
	}

	if (endType == abi.EndTypeEndTimeHeight && (param.EndCondition <= 0 || param.EndCondition < param.Start)) ||
		(endType == abi.EndTypeTimes && param.EndCondition <= 0) ||
		(endType == abi.EndTypePermanent && param.EndCondition != 0) {
		return util.ErrInvalidMethodParam
	}

	block.Data, _ = abi.ABIMintage.PackMethod(
		abi.MethodNameTimerNewTask,
		param.TaskType,
		param.Start,
		param.Window,
		param.Gap,
		param.EndCondition,
		param.ReceiverAddress)
	return nil
}

func (p *MethodTimerNewTask) DoReceive(db vm_db.VmDb, block *ledger.AccountBlock, sendBlock *ledger.AccountBlock, vm vmEnvironment) ([]*ledger.AccountBlock, error) {
	param := new(abi.ParamTimerNewTask)
	abi.ABITimer.UnpackMethod(param, abi.MethodNameTimerNewTask, block.Data)
	timeHeight, endType, _ := abi.GetTimerTaskTypeDetail(param.TaskType)

	current := getCurrent(timeHeight, db, vm)
	next := firstTrigger(param.Start, current, param.Gap)
	if endType == abi.EndTypeEndTimeHeight &&
		next > param.EndCondition {
		return nil, util.ErrInvalidMethodParam
	}

	nextId := getAndSetTimerNextId(db)
	timerId := abi.GetTimerId(sendBlock.AccountAddress, nextId)
	err := db.SetValue(sendBlock.Hash.Bytes(), timerId)
	util.DealWithErr(err)

	err = db.SetValue(abi.GetTimerQueueKey(timeHeight, timerId, next), timerId)
	util.DealWithErr(err)

	taskInfo, _ := abi.ABITimer.PackVariable(abi.VariableNameTimerTaskInfo, sendBlock.Hash, param.TaskType, param.Window, param.Gap, param.EndCondition, param.ReceiverAddress)
	err = db.SetValue(abi.GetTimerTaskInfoKey(timerId), taskInfo)
	util.DealWithErr(err)

	chargeAmount := new(big.Int).Sub(sendBlock.Amount, timerNewTaskFee)
	taskTriggerInfo, _ := abi.ABITimer.PackVariable(abi.VariableNameTimerTaskTriggerInfo, chargeAmount, uint64(0), next, uint64(0))
	err = db.SetValue(abi.GetTimerTaskTriggerInfoKey(timerId), taskTriggerInfo)
	util.DealWithErr(err)

	addFee(db, timerNewTaskFee)
	return nil, nil
}

type MethodTimerDeleteTask struct{}

func (p *MethodTimerDeleteTask) GetFee(block *ledger.AccountBlock) (*big.Int, error) {
	return big.NewInt(0), nil
}

func (p *MethodTimerDeleteTask) GetRefundData(sendBlock *ledger.AccountBlock) ([]byte, bool) {
	return []byte{}, false
}

func (p *MethodTimerDeleteTask) GetSendQuota(data []byte, gasTable *util.GasTable) (uint64, error) {
	return gasTable.TimerDeleteTaskGas, nil
}
func (p *MethodTimerDeleteTask) GetReceiveQuota(gasTable *util.GasTable) uint64 {
	return 0
}

func (p *MethodTimerDeleteTask) DoSend(db vm_db.VmDb, block *ledger.AccountBlock) error {
	if block.Amount.Sign() > 0 {
		return util.ErrInvalidMethodParam
	}
	param := new(abi.ParamTimerDeleteTask)
	err := abi.ABITimer.UnpackMethod(param, abi.MethodNameTimerDeleteTask, block.Data)
	if err != nil {
		return err
	}
	block.Data, _ = abi.ABITimer.PackMethod(abi.MethodNameTimerDeleteTask, param.TaskId, param.RefundAddress)
	return nil
}
func (p *MethodTimerDeleteTask) DoReceive(db vm_db.VmDb, block *ledger.AccountBlock, sendBlock *ledger.AccountBlock, vm vmEnvironment) ([]*ledger.AccountBlock, error) {
	param := new(abi.ParamTimerDeleteTask)
	abi.ABITimer.UnpackMethod(param, abi.MethodNameTimerDeleteTask, block.Data)
	timerId, err := db.GetValue(param.TaskId.Bytes())
	util.DealWithErr(err)
	if len(timerId) == 0 {
		return nil, util.ErrInvalidMethodParam
	}
	if abi.GetOwnerFromTimerId(timerId) != sendBlock.AccountAddress {
		return nil, util.ErrInvalidMethodParam
	}

	err = db.SetValue(param.TaskId.Bytes(), nil)
	util.DealWithErr(err)

	taskInfoKey, taskInfo, err := abi.GetTaskInfoByTimerId(db, timerId)
	util.DealWithErr(err)
	err = db.SetValue(taskInfoKey, nil)
	util.DealWithErr(err)

	taskTriggerInfo, refundBlocks := deleteTaskTriggerInfoAndRefund(db, block, sendBlock, timerId, param.RefundAddress)

	timeHeight, _, _ := abi.GetTimerTaskTypeDetail(taskInfo.TaskType)
	db.SetValue(abi.GetTimerQueueKey(timeHeight, timerId, taskTriggerInfo.Next), nil)
	db.SetValue(abi.GetTimerStoppedQueueKey(timeHeight, timerId, taskTriggerInfo.Delete), nil)
	return refundBlocks, nil
}

type MethodTimerRecharge struct{}

func (p *MethodTimerRecharge) GetFee(block *ledger.AccountBlock) (*big.Int, error) {
	return big.NewInt(0), nil
}

func (p *MethodTimerRecharge) GetRefundData(sendBlock *ledger.AccountBlock) ([]byte, bool) {
	return []byte{}, false
}

func (p *MethodTimerRecharge) GetSendQuota(data []byte, gasTable *util.GasTable) (uint64, error) {
	return gasTable.TimerRechargeGas, nil
}
func (p *MethodTimerRecharge) GetReceiveQuota(gasTable *util.GasTable) uint64 {
	return 0
}

func (p *MethodTimerRecharge) DoSend(db vm_db.VmDb, block *ledger.AccountBlock) error {
	if err := checkTimerChargeAmount(block.Amount, block.TokenId); err != nil {
		return err
	}
	taskId := new(types.Hash)
	err := abi.ABITimer.UnpackMethod(taskId, abi.MethodNameTimerRecharge, block.Data)
	if err != nil {
		return err
	}
	block.Data, _ = abi.ABITimer.PackMethod(abi.MethodNameTimerRecharge, *taskId)
	return nil
}
func (p *MethodTimerRecharge) DoReceive(db vm_db.VmDb, block *ledger.AccountBlock, sendBlock *ledger.AccountBlock, vm vmEnvironment) ([]*ledger.AccountBlock, error) {
	taskId := new(types.Hash)
	abi.ABITimer.UnpackMethod(taskId, abi.MethodNameTimerRecharge, block.Data)
	timerId, err := db.GetValue(taskId.Bytes())
	util.DealWithErr(err)
	if len(timerId) == 0 {
		return nil, util.ErrInvalidMethodParam
	}

	taskTriggerInfoKey, taskTriggerInfo, err := abi.GetTaskTriggerInfoByTimerId(db, timerId)
	util.DealWithErr(err)
	stopped := taskTriggerInfo.Delete > 0
	taskTriggerInfo.Balance.Add(taskTriggerInfo.Balance, sendBlock.Amount)
	if stopped {
		_, taskInfo, err := abi.GetTaskInfoByTimerId(db, timerId)
		util.DealWithErr(err)
		timeHeight, _, gapType := abi.GetTimerTaskTypeDetail(taskInfo.TaskType)
		db.SetValue(abi.GetTimerStoppedQueueKey(timeHeight, timerId, taskTriggerInfo.Delete), nil)

		if gapType == abi.GapTypeFixed {
			taskTriggerInfo.Next = firstTrigger(taskTriggerInfo.Next, getCurrent(timeHeight, db, vm), taskInfo.Gap)
		} else {
			taskTriggerInfo.Next = firstTrigger(taskTriggerInfo.Next, getCurrent(timeHeight, db, vm), taskInfo.Gap)
		}

		err = db.SetValue(abi.GetTimerQueueKey(timeHeight, timerId, taskTriggerInfo.Next), timerId)
		util.DealWithErr(err)

		taskTriggerInfo.Delete = 0
	}
	newTaskTriggerInfoValue, _ := abi.ABITimer.PackVariable(abi.VariableNameTimerTaskTriggerInfo, taskTriggerInfo.Balance, taskTriggerInfo.TriggerTimes, taskTriggerInfo.Next)
	err = db.SetValue(taskTriggerInfoKey, newTaskTriggerInfoValue)
	util.DealWithErr(err)
	return nil, nil
}

func ReceiveTaskTrigger(db vm_db.VmDb, block *ledger.AccountBlock, sb *ledger.SnapshotBlock, vm vmEnvironment) ([]*ledger.AccountBlock, error) {
	lastTriggerInfo, err := abi.GetTimerLastTriggerInfo(db)
	util.DealWithErr(err)
	currentHeight := sb.Height
	currentTime := uint64(sb.Timestamp.Unix())
	if lastTriggerInfo.Height >= currentHeight || lastTriggerInfo.Timestamp >= currentTime {
		panic("expired snapshot block")
	}

	taskNum := 0
	returnBlocks := make([]*ledger.AccountBlock, 0)
	taskNum, returnBlocks = trigger(db, abi.TimeHeightHeight, currentHeight, taskNum, returnBlocks)
	if taskNum >= TimerTriggerTasksNumMax {
		return returnBlocks, nil
	}
	taskNum, returnBlocks = trigger(db, abi.TimeHeightTime, currentTime, taskNum, returnBlocks)
	if taskNum >= TimerTriggerTasksNumMax {
		return returnBlocks, nil
	}
	// TODO trigger task: if finish, delete all
	// TODO trigger task: if not finish, update taskTriggerInfo, delete queue
	// TODO trigger task: if balance > zero, add queue
	// TODO trigger task: if balance = zero, add stopped queue

	// TODO delete task, check stopped queue, if stopped, delete all

	// TODO batch destroy vite
	return nil, nil
}

func trigger(db vm_db.VmDb, timerQueueKeyPrefix uint8, current uint64, taskNum int, returnBlocks []*ledger.AccountBlock) (int, []*ledger.AccountBlock) {
	iterator, err := db.NewStorageIterator(abi.GetTimerQueueKeyPrefix(timerQueueKeyPrefix))
	util.DealWithErr(err)
	defer iterator.Release()
	for {
		if taskNum >= TimerTriggerTasksNumMax {
			break
		}
		if !iterator.Next() {
			if iterator.Error() != nil {
				util.DealWithErr(iterator.Error())
			}
			break
		}
		if !abi.IsTimerQueueKey(iterator.Key()) {
			continue
		}
		timerId := iterator.Value()
		next := abi.GetNextTriggerFromTimerQueueKey(iterator.Key())
		if next > current {
			// task not due
			break
		}

		taskNum = taskNum + 1
		err := db.SetValue(iterator.Key(), nil)
		util.DealWithErr(err)

		_, taskInfo, err := abi.GetTaskInfoByTimerId(db, timerId)
		util.DealWithErr(err)
		if next+taskInfo.Window <= current {
			// skip current task
			next = lastTrigger(next, current, taskInfo.Gap)
			if next > current || next+taskInfo.Window <= current {
				// prepare next trigger
				next = next + taskInfo.Gap
				err = db.SetValue(abi.GetTimerNewQueueKey(iterator.Key(), next), iterator.Value())
				continue
			}
		}
		// trigger task
		data, _ := abi.ABITimerNotify.PackMethod(abi.MethodNameTimerNotify, current, taskInfo.TaskId)
		returnBlocks = append(returnBlocks, &ledger.AccountBlock{
			AccountAddress: types.AddressTimer,
			ToAddress:      taskInfo.ReceiverAddress,
			BlockType:      ledger.BlockTypeSendCall,
			Amount:         big.NewInt(0),
			TokenId:        ledger.ViteTokenId,
			Data:           data,
		})
		addFee(db, timerChargeAmountPerTask)

		taskTriggerInfoKey, taskTriggerInfo, err := abi.GetTaskTriggerInfoByTimerId(db, timerId)
		util.DealWithErr(err)
		_, endType, _ := abi.GetTimerTaskTypeDetail(taskInfo.TaskType)
		next = nextTrigger(next, current, taskInfo.Gap)
		if (endType == abi.EndTypeTimes && taskTriggerInfo.TriggerTimes+1 == taskInfo.EndCondition) ||
			(endType == abi.EndTypeEndTimeHeight && (current >= taskInfo.EndCondition || next > taskInfo.EndCondition)) {
			// TODO task finish, delete all

			continue
		}
		// prepare next trigger
		taskTriggerInfo.Balance.Sub(taskTriggerInfo.Balance, timerChargeAmountPerTask)
		taskTriggerInfoValue, _ := abi.ABITimer.PackVariable(abi.VariableNameTimerTaskTriggerInfo, taskTriggerInfo.Balance, taskTriggerInfo.TriggerTimes+1, next)
		err = db.SetValue(taskTriggerInfoKey, taskTriggerInfoValue)
		util.DealWithErr(err)

		if taskTriggerInfo.Balance.Sign() > 0 {
			err := db.SetValue(abi.GetTimerNewQueueKey(iterator.Key(), next), timerId)
			util.DealWithErr(err)
		} else {
			err := db.SetValue(abi.GetTimerNewStoppedQueueKey(iterator.Key(), next), timerId)
			util.DealWithErr(err)
		}
	}
	return taskNum, returnBlocks
}

func addFee(db vm_db.VmDb, amount *big.Int) {
	value, err := db.GetValue(abi.GetTimerFeeKey())
	util.DealWithErr(err)
	feeAmount := new(big.Int).SetBytes(value)
	feeAmount.Add(feeAmount, amount)
	db.SetValue(abi.GetTimerFeeKey(), feeAmount.Bytes())
}

func getCurrent(timeHeight uint8, db vm_db.VmDb, vm vmEnvironment) uint64 {
	currentSb := vm.GlobalStatus().SnapshotBlock()
	lastTriggerInfo, err := abi.GetTimerLastTriggerInfo(db)
	util.DealWithErr(err)
	var current uint64
	if timeHeight == abi.TimeHeightHeight {
		current = currentSb.Height
		if lastTriggerInfo != nil {
			current = helper.Max(current, lastTriggerInfo.Height)
		}
	} else {
		current = uint64(currentSb.Timestamp.Unix())
		if lastTriggerInfo != nil {
			current = helper.Max(current, lastTriggerInfo.Timestamp)
		}
	}
	return current
}

func firstTrigger(start, current, gap uint64) uint64 {
	if start > current {
		return start
	}
	return start + (current-start+gap-1)/gap*gap
}

func nextTrigger(start, current, gap uint64) uint64 {
	if start >= current {
		return start
	}
	return start + (current-start+gap-1)/gap*gap
}

func lastTrigger(start, current, gap uint64) uint64 {
	return nextTrigger(start, current, gap) - gap
}

func checkTimerChargeAmount(amount *big.Int, tokenId types.TokenTypeId) error {
	if tokenId != ledger.ViteTokenId || amount.Cmp(timerChargeAmountPerTask) < 0 || new(big.Int).Mod(amount, timerChargeAmountPerTask).Sign() != 0 {
		return util.ErrInvalidMethodParam
	}
	return nil
}

func getAndSetTimerNextId(db vm_db.VmDb) uint64 {
	timerNextIdKey := abi.GetTimerNextIdKey()
	value, err := db.GetValue(timerNextIdKey)
	util.DealWithErr(err)
	nextId := uint64(1)
	nextIdBig := big.NewInt(1)
	if len(value) > 0 {
		nextId = nextIdBig.SetBytes(value).Uint64()
	}
	nextIdBig.Add(nextIdBig, helper.Big1)
	err = db.SetValue(timerNextIdKey, nextIdBig.Bytes())
	util.DealWithErr(err)
	return nextId
}

func deleteTaskTriggerInfoAndRefund(db vm_db.VmDb, block *ledger.AccountBlock, sendBlock *ledger.AccountBlock, timerId []byte, refundAddr types.Address) (*abi.TimerTaskTriggerInfo, []*ledger.AccountBlock) {
	taskTriggerInfoKey, taskTriggerInfo, err := abi.GetTaskTriggerInfoByTimerId(db, timerId)
	util.DealWithErr(err)
	err = db.SetValue(taskTriggerInfoKey, nil)
	util.DealWithErr(err)

	if taskTriggerInfo.Balance.Sign() > 0 {
		return taskTriggerInfo, []*ledger.AccountBlock{
			{
				AccountAddress: block.AccountAddress,
				ToAddress:      refundAddr,
				BlockType:      ledger.BlockTypeSendCall,
				Amount:         taskTriggerInfo.Balance,
				TokenId:        ledger.ViteTokenId,
			},
		}
	}
	return taskTriggerInfo, nil
}

func getTaskStop(info *abi.TimerTaskInfo, triggerInfo *abi.TimerTaskTriggerInfo, endType uint8) uint64 {
	if endType == abi.EndTypeEndTimeHeight {
		return info.EndCondition
	} else if endType == abi.EndTypeTimes {
		return triggerInfo.Next + (info.EndCondition-triggerInfo.TriggerTimes-1)*info.Gap
	} else if endType == abi.EndTypePermanent {
		return helper.MaxUint64
	}
	panic(util.ErrInvalidMethodParam)
}
