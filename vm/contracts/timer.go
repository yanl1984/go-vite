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

type MethodTimerNewTimer struct{}

func (p *MethodTimerNewTimer) GetFee(block *ledger.AccountBlock) (*big.Int, error) {
	return big.NewInt(0), nil
}

func (p *MethodTimerNewTimer) GetRefundData(sendBlock *ledger.AccountBlock) ([]byte, bool) {
	return []byte{}, false
}

func (p *MethodTimerNewTimer) GetSendQuota(data []byte, gasTable *util.GasTable) (uint64, error) {
	return gasTable.TimerNewTimerGas, nil
}
func (p *MethodTimerNewTimer) GetReceiveQuota(gasTable *util.GasTable) uint64 {
	return 0
}

func (p *MethodTimerNewTimer) DoSend(db vm_db.VmDb, block *ledger.AccountBlock) error {
	param := new(abi.ParamNewTimer)
	err := abi.ABITimer.UnpackMethod(param, abi.MethodNameTimerNewTimer, block.Data)
	if err != nil {
		return err
	}

	if block.Amount.Sign() > 0 {
		if block.Amount.Cmp(newTimerFee) < 0 {
			return util.ErrInvalidMethodParam
		}
		chargeAmount := new(big.Int).Sub(block.Amount, newTimerFee)
		if err = checkTimerChargeAmount(chargeAmount, block.TokenId); err != nil {
			return err
		}
	}

	timeHeight, expiringType, intervalType := abi.GetTimerTypeDetail(param.TimerType)
	if generatedTimerType := abi.GenerateTimerType(timeHeight, expiringType, intervalType); generatedTimerType != param.TimerType {
		return util.ErrInvalidMethodParam
	}
	if (timeHeight != abi.TimerTimeHeightTime && timeHeight != abi.TimerTimeHeightHeight) ||
		(expiringType != abi.TimerExpiringTypeTimeHeight && expiringType != abi.TimerExpiringTypeTimes && expiringType != abi.TimerExpiringTypePermanent) ||
		(intervalType != abi.TimerIntervalTypePostpone && intervalType != abi.TimerIntervalTypeFixed) {
		return util.ErrInvalidMethodParam
	}

	_, err = util.GetQuotaRatioForS(db, param.InvokingAddr)
	if err != nil {
		return err
	}

	_, err = util.GetQuotaRatioForS(db, param.RefundAddr)
	if err != nil {
		return err
	}

	if (timeHeight == abi.TimerTimeHeightTime && (param.Interval < TimerTimeIntervalMin || param.Interval > TimerTimeIntervalMax)) ||
		(timeHeight == abi.TimerTimeHeightHeight && (param.Interval < TimerHeightIntervalMin || param.Interval > TimerHeightIntervalMax)) {
		return util.ErrInvalidMethodParam
	}

	if (timeHeight == abi.TimerTimeHeightTime && param.Window < TimerTimeWindowMin) ||
		(timeHeight == abi.TimerTimeHeightHeight && param.Window < TimerHeightWindowMin) ||
		param.Window > param.Interval {
		return util.ErrInvalidMethodParam
	}

	if (expiringType == abi.TimerExpiringTypeTimeHeight && (param.ExpiringCondition <= 0 || param.ExpiringCondition < param.Start)) ||
		(expiringType == abi.TimerExpiringTypeTimes && param.ExpiringCondition <= 0) ||
		(expiringType == abi.TimerExpiringTypePermanent && param.ExpiringCondition != 0) {
		return util.ErrInvalidMethodParam
	}

	block.Data, _ = abi.ABITimer.PackMethod(
		abi.MethodNameTimerNewTimer,
		param.TimerType,
		param.Start,
		param.Window,
		param.Interval,
		param.ExpiringCondition,
		param.InvokingAddr,
		param.RefundAddr)
	return nil
}

func (p *MethodTimerNewTimer) DoReceive(db vm_db.VmDb, block *ledger.AccountBlock, sendBlock *ledger.AccountBlock, vm vmEnvironment) ([]*ledger.AccountBlock, error) {
	param := new(abi.ParamNewTimer)
	abi.ABITimer.UnpackMethod(param, abi.MethodNameTimerNewTimer, sendBlock.Data)
	timeHeight, endType, _ := abi.GetTimerTypeDetail(param.TimerType)

	current := getCurrent(timeHeight, db, vm)
	next := nextTrigger(param.Start, current, param.Interval)
	if timerFinish(endType, param.ExpiringCondition, next) {
		return nil, util.ErrInvalidMethodParam
	}

	nextId := getAndSetTimerNextId(db)
	innerId := abi.GetInnerId(sendBlock.AccountAddress, nextId)
	err := db.SetValue(abi.GetTimerIdKey(sendBlock.Hash), innerId)
	util.DealWithErr(err)

	err = db.SetValue(abi.GetTimerQueueKey(timeHeight, innerId, next), innerId)
	util.DealWithErr(err)

	isOwner := getTimerOwner(db) == sendBlock.AccountAddress
	if (isOwner && sendBlock.Amount.Sign() > 0) || (!isOwner && sendBlock.Amount.Sign() == 0) {
		return nil, util.ErrInvalidMethodParam
	}
	if isBuiltInContract := types.IsBuiltinContractAddr(param.InvokingAddr); isOwner && !isBuiltInContract {
		return nil, util.ErrInvalidMethodParam
	}
	info, _ := abi.ABITimer.PackVariable(
		abi.VariableNameTimerInfo,
		sendBlock.Hash,
		abi.GetVariableTimerTypeByParamTimerType(param.TimerType, isOwner),
		param.Window,
		param.Interval,
		param.ExpiringCondition,
		param.InvokingAddr,
		param.RefundAddr)
	err = db.SetValue(abi.GetTimerInfoKey(innerId), info)
	util.DealWithErr(err)

	var amount *big.Int
	if isOwner {
		amount = big.NewInt(0)
	} else {
		amount = new(big.Int).Sub(sendBlock.Amount, newTimerFee)
		addFee(db, newTimerFee)
	}
	triggerInfo, _ := abi.ABITimer.PackVariable(abi.VariableNameTimerTriggerInfo, amount, uint64(0), next, uint64(0))
	err = db.SetValue(abi.GetTimerTriggerInfoKey(innerId), triggerInfo)
	util.DealWithErr(err)

	return nil, nil
}

type MethodTimerDeleteTimer struct{}

func (p *MethodTimerDeleteTimer) GetFee(block *ledger.AccountBlock) (*big.Int, error) {
	return big.NewInt(0), nil
}

func (p *MethodTimerDeleteTimer) GetRefundData(sendBlock *ledger.AccountBlock) ([]byte, bool) {
	return []byte{}, false
}

func (p *MethodTimerDeleteTimer) GetSendQuota(data []byte, gasTable *util.GasTable) (uint64, error) {
	return gasTable.TimerDeleteTimerGas, nil
}
func (p *MethodTimerDeleteTimer) GetReceiveQuota(gasTable *util.GasTable) uint64 {
	return 0
}

func (p *MethodTimerDeleteTimer) DoSend(db vm_db.VmDb, block *ledger.AccountBlock) error {
	if block.Amount.Sign() > 0 {
		return util.ErrInvalidMethodParam
	}
	param := new(abi.ParamDeleteTimer)
	err := abi.ABITimer.UnpackMethod(param, abi.MethodNameTimerDeleteTimer, block.Data)
	if err != nil {
		return err
	}
	_, err = util.GetQuotaRatioForS(db, param.RefundAddr)
	if err != nil {
		return err
	}
	block.Data, _ = abi.ABITimer.PackMethod(abi.MethodNameTimerDeleteTimer, param.TimerId, param.RefundAddr)
	return nil
}
func (p *MethodTimerDeleteTimer) DoReceive(db vm_db.VmDb, block *ledger.AccountBlock, sendBlock *ledger.AccountBlock, vm vmEnvironment) ([]*ledger.AccountBlock, error) {
	param := new(abi.ParamDeleteTimer)
	abi.ABITimer.UnpackMethod(param, abi.MethodNameTimerDeleteTimer, sendBlock.Data)
	innerId, err := abi.GetInnerIdByTimerId(db, param.TimerId)
	util.DealWithErr(err)
	if len(innerId) == 0 {
		return nil, util.ErrInvalidMethodParam
	}
	if abi.GetOwnerFromInnerId(innerId) != sendBlock.AccountAddress {
		return nil, util.ErrInvalidMethodParam
	}

	err = db.SetValue(abi.GetTimerIdKey(param.TimerId), nil)
	util.DealWithErr(err)

	infoKey, info, err := abi.GetTimerInfoByInnerId(db, innerId)
	util.DealWithErr(err)
	err = db.SetValue(infoKey, nil)
	util.DealWithErr(err)

	triggerInfo, refundBlocks := deleteTimerTriggerInfoAndRefund(db, innerId, param.RefundAddr)
	if !triggerInfo.IsFinish() {
		timeHeight, _, _ := abi.GetTimerTypeDetail(info.TimerType)
		if triggerInfo.IsStopped() {
			db.SetValue(abi.GetTimerStoppedQueueKey(innerId, triggerInfo.Delete), nil)
		} else {
			db.SetValue(abi.GetTimerQueueKey(timeHeight, innerId, triggerInfo.Next), nil)
		}
	}
	return refundBlocks, nil
}

type MethodTimerDeposit struct{}

func (p *MethodTimerDeposit) GetFee(block *ledger.AccountBlock) (*big.Int, error) {
	return big.NewInt(0), nil
}

func (p *MethodTimerDeposit) GetRefundData(sendBlock *ledger.AccountBlock) ([]byte, bool) {
	return []byte{}, false
}

func (p *MethodTimerDeposit) GetSendQuota(data []byte, gasTable *util.GasTable) (uint64, error) {
	return gasTable.TimerDepositGas, nil
}
func (p *MethodTimerDeposit) GetReceiveQuota(gasTable *util.GasTable) uint64 {
	return 0
}

func (p *MethodTimerDeposit) DoSend(db vm_db.VmDb, block *ledger.AccountBlock) error {
	if err := checkTimerChargeAmount(block.Amount, block.TokenId); err != nil {
		return err
	}
	timerId := new(types.Hash)
	err := abi.ABITimer.UnpackMethod(timerId, abi.MethodNameTimerDeposit, block.Data)
	if err != nil {
		return err
	}
	block.Data, _ = abi.ABITimer.PackMethod(abi.MethodNameTimerDeposit, *timerId)
	return nil
}
func (p *MethodTimerDeposit) DoReceive(db vm_db.VmDb, block *ledger.AccountBlock, sendBlock *ledger.AccountBlock, vm vmEnvironment) ([]*ledger.AccountBlock, error) {
	timerId := new(types.Hash)
	abi.ABITimer.UnpackMethod(timerId, abi.MethodNameTimerDeposit, sendBlock.Data)
	innerId, err := abi.GetInnerIdByTimerId(db, *timerId)
	util.DealWithErr(err)
	if len(innerId) == 0 {
		return nil, util.ErrInvalidMethodParam
	}

	triggerInfoKey, triggerInfo, err := abi.GetTimerTriggerInfoByInnerId(db, innerId)
	util.DealWithErr(err)
	_, info, err := abi.GetTimerInfoByInnerId(db, innerId)
	util.DealWithErr(err)
	if chargeType := abi.GetChargeTypeFromTimerType(info.TimerType); chargeType == abi.TimerChargeTypeFree {
		return nil, util.ErrInvalidMethodParam
	}

	triggerInfo.Amount.Add(triggerInfo.Amount, sendBlock.Amount)
	if triggerInfo.IsStopped() {
		timeHeight, expiringType, _ := abi.GetTimerTypeDetail(info.TimerType)
		triggerInfo.Next = nextTrigger(triggerInfo.Next, getCurrent(timeHeight, db, vm), info.Interval)
		if timerFinish(expiringType, info.ExpiringCondition, triggerInfo.Next) {
			return nil, util.ErrInvalidMethodParam
		}

		db.SetValue(abi.GetTimerStoppedQueueKey(innerId, triggerInfo.Delete), nil)

		err = db.SetValue(abi.GetTimerQueueKey(timeHeight, innerId, triggerInfo.Next), innerId)
		util.DealWithErr(err)

		triggerInfo.Delete = 0
	}
	newtriggerInfoValue, _ := abi.ABITimer.PackVariable(abi.VariableNameTimerTriggerInfo, triggerInfo.Amount, triggerInfo.TriggerTimes, triggerInfo.Next, triggerInfo.Delete)
	err = db.SetValue(triggerInfoKey, newtriggerInfoValue)
	util.DealWithErr(err)
	return nil, nil
}

type MethodTimerTransferOwnership struct{}

func (p *MethodTimerTransferOwnership) GetFee(block *ledger.AccountBlock) (*big.Int, error) {
	return big.NewInt(0), nil
}

func (p *MethodTimerTransferOwnership) GetRefundData(sendBlock *ledger.AccountBlock) ([]byte, bool) {
	return []byte{}, false
}

func (p *MethodTimerTransferOwnership) GetSendQuota(data []byte, gasTable *util.GasTable) (uint64, error) {
	return gasTable.TimerTransferOwnership, nil
}
func (p *MethodTimerTransferOwnership) GetReceiveQuota(gasTable *util.GasTable) uint64 {
	return 0
}

func (p *MethodTimerTransferOwnership) DoSend(db vm_db.VmDb, block *ledger.AccountBlock) error {
	if block.Amount.Sign() != 0 {
		return util.ErrInvalidMethodParam
	}
	newOwner := new(types.Address)
	err := abi.ABITimer.UnpackMethod(newOwner, abi.MethodNameTimerTransferOwnership, block.Data)
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
	block.Data, _ = abi.ABITimer.PackMethod(abi.MethodNameTimerTransferOwnership, *newOwner)
	return nil
}
func (p *MethodTimerTransferOwnership) DoReceive(db vm_db.VmDb, block *ledger.AccountBlock, sendBlock *ledger.AccountBlock, vm vmEnvironment) ([]*ledger.AccountBlock, error) {
	owner := getTimerOwner(db)
	if owner != sendBlock.AccountAddress {
		return nil, util.ErrInvalidMethodParam
	}
	newOwner := new(types.Address)
	abi.ABITimer.UnpackMethod(newOwner, abi.MethodNameTimerTransferOwnership, sendBlock.Data)
	err := db.SetValue(abi.GetTimerOwnerKey(), newOwner.Bytes())
	util.DealWithErr(err)
	return nil, nil
}

func ReceiveTimerTrigger(db vm_db.VmDb, sb *ledger.SnapshotBlock, vm vmEnvironment) ([]*ledger.AccountBlock, error) {
	// check current snapshot block > last snapshot block
	lastTriggerInfo, err := abi.GetTimerLastTriggerInfo(db)
	util.DealWithErr(err)
	currentHeight := sb.Height
	currentTime := uint64(sb.Timestamp.Unix())
	if lastTriggerInfo != nil && (lastTriggerInfo.Height >= currentHeight || lastTriggerInfo.Timestamp >= currentTime) {
		return nil, util.ErrInvalidMethodParam
	}

	lastTriggerInfoValue, _ := abi.ABITimer.PackVariable(abi.VariableNameTimerLastTriggerInfo, currentTime, currentHeight)
	err = db.SetValue(abi.GetTimerLastTriggerInfoKey(), lastTriggerInfoValue)
	util.DealWithErr(err)

	num := 0
	returnBlocks := make([]*ledger.AccountBlock, 0)
	//trigger by height
	num, returnBlocks = trigger(db, abi.TimerTimeHeightHeight, currentHeight, currentHeight, sb, num, returnBlocks)
	if num >= TimerTriggerNumMax {
		return returnBlocks, nil
	}
	// trigger by time
	num, returnBlocks = trigger(db, abi.TimerTimeHeightTime, currentTime, currentHeight, sb, num, returnBlocks)
	if num >= TimerTriggerNumMax {
		return returnBlocks, nil
	}

	// delete arrearage timers for over 7 days
	num, returnBlocks = deleteExpiredTimer(db, currentHeight, sb, num, returnBlocks)
	if num >= TimerTriggerNumMax {
		return returnBlocks, nil
	}

	// burn fee
	returnBlocks = burnFee(db, returnBlocks)
	return returnBlocks, nil
}

func trigger(db vm_db.VmDb, timerQueueKeyPrefix uint8, current, currentHeight uint64, sb *ledger.SnapshotBlock, num int, returnBlocks []*ledger.AccountBlock) (int, []*ledger.AccountBlock) {
	iterator, err := db.NewStorageIterator(abi.GetTimerQueueKeyPrefix(timerQueueKeyPrefix))
	util.DealWithErr(err)
	defer iterator.Release()
	for {
		if num >= TimerTriggerNumMax {
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
		innerId := iterator.Value()
		next := abi.GetNextTriggerFromTimerQueueKey(iterator.Key())
		if next > current {
			// next timer not due
			break
		}

		num = num + 1
		err := db.SetValue(iterator.Key(), nil)
		util.DealWithErr(err)

		infoKey, info, err := abi.GetTimerInfoByInnerId(db, innerId)
		util.DealWithErr(err)
		_, expiringType, intervalType := abi.GetTimerTypeDetail(info.TimerType)
		if timerFinish(expiringType, info.ExpiringCondition, current) {
			// timer finish
			triggerInfoKey, triggerInfo, err := abi.GetTimerTriggerInfoByInnerId(db, innerId)
			util.DealWithErr(err)
			refundBlocks := deleteAndRefund(db, infoKey, triggerInfoKey, info, triggerInfo, sb)
			returnBlocks = append(returnBlocks, refundBlocks...)
			continue
		}
		_, err = util.GetQuotaRatioBySnapshotBlock(db, info.InvokingAddr, sb)
		if err != nil {
			// receiver address deleted, delete timer
			triggerInfoKey, triggerInfo, err := abi.GetTimerTriggerInfoByInnerId(db, innerId)
			util.DealWithErr(err)
			refundBlocks := deleteAndRefund(db, infoKey, triggerInfoKey, info, triggerInfo, sb)
			returnBlocks = append(returnBlocks, refundBlocks...)
			continue
		}
		if next+info.Window <= current {
			// current timer skipped
			next = lastTrigger(next, current, info.Interval)
			if next > current || next+info.Window <= current {
				next = next + info.Interval
				triggerInfoKey, triggerInfo, err := abi.GetTimerTriggerInfoByInnerId(db, innerId)
				util.DealWithErr(err)
				if timerFinish(expiringType, info.ExpiringCondition, next) {
					// next timer finish
					refundBlocks := deleteAndRefund(db, infoKey, triggerInfoKey, info, triggerInfo, sb)
					returnBlocks = append(returnBlocks, refundBlocks...)
					continue
				}
				// prepare next trigger
				triggerInfoValue, _ := abi.ABITimer.PackVariable(abi.VariableNameTimerTriggerInfo, triggerInfo.Amount, triggerInfo.TriggerTimes, next, triggerInfo.Delete)
				err = db.SetValue(triggerInfoKey, triggerInfoValue)
				util.DealWithErr(err)
				err = db.SetValue(abi.GetTimerNewQueueKey(iterator.Key(), next), iterator.Value())
				continue
			}
		}
		// trigger
		data, _ := abi.ABITimerNotify.PackMethod(abi.MethodNameTimerNotify, current, info.TimerId)
		returnBlocks = append(returnBlocks, &ledger.AccountBlock{
			AccountAddress: types.AddressTimer,
			ToAddress:      info.InvokingAddr,
			BlockType:      ledger.BlockTypeSendCall,
			Amount:         big.NewInt(0),
			TokenId:        ledger.ViteTokenId,
			Data:           data,
		})
		triggerInfoKey, triggerInfo, err := abi.GetTimerTriggerInfoByInnerId(db, innerId)
		util.DealWithErr(err)
		chargeType := abi.GetChargeTypeFromTimerType(info.TimerType)
		if chargeType == abi.TimerChargeTypeCharge {
			addFee(db, timerChargeAmountPerTrigger)
			triggerInfo.Amount.Sub(triggerInfo.Amount, timerChargeAmountPerTrigger)
		}

		next = nextTriggerByIntervalType(next, current, info.Interval, intervalType)
		if (expiringType == abi.TimerExpiringTypeTimes && triggerInfo.TriggerTimes+1 == info.ExpiringCondition) ||
			timerFinish(expiringType, info.ExpiringCondition, next) {
			// timer finish, delete all
			refundBlocks := deleteAndRefund(db, infoKey, triggerInfoKey, info, triggerInfo, sb)
			returnBlocks = append(returnBlocks, refundBlocks...)
			continue
		}

		// prepare next trigger
		if chargeType == abi.TimerChargeTypeFree || triggerInfo.Amount.Sign() > 0 {
			err := db.SetValue(abi.GetTimerNewQueueKey(iterator.Key(), next), innerId)
			util.DealWithErr(err)
		} else {
			// arrearage timer, put into stopped queue
			triggerInfo.Delete = currentHeight + TimerArrearageDeleteHeight
			err := db.SetValue(abi.GetTimerStoppedQueueKey(innerId, triggerInfo.Delete), innerId)
			util.DealWithErr(err)
		}
		triggerInfoValue, _ := abi.ABITimer.PackVariable(abi.VariableNameTimerTriggerInfo, triggerInfo.Amount, triggerInfo.TriggerTimes+1, next, triggerInfo.Delete)
		err = db.SetValue(triggerInfoKey, triggerInfoValue)
		util.DealWithErr(err)
	}
	return num, returnBlocks
}

func deleteExpiredTimer(db vm_db.VmDb, currentHeight uint64, sb *ledger.SnapshotBlock, num int, returnBlocks []*ledger.AccountBlock) (int, []*ledger.AccountBlock) {
	iterator, err := db.NewStorageIterator(abi.GetTimerStoppedQueueKeyPrefix())
	util.DealWithErr(err)
	defer iterator.Release()
	for {
		if num >= TimerTriggerNumMax {
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
			// next arrearage timer not due
			break
		}

		num = num + 1
		innerId := iterator.Value()
		err := db.SetValue(iterator.Key(), nil)
		util.DealWithErr(err)
		infoKey, info, err := abi.GetTimerInfoByInnerId(db, innerId)
		util.DealWithErr(err)

		deleteAndRefund(db, infoKey, abi.GetTimerTriggerInfoKey(innerId), info, nil, sb)
	}
	return num, returnBlocks
}

func deleteAndRefund(db vm_db.VmDb, infoKey, triggerInfoKey []byte, info *abi.TimerInfo, triggerInfo *abi.TimerTriggerInfo, sb *ledger.SnapshotBlock) []*ledger.AccountBlock {
	err := db.SetValue(abi.GetTimerIdKey(info.TimerId), nil)
	util.DealWithErr(err)
	err = db.SetValue(infoKey, nil)
	util.DealWithErr(err)
	err = db.SetValue(triggerInfoKey, nil)
	util.DealWithErr(err)
	_, err = util.GetQuotaRatioBySnapshotBlock(db, info.RefundAddr, sb)
	if err == nil {
		return refundTimer(triggerInfo, info.RefundAddr)
	}
	return nil
}

func burnFee(db vm_db.VmDb, returnBlocks []*ledger.AccountBlock) []*ledger.AccountBlock {
	feeKey, feeAmount, err := abi.GetTimerFee(db)
	util.DealWithErr(err)
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
		err := db.SetValue(feeKey, nil)
		util.DealWithErr(err)
	}
	return returnBlocks
}

func addFee(db vm_db.VmDb, amount *big.Int) {
	key, feeAmount, err := abi.GetTimerFee(db)
	util.DealWithErr(err)
	feeAmount.Add(feeAmount, amount)
	db.SetValue(key, feeAmount.Bytes())
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

func nextTrigger(start, current, gap uint64) uint64 {
	if start > current {
		return start
	}
	return start + (current-start)/gap*gap + gap
}

func nextTriggerByIntervalType(last, current, interval uint64, intervalType uint8) uint64 {
	if intervalType == abi.TimerIntervalTypeFixed {
		return last + (current-last)/interval*interval + interval
	} else {
		return current + interval
	}
}

func lastTrigger(start, current, interval uint64) uint64 {
	return start + (current-start)/interval*interval
}

func checkTimerChargeAmount(amount *big.Int, tokenId types.TokenTypeId) error {
	if tokenId != ledger.ViteTokenId || amount.Cmp(timerChargeAmountPerTrigger) < 0 || new(big.Int).Mod(amount, timerChargeAmountPerTrigger).Sign() != 0 {
		return util.ErrInvalidMethodParam
	}
	return nil
}

func getAndSetTimerNextId(db vm_db.VmDb) uint64 {
	timerNextIdKey, nextId, err := abi.GetTimerNextIndex(db)
	util.DealWithErr(err)
	err = db.SetValue(timerNextIdKey, new(big.Int).SetUint64(nextId+1).Bytes())
	util.DealWithErr(err)
	return nextId
}

func deleteTimerTriggerInfoAndRefund(db vm_db.VmDb, innerId []byte, refundAddr types.Address) (*abi.TimerTriggerInfo, []*ledger.AccountBlock) {
	triggerInfoKey, triggerInfo, err := abi.GetTimerTriggerInfoByInnerId(db, innerId)
	util.DealWithErr(err)
	err = db.SetValue(triggerInfoKey, nil)
	util.DealWithErr(err)
	return triggerInfo, refundTimer(triggerInfo, refundAddr)
}

func refundTimer(triggerInfo *abi.TimerTriggerInfo, refundAddr types.Address) []*ledger.AccountBlock {
	if triggerInfo != nil && triggerInfo.Amount.Sign() > 0 {
		return []*ledger.AccountBlock{
			{
				AccountAddress: types.AddressTimer,
				ToAddress:      refundAddr,
				BlockType:      ledger.BlockTypeSendCall,
				Amount:         triggerInfo.Amount,
				TokenId:        ledger.ViteTokenId,
			},
		}
	}
	return nil
}

func getTimerOwner(db vm_db.VmDb) types.Address {
	owner, err := abi.GetTimerOwner(db)
	util.DealWithErr(err)
	if owner != nil {
		return *owner
	}
	return nodeConfig.params.TimerOwnerAddressDefault
}

func timerFinish(endType uint8, endCondition, future uint64) bool {
	if endType == abi.TimerExpiringTypeTimeHeight &&
		future > endCondition {
		return true
	}
	return false
}
