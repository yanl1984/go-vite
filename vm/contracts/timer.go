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

	if block.Amount.Sign() > 0 {
		if block.Amount.Cmp(timerNewTaskFee) < 0 {
			return util.ErrInvalidMethodParam
		}
		chargeAmount := new(big.Int).Sub(block.Amount, timerNewTaskFee)
		if err = checkTimerChargeAmount(chargeAmount, block.TokenId); err != nil {
			return err
		}
	}

	timeHeight, endType, gapType := abi.GetTimerTaskTypeDetail(param.TaskType)
	if generatedTimerTaskType := abi.GenerateTimerTaskType(timeHeight, endType, gapType); generatedTimerTaskType != param.TaskType {
		return util.ErrInvalidMethodParam
	}
	if (timeHeight != abi.TimerTimeHeightTime && timeHeight != abi.TimerTimeHeightHeight) ||
		(endType != abi.TimerEndTypeEndTimeHeight && endType != abi.TimerEndTypeTimes && endType != abi.TimerEndTypePermanent) ||
		(gapType != abi.TimerGapTypePostpone && gapType != abi.TimerGapTypeFixed) {
		return util.ErrInvalidMethodParam
	}

	_, err = util.GetQuotaRatioForS(db, param.ReceiverAddress)
	if err != nil {
		return err
	}

	_, err = util.GetQuotaRatioForS(db, param.RefundAddress)
	if err != nil {
		return err
	}

	if (timeHeight == abi.TimerTimeHeightTime && (param.Gap < TimerTimeGapMin || param.Gap > TimerTimeGapMax)) ||
		(timeHeight == abi.TimerTimeHeightHeight && (param.Gap < TimerHeightGapMin || param.Gap > TimerHeightGapMax)) {
		return util.ErrInvalidMethodParam
	}

	if (timeHeight == abi.TimerTimeHeightTime && param.Window < TimerTimeWindowMin) ||
		(timeHeight == abi.TimerTimeHeightHeight && param.Window < TimerHeightWindowMin) ||
		param.Window > param.Gap {
		return util.ErrInvalidMethodParam
	}

	if (endType == abi.TimerEndTypeEndTimeHeight && (param.EndCondition <= 0 || param.EndCondition < param.Start)) ||
		(endType == abi.TimerEndTypeTimes && param.EndCondition <= 0) ||
		(endType == abi.TimerEndTypePermanent && param.EndCondition != 0) {
		return util.ErrInvalidMethodParam
	}

	block.Data, _ = abi.ABITimer.PackMethod(
		abi.MethodNameTimerNewTask,
		param.TaskType,
		param.Start,
		param.Window,
		param.Gap,
		param.EndCondition,
		param.ReceiverAddress,
		param.RefundAddress)
	return nil
}

func (p *MethodTimerNewTask) DoReceive(db vm_db.VmDb, block *ledger.AccountBlock, sendBlock *ledger.AccountBlock, vm vmEnvironment) ([]*ledger.AccountBlock, error) {
	param := new(abi.ParamTimerNewTask)
	abi.ABITimer.UnpackMethod(param, abi.MethodNameTimerNewTask, block.Data)
	timeHeight, endType, _ := abi.GetTimerTaskTypeDetail(param.TaskType)

	current := getCurrent(timeHeight, db, vm)
	next := firstTrigger(param.Start, current, param.Gap)
	if taskFinish(endType, param.EndCondition, next) {
		return nil, util.ErrInvalidMethodParam
	}

	nextId := getAndSetTimerNextId(db)
	timerId := abi.GetTimerId(sendBlock.AccountAddress, nextId)
	err := db.SetValue(sendBlock.Hash.Bytes(), timerId)
	util.DealWithErr(err)

	err = db.SetValue(abi.GetTimerQueueKey(timeHeight, timerId, next), timerId)
	util.DealWithErr(err)

	isOwner := getTimerOwner(db) == sendBlock.AccountAddress
	if (isOwner && sendBlock.Amount.Sign() > 0) || (!isOwner && sendBlock.Amount.Sign() == 0) {
		return nil, util.ErrInvalidMethodParam
	}
	if isBuiltInContract := types.IsBuiltinContractAddr(param.ReceiverAddress); (isOwner && !isBuiltInContract) || (!isOwner && isBuiltInContract) {
		return nil, util.ErrInvalidMethodParam
	}
	taskInfo, _ := abi.ABITimer.PackVariable(
		abi.VariableNameTimerTaskInfo,
		sendBlock.Hash,
		abi.GetVariableTaskTypeByParamTaskType(param.TaskType, isOwner),
		param.Window,
		param.Gap,
		param.EndCondition,
		param.ReceiverAddress,
		param.RefundAddress)
	err = db.SetValue(abi.GetTimerTaskInfoKey(timerId), taskInfo)
	util.DealWithErr(err)

	var chargeAmount *big.Int
	if isOwner {
		chargeAmount = big.NewInt(0)
	} else {
		chargeAmount = new(big.Int).Sub(sendBlock.Amount, timerNewTaskFee)
	}
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
	_, err = util.GetQuotaRatioForS(db, param.RefundAddress)
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

	taskTriggerInfo, refundBlocks := deleteTaskTriggerInfoAndRefund(db, timerId, param.RefundAddress)
	if !taskTriggerInfo.IsFinish() {
		timeHeight, _, _ := abi.GetTimerTaskTypeDetail(taskInfo.TaskType)
		if taskTriggerInfo.IsStopped() {
			db.SetValue(abi.GetTimerStoppedQueueKey(timerId, taskTriggerInfo.Delete), nil)
		} else {
			db.SetValue(abi.GetTimerQueueKey(timeHeight, timerId, taskTriggerInfo.Next), nil)
		}
	}
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
	_, taskInfo, err := abi.GetTaskInfoByTimerId(db, timerId)
	util.DealWithErr(err)
	if chargeType := abi.GetChargeTypeFromTaskType(taskInfo.TaskType); chargeType == abi.TimerChargeTypeFree {
		return nil, util.ErrInvalidMethodParam
	}

	taskTriggerInfo.Balance.Add(taskTriggerInfo.Balance, sendBlock.Amount)
	if taskTriggerInfo.IsStopped() {
		timeHeight, endType, _ := abi.GetTimerTaskTypeDetail(taskInfo.TaskType)
		taskTriggerInfo.Next = firstTrigger(taskTriggerInfo.Next, getCurrent(timeHeight, db, vm), taskInfo.Gap)
		if taskFinish(endType, taskInfo.EndCondition, taskTriggerInfo.Next) {
			return nil, util.ErrInvalidMethodParam
		}

		db.SetValue(abi.GetTimerStoppedQueueKey(timerId, taskTriggerInfo.Delete), nil)

		err = db.SetValue(abi.GetTimerQueueKey(timeHeight, timerId, taskTriggerInfo.Next), timerId)
		util.DealWithErr(err)

		taskTriggerInfo.Delete = 0
	}
	newTaskTriggerInfoValue, _ := abi.ABITimer.PackVariable(abi.VariableNameTimerTaskTriggerInfo, taskTriggerInfo.Balance, taskTriggerInfo.TriggerTimes, taskTriggerInfo.Next, taskTriggerInfo.Delete)
	err = db.SetValue(taskTriggerInfoKey, newTaskTriggerInfoValue)
	util.DealWithErr(err)
	return nil, nil
}

type MethodTimerUpdateOwner struct{}

func (p *MethodTimerUpdateOwner) GetFee(block *ledger.AccountBlock) (*big.Int, error) {
	return big.NewInt(0), nil
}

func (p *MethodTimerUpdateOwner) GetRefundData(sendBlock *ledger.AccountBlock) ([]byte, bool) {
	return []byte{}, false
}

func (p *MethodTimerUpdateOwner) GetSendQuota(data []byte, gasTable *util.GasTable) (uint64, error) {
	return gasTable.TimerRechargeGas, nil
}
func (p *MethodTimerUpdateOwner) GetReceiveQuota(gasTable *util.GasTable) uint64 {
	return 0
}

func (p *MethodTimerUpdateOwner) DoSend(db vm_db.VmDb, block *ledger.AccountBlock) error {
	if block.Amount.Sign() != 0 {
		return util.ErrInvalidMethodParam
	}
	newOwner := new(types.Address)
	err := abi.ABITimer.UnpackMethod(newOwner, abi.MethodNameTimerUpdateOwner, block.Data)
	if err != nil {
		return err
	}
	if *newOwner == block.AccountAddress {
		return util.ErrInvalidMethodParam
	}
	_, err = util.GetQuotaRatioForS(db, *newOwner)
	if err != nil {
		return err
	}
	block.Data, _ = abi.ABITimer.PackMethod(abi.MethodNameTimerUpdateOwner, *newOwner)
	return nil
}
func (p *MethodTimerUpdateOwner) DoReceive(db vm_db.VmDb, block *ledger.AccountBlock, sendBlock *ledger.AccountBlock, vm vmEnvironment) ([]*ledger.AccountBlock, error) {
	owner := getTimerOwner(db)
	if owner != sendBlock.AccountAddress {
		return nil, util.ErrInvalidMethodParam
	}
	newOwner := new(types.Address)
	abi.ABITimer.UnpackMethod(newOwner, abi.MethodNameTimerUpdateOwner, block.Data)
	err := db.SetValue(abi.GetTimerOwnerKey(), newOwner.Bytes())
	util.DealWithErr(err)
	return nil, nil
}

func ReceiveTaskTrigger(db vm_db.VmDb, sb *ledger.SnapshotBlock, vm vmEnvironment) ([]*ledger.AccountBlock, error) {
	// check current snapshot block > last snapshot block
	lastTriggerInfo, err := abi.GetTimerLastTriggerInfo(db)
	util.DealWithErr(err)
	currentHeight := sb.Height
	currentTime := uint64(sb.Timestamp.Unix())
	if lastTriggerInfo.Height >= currentHeight || lastTriggerInfo.Timestamp >= currentTime {
		panic("expired snapshot block")
	}

	taskNum := 0
	returnBlocks := make([]*ledger.AccountBlock, 0)
	//trigger by height
	taskNum, returnBlocks = trigger(db, abi.TimerTimeHeightHeight, currentHeight, currentHeight, sb, taskNum, returnBlocks)
	if taskNum >= TimerTriggerTasksNumMax {
		return returnBlocks, nil
	}
	// trigger by time
	taskNum, returnBlocks = trigger(db, abi.TimerTimeHeightTime, currentTime, currentHeight, sb, taskNum, returnBlocks)
	if taskNum >= TimerTriggerTasksNumMax {
		return returnBlocks, nil
	}

	// delete arrearage tasks for over 7 days
	taskNum, returnBlocks = deleteExpiredTask(db, currentHeight, sb, taskNum, returnBlocks)
	if taskNum >= TimerTriggerTasksNumMax {
		return returnBlocks, nil
	}

	// burn fee
	returnBlocks = burnFee(db, returnBlocks)

	// update last snapshot block
	if len(returnBlocks) > 0 {
		lastTriggerInfoValue, _ := abi.ABITimer.PackVariable(abi.VariableNameTimerLastTriggerInfo, currentTime, currentHeight)
		err = db.SetValue(abi.GetTimerLastTriggerInfoKey(), lastTriggerInfoValue)
		util.DealWithErr(err)
	}
	return returnBlocks, nil
}

func trigger(db vm_db.VmDb, timerQueueKeyPrefix uint8, current, currentHeight uint64, sb *ledger.SnapshotBlock, taskNum int, returnBlocks []*ledger.AccountBlock) (int, []*ledger.AccountBlock) {
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
			// next task not due
			break
		}

		taskNum = taskNum + 1
		err := db.SetValue(iterator.Key(), nil)
		util.DealWithErr(err)

		taskInfoKey, taskInfo, err := abi.GetTaskInfoByTimerId(db, timerId)
		util.DealWithErr(err)
		_, endType, gapType := abi.GetTimerTaskTypeDetail(taskInfo.TaskType)
		if taskFinish(endType, taskInfo.EndCondition, current) {
			// task finish
			taskTriggerInfoKey, taskTriggerInfo, err := abi.GetTaskTriggerInfoByTimerId(db, timerId)
			util.DealWithErr(err)
			refundBlocks := deleteAndRefund(db, taskInfoKey, taskTriggerInfoKey, taskInfo, taskTriggerInfo, sb)
			returnBlocks = append(returnBlocks, refundBlocks...)
			continue
		}
		_, err = util.GetQuotaRatioBySnapshotBlock(db, taskInfo.ReceiverAddress, sb)
		if err != nil {
			// receiver address deleted, delete task
			taskTriggerInfoKey, taskTriggerInfo, err := abi.GetTaskTriggerInfoByTimerId(db, timerId)
			util.DealWithErr(err)
			refundBlocks := deleteAndRefund(db, taskInfoKey, taskTriggerInfoKey, taskInfo, taskTriggerInfo, sb)
			returnBlocks = append(returnBlocks, refundBlocks...)
			continue
		}
		if next+taskInfo.Window <= current {
			// current task skipped
			next = lastTrigger(next, current, taskInfo.Gap)
			if next > current || next+taskInfo.Window <= current {
				next = next + taskInfo.Gap
				if taskFinish(endType, taskInfo.EndCondition, next) {
					// next task finish
					taskTriggerInfoKey, taskTriggerInfo, err := abi.GetTaskTriggerInfoByTimerId(db, timerId)
					util.DealWithErr(err)
					refundBlocks := deleteAndRefund(db, taskInfoKey, taskTriggerInfoKey, taskInfo, taskTriggerInfo, sb)
					returnBlocks = append(returnBlocks, refundBlocks...)
					continue
				}
				// prepare next trigger
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
		taskTriggerInfoKey, taskTriggerInfo, err := abi.GetTaskTriggerInfoByTimerId(db, timerId)
		util.DealWithErr(err)
		chargeType := abi.GetChargeTypeFromTaskType(taskInfo.TaskType)
		if chargeType == abi.TimerChargeTypeCharge {
			addFee(db, timerChargeAmountPerTask)
			taskTriggerInfo.Balance.Sub(taskTriggerInfo.Balance, timerChargeAmountPerTask)
		}

		next = nextTrigger(next, current, taskInfo.Gap, gapType)
		if (endType == abi.TimerEndTypeTimes && taskTriggerInfo.TriggerTimes+1 == taskInfo.EndCondition) ||
			taskFinish(endType, taskInfo.EndCondition, next) {
			// task finish, delete all
			refundBlocks := deleteAndRefund(db, taskInfoKey, taskTriggerInfoKey, taskInfo, taskTriggerInfo, sb)
			returnBlocks = append(returnBlocks, refundBlocks...)
			continue
		}

		// prepare next trigger
		if chargeType == abi.TimerChargeTypeFree || taskTriggerInfo.Balance.Sign() > 0 {
			err := db.SetValue(abi.GetTimerNewQueueKey(iterator.Key(), next), timerId)
			util.DealWithErr(err)
		} else {
			// arrearage task, put into stopped queue
			err := db.SetValue(abi.GetTimerNewStoppedQueueKey(iterator.Key(), next), timerId)
			util.DealWithErr(err)
			taskTriggerInfo.Delete = currentHeight + TimerArrearageDeleteHeight
		}
		taskTriggerInfoValue, _ := abi.ABITimer.PackVariable(abi.VariableNameTimerTaskTriggerInfo, taskTriggerInfo.Balance, taskTriggerInfo.TriggerTimes+1, next, taskTriggerInfo.Delete)
		err = db.SetValue(taskTriggerInfoKey, taskTriggerInfoValue)
		util.DealWithErr(err)
	}
	return taskNum, returnBlocks
}

func deleteExpiredTask(db vm_db.VmDb, currentHeight uint64, sb *ledger.SnapshotBlock, taskNum int, returnBlocks []*ledger.AccountBlock) (int, []*ledger.AccountBlock) {
	iterator, err := db.NewStorageIterator(abi.GetTimerStoppedQueueKeyPrefix())
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
		if !abi.IsTimerStoppedQueueKey(iterator.Key()) {
			continue
		}
		deleteHeight := abi.GetDeleteHeightFromTimerStoppedQueueKey(iterator.Key())
		if deleteHeight > currentHeight {
			// next arrearage task not due
			break
		}

		taskNum = taskNum + 1
		timerId := iterator.Value()
		err := db.SetValue(iterator.Key(), nil)
		util.DealWithErr(err)
		taskInfoKey, taskInfo, err := abi.GetTaskInfoByTimerId(db, timerId)
		util.DealWithErr(err)

		deleteAndRefund(db, taskInfoKey, abi.GetTimerTaskTriggerInfoKey(timerId), taskInfo, nil, sb)
	}
	return taskNum, returnBlocks
}

func deleteAndRefund(db vm_db.VmDb, taskInfoKey, taskTriggerInfoKey []byte, taskInfo *abi.TimerTaskInfo, taskTriggerInfo *abi.TimerTaskTriggerInfo, sb *ledger.SnapshotBlock) []*ledger.AccountBlock {
	err := db.SetValue(taskInfo.TaskId.Bytes(), nil)
	util.DealWithErr(err)
	err = db.SetValue(taskInfoKey, nil)
	util.DealWithErr(err)
	err = db.SetValue(taskTriggerInfoKey, nil)
	util.DealWithErr(err)
	_, err = util.GetQuotaRatioBySnapshotBlock(db, taskInfo.RefundAddress, sb)
	if err != nil {
		return refundTask(taskTriggerInfo, taskInfo.RefundAddress)
	}
	return nil
}

func burnFee(db vm_db.VmDb, returnBlocks []*ledger.AccountBlock) []*ledger.AccountBlock {
	feeKey, feeAmount := getFee(db)
	if feeAmount.Cmp(timerBurnFeeMin) >= 0 {
		burnData, _ := abi.ABIMintage.PackMethod(abi.MethodNameBurn)
		returnBlocks = append(returnBlocks, &ledger.AccountBlock{
			AccountAddress: types.AddressTimer,
			ToAddress:      types.AddressMintage,
			BlockType:      ledger.BlockTypeSendCall,
			Amount:         feeAmount,
			TokenId:        ledger.ViteTokenId,
			Data:           burnData,
		})
	}
	err := db.SetValue(feeKey, nil)
	util.DealWithErr(err)
	return returnBlocks
}

func addFee(db vm_db.VmDb, amount *big.Int) {
	key, feeAmount := getFee(db)
	feeAmount.Add(feeAmount, amount)
	db.SetValue(key, feeAmount.Bytes())
}

func getFee(db vm_db.VmDb) ([]byte, *big.Int) {
	key := abi.GetTimerFeeKey()
	value, err := db.GetValue(key)
	util.DealWithErr(err)
	return key, new(big.Int).SetBytes(value)
}

func getCurrent(timeHeight uint8, db vm_db.VmDb, vm vmEnvironment) uint64 {
	currentSb := vm.GlobalStatus().SnapshotBlock()
	lastTriggerInfo, err := abi.GetTimerLastTriggerInfo(db)
	util.DealWithErr(err)
	var current uint64
	if timeHeight == abi.TimerTimeHeightHeight {
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

func nextTrigger(last, current, gap uint64, gapType uint8) uint64 {
	if gapType == abi.TimerGapTypeFixed {
		return last + (current-last+gap-1)/gap*gap
	} else {
		return current + gap
	}
}

func lastTrigger(start, current, gap uint64) uint64 {
	return start + (current-start+gap-1)/gap*gap - gap
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

func deleteTaskTriggerInfoAndRefund(db vm_db.VmDb, timerId []byte, refundAddr types.Address) (*abi.TimerTaskTriggerInfo, []*ledger.AccountBlock) {
	taskTriggerInfoKey, taskTriggerInfo, err := abi.GetTaskTriggerInfoByTimerId(db, timerId)
	util.DealWithErr(err)
	err = db.SetValue(taskTriggerInfoKey, nil)
	util.DealWithErr(err)
	return taskTriggerInfo, refundTask(taskTriggerInfo, refundAddr)
}

func refundTask(taskTriggerInfo *abi.TimerTaskTriggerInfo, refundAddr types.Address) []*ledger.AccountBlock {
	if taskTriggerInfo != nil && taskTriggerInfo.Balance.Sign() > 0 {
		return []*ledger.AccountBlock{
			{
				AccountAddress: types.AddressTimer,
				ToAddress:      refundAddr,
				BlockType:      ledger.BlockTypeSendCall,
				Amount:         taskTriggerInfo.Balance,
				TokenId:        ledger.ViteTokenId,
			},
		}
	}
	return nil
}

func getTimerOwner(db vm_db.VmDb) types.Address {
	value, err := db.GetValue(abi.GetTimerOwnerKey())
	util.DealWithErr(err)
	if len(value) > 0 {
		owner, _ := types.BytesToAddress(value)
		return owner
	}
	return nodeConfig.params.TimerOwnerAddressDefault
}

func taskFinish(endType uint8, endCondition, current uint64) bool {
	if endType == abi.TimerEndTypeEndTimeHeight &&
		current > endCondition {
		return true
	}
	return false
}
