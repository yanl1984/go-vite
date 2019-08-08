package contracts

import (
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

	if err = checkTimerChargeAmount(block); err != nil {
		return err
	}

	timeHeight, endType, skipType, triggerType := abi.GetTimerTaskTypeDetail(param.TaskType)
	if generatedTimerTaskType := abi.GenerateTimerTaskType(timeHeight, endType, skipType, triggerType); generatedTimerTaskType != param.TaskType {
		return util.ErrInvalidMethodParam
	}
	if (timeHeight != abi.TimeHeightTime && timeHeight != abi.TimeHeightHeight) ||
		(endType != abi.EndTypeOnce && endType != abi.EndTypeEndTimeHeight && endType != abi.EndTypeTimes && endType != abi.EndTypePermanent) ||
		(skipType != abi.SkipTypeImmediate && skipType != abi.SkipTypeSkip) ||
		(triggerType != abi.TriggerTypeFixed && triggerType != abi.TriggerTypeVariable) {
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

	if (endType == abi.EndTypeOnce && param.EndCondition != 0) ||
		(endType == abi.EndTypeEndTimeHeight && param.EndCondition <= 0) ||
		(endType == abi.EndTypeTimes && param.EndCondition <= 0) ||
		(endType == abi.EndTypePermanent && param.EndCondition != 0) {
		return util.ErrInvalidMethodParam
	}

	block.Data, _ = abi.ABIMintage.PackMethod(
		abi.MethodNameTimerNewTask,
		abi.GenerateTimerTaskType(timeHeight, endType, skipType, triggerType),
		param.Start,
		param.Gap,
		param.EndCondition,
		param.ReceiverAddress,
		param.Bid)
	return nil
}

func checkTimerChargeAmount(block *ledger.AccountBlock) error {
	if block.TokenId != ledger.ViteTokenId || block.Amount.Cmp(timerChargeAmountPerTask) < 0 || new(big.Int).Mod(block.Amount, timerChargeAmountPerTask).Sign() != 0 {
		return util.ErrInvalidMethodParam
	}
	return nil
}
func (p *MethodTimerNewTask) DoReceive(db vm_db.VmDb, block *ledger.AccountBlock, sendBlock *ledger.AccountBlock, vm vmEnvironment) ([]*ledger.AccountBlock, error) {
	param := new(abi.ParamTimerNewTask)
	abi.ABITimer.UnpackMethod(param, abi.MethodNameTimerNewTask, block.Data)

	// TODO
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
	// TODO
	return nil
}
func (p *MethodTimerDeleteTask) DoReceive(db vm_db.VmDb, block *ledger.AccountBlock, sendBlock *ledger.AccountBlock, vm vmEnvironment) ([]*ledger.AccountBlock, error) {
	// TODO
	return nil, nil
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
	// TODO
	return nil
}
func (p *MethodTimerRecharge) DoReceive(db vm_db.VmDb, block *ledger.AccountBlock, sendBlock *ledger.AccountBlock, vm vmEnvironment) ([]*ledger.AccountBlock, error) {
	// TODO
	return nil, nil
}
