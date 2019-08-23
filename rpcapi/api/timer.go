package api

import (
	"github.com/vitelabs/go-vite/chain"
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/log15"
	"github.com/vitelabs/go-vite/vite"
	"github.com/vitelabs/go-vite/vm/contracts/abi"
	"math/big"
)

type TimerApi struct {
	chain chain.Chain
	log   log15.Logger
}

func NewTimerApi(vite *vite.Vite) *TimerApi {
	return &TimerApi{
		chain: vite.Chain(),
		log:   log15.New("module", "rpc_api/timer"),
	}
}

func (p TimerApi) String() string {
	return "TimerApi"
}

type NewTimerParam struct {
	TimeHeight        uint8         `json:"timeHeight"`
	ExpiringType      uint8         `json:"expiringType"`
	IntervalType      uint8         `json:"intervalType"`
	Start             string        `json:"start"`
	Window            string        `json:"window"`
	Interval          string        `json:"interval"`
	ExpiringCondition string        `json:"expiringCondition"`
	InvokingAddr      types.Address `json:"invokingAddr"`
	RefundAddress     types.Address `json:"refundAddr"`
}

func (t *TimerApi) GetNewTimerData(param NewTimerParam) ([]byte, error) {
	start, err := StringToUint64(param.Start)
	if err != nil {
		return nil, err
	}
	window, err := StringToUint64(param.Window)
	if err != nil {
		return nil, err
	}
	interval, err := StringToUint64(param.Interval)
	if err != nil {
		return nil, err
	}
	expiringCondition, err := StringToUint64(param.ExpiringCondition)
	if err != nil {
		return nil, err
	}
	return abi.ABITimer.PackMethod(
		abi.MethodNameTimerNewTimer,
		abi.GenerateTimerType(param.TimeHeight, param.ExpiringType, param.IntervalType),
		start,
		window,
		interval,
		expiringCondition,
		param.InvokingAddr,
		param.RefundAddress)
}

func (t *TimerApi) GetDeleteTimerData(timerId types.Hash, refundAddress types.Address) ([]byte, error) {
	return abi.ABITimer.PackMethod(abi.MethodNameTimerDeleteTimer, timerId, refundAddress)
}

func (t *TimerApi) GetDepositData(timerId types.Hash) ([]byte, error) {
	return abi.ABITimer.PackMethod(abi.MethodNameTimerDeposit, timerId)
}

func (t *TimerApi) GetUpdateOwnerData(newOwner types.Address) ([]byte, error) {
	return abi.ABITimer.PackMethod(abi.MethodNameTimerTransferOwnership, newOwner)
}

type TimerInfo struct {
	TimerId           types.Hash    `json:"timerId"`
	TimeHeight        uint8         `json:"timeHeight"`
	ExpiringType      uint8         `json:"expiringType"`
	IntervalType      uint8         `json:"intervalType"`
	Window            string        `json:"window"`
	Interval          string        `json:"interval"`
	ExpiringCondition string        `json:"expiringCondition"`
	InvokingAddr      types.Address `json:"invokingAddr"`
	RefundAddr        types.Address `json:"refundAddr"`
	Owner             types.Address `json:"owner"`
	Index             string        `json:"index"`
	Amount            *string       `json:"amount"`
	TriggerTimes      string        `json:"triggerTimes"`
	Next              string        `json:"next"`
	Delete            string        `json:"delete"`
}

func (t *TimerApi) GetTimerInfoListByOwner(owner types.Address) ([]*TimerInfo, error) {
	db, err := getVmDb(t.chain, types.AddressTimer)
	if err != nil {
		return nil, err
	}
	list, err := abi.GetTimerInfoListByOwner(db, owner)
	if err != nil {
		return nil, err
	}
	infoList := make([]*TimerInfo, len(list))
	for i, info := range list {
		_, triggerInfo, err := abi.GetTimerTriggerInfoByInnerId(db, info.InnerId)
		if err != nil {
			return nil, err
		}
		if triggerInfo == nil {
			continue
		}
		infoList[i] = convertTimerInfo(info, triggerInfo, info.InnerId)
	}
	return infoList, nil
}

func (t *TimerApi) GetTimerInfoById(timerId types.Hash) (*TimerInfo, error) {
	db, err := getVmDb(t.chain, types.AddressTimer)
	if err != nil {
		return nil, err
	}
	innerId, err := abi.GetInnerIdByTimerId(db, timerId)
	if err != nil {
		return nil, err
	}
	if len(innerId) == 0 {
		return nil, nil
	}
	_, info, err := abi.GetTimerInfoByInnerId(db, innerId)
	if err != nil {
		return nil, err
	}
	if info == nil {
		return nil, nil
	}
	_, triggerInfo, err := abi.GetTimerTriggerInfoByInnerId(db, innerId)
	if err != nil {
		return nil, err
	}
	if triggerInfo == nil {
		return nil, nil
	}
	return convertTimerInfo(info, triggerInfo, innerId), nil
}

func convertTimerInfo(info *abi.TimerInfo, triggerInfo *abi.TimerTriggerInfo, innerId []byte) *TimerInfo {
	target := &TimerInfo{
		TimerId:           info.TimerId,
		Window:            Uint64ToString(info.Window),
		Interval:          Uint64ToString(info.Interval),
		ExpiringCondition: Uint64ToString(info.ExpiringCondition),
		InvokingAddr:      info.InvokingAddr,
		RefundAddr:        info.RefundAddr,
		Owner:             abi.GetOwnerFromInnerId(innerId),
		Index:             Uint64ToString(new(big.Int).SetBytes(abi.GetIndexFromInnerId(innerId)).Uint64()),
		Amount:            bigIntToString(triggerInfo.Amount),
		TriggerTimes:      Uint64ToString(triggerInfo.TriggerTimes),
		Next:              Uint64ToString(triggerInfo.Next),
		Delete:            Uint64ToString(triggerInfo.Delete),
	}
	target.TimeHeight, target.ExpiringType, target.IntervalType = abi.GetTimerTypeDetail(info.TimerType)
	return target
}

type TimerContractInfo struct {
	Fee               *string        `json:"fee"`
	LastTriggerHeight string         `json:"lastTriggerHeight"`
	LastTriggerTime   string         `json:"lastTriggerTime"`
	NextId            string         `json:"nextId"`
	Owner             *types.Address `json:"owner"`
}

func (t *TimerApi) GetTimerInfo() (*TimerContractInfo, error) {
	db, err := getVmDb(t.chain, types.AddressTimer)
	if err != nil {
		return nil, err
	}
	_, fee, err := abi.GetTimerFee(db)
	if err != nil {
		return nil, err
	}
	lastTriggerInfo, err := abi.GetTimerLastTriggerInfo(db)
	if err != nil {
		return nil, err
	}
	_, nextId, err := abi.GetTimerNextIndex(db)
	if err != nil {
		return nil, err
	}
	owner, err := abi.GetTimerOwner(db)
	if err != nil {
		return nil, err
	}
	return &TimerContractInfo{
		Fee:               bigIntToString(fee),
		LastTriggerHeight: Uint64ToString(lastTriggerInfo.Height),
		LastTriggerTime:   Uint64ToString(lastTriggerInfo.Timestamp),
		NextId:            Uint64ToString(nextId),
		Owner:             owner,
	}, nil
}

type TimerContractSummary struct {
	QueueCount        string `json:"queueCount"`
	StoppedQueueCount string `json:"stoppedQueueCount"`
}

func (t *TimerApi) GetTimerContractSummary() (*TimerContractSummary, error) {
	db, err := getVmDb(t.chain, types.AddressTimer)
	if err != nil {
		return nil, err
	}
	queueCount, stoppedQueueCount, err := abi.GetTimerSummary(db)
	if err != nil {
		return nil, err
	}
	return &TimerContractSummary{
		QueueCount:        Uint64ToString(queueCount),
		StoppedQueueCount: Uint64ToString(stoppedQueueCount),
	}, nil
}
