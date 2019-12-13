package contracts

import (
	"bytes"
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/ledger"
	"github.com/vitelabs/go-vite/log15"
	cabi "github.com/vitelabs/go-vite/vm/contracts/abi"
	"github.com/vitelabs/go-vite/vm/contracts/defi"
	"github.com/vitelabs/go-vite/vm/util"
	"github.com/vitelabs/go-vite/vm_db"
	"math/big"
)

var deFiLogger = log15.New("module", "deFi")

type MethodDeFiDeposit struct {
	MethodName string
}

func (md *MethodDeFiDeposit) GetFee(block *ledger.AccountBlock) (*big.Int, error) {
	return big.NewInt(0), nil
}

func (md *MethodDeFiDeposit) GetRefundData(sendBlock *ledger.AccountBlock, sbHeight uint64) ([]byte, bool) {
	return []byte{}, false
}

func (md *MethodDeFiDeposit) GetSendQuota(data []byte, gasTable *util.QuotaTable) (uint64, error) {
	return util.RequestQuotaCost(data, gasTable)
}

func (md *MethodDeFiDeposit) GetReceiveQuota(gasTable *util.QuotaTable) uint64 {
	return gasTable.DeFiDepositQuota
}

func (md *MethodDeFiDeposit) DoSend(db vm_db.VmDb, block *ledger.AccountBlock) error {
	if block.Amount.Sign() <= 0 || block.TokenId != ledger.ViteTokenId {
		return defi.InvalidInputParamErr
	}
	return nil
}

func (md *MethodDeFiDeposit) DoReceive(db vm_db.VmDb, block *ledger.AccountBlock, sendBlock *ledger.AccountBlock, vm vmEnvironment) ([]*ledger.AccountBlock, error) {
	defi.OnAccDeposit(db, sendBlock.AccountAddress, sendBlock.TokenId, sendBlock.Amount)
	return nil, nil
}

type MethodDeFiWithdraw struct {
	MethodName string
}

func (md *MethodDeFiWithdraw) GetFee(block *ledger.AccountBlock) (*big.Int, error) {
	return big.NewInt(0), nil
}

func (md *MethodDeFiWithdraw) GetRefundData(sendBlock *ledger.AccountBlock, sbHeight uint64) ([]byte, bool) {
	return []byte{}, false
}

func (md *MethodDeFiWithdraw) GetSendQuota(data []byte, gasTable *util.QuotaTable) (uint64, error) {
	return util.RequestQuotaCost(data, gasTable)
}

func (md *MethodDeFiWithdraw) GetReceiveQuota(gasTable *util.QuotaTable) uint64 {
	return gasTable.DeFiWithdrawQuota
}

func (md *MethodDeFiWithdraw) DoSend(db vm_db.VmDb, block *ledger.AccountBlock) error {
	var err error
	param := new(defi.ParamWithdraw)
	if err = cabi.ABIDeFi.UnpackMethod(param, md.MethodName, block.Data); err != nil {
		return err
	}
	if param.Amount.Sign() <= 0 || block.TokenId != ledger.ViteTokenId {
		return defi.InvalidInputParamErr
	}
	return nil
}

func (md *MethodDeFiWithdraw) DoReceive(db vm_db.VmDb, block *ledger.AccountBlock, sendBlock *ledger.AccountBlock, vm vmEnvironment) ([]*ledger.AccountBlock, error) {
	var (
		param = new(defi.ParamWithdraw)
		err   error
	)
	cabi.ABIDeFi.UnpackMethod(param, md.MethodName, sendBlock.Data)
	if _, err = defi.OnAccWithdraw(db, sendBlock.AccountAddress, param.Token.Bytes(), param.Amount); err != nil {
		return nil, err
	} else {
		return []*ledger.AccountBlock{
			{
				AccountAddress: types.AddressDeFi,
				ToAddress:      sendBlock.AccountAddress,
				BlockType:      ledger.BlockTypeSendCall,
				Amount:         param.Amount,
				TokenId:        param.Token,
				Data:           []byte{},
			},
		}, nil
	}
}

type MethodDeFiNewLoan struct {
	MethodName string
}

func (md *MethodDeFiNewLoan) GetFee(block *ledger.AccountBlock) (*big.Int, error) {
	return big.NewInt(0), nil
}

func (md *MethodDeFiNewLoan) GetRefundData(sendBlock *ledger.AccountBlock, sbHeight uint64) ([]byte, bool) {
	return []byte{}, false
}

func (md *MethodDeFiNewLoan) GetSendQuota(data []byte, gasTable *util.QuotaTable) (uint64, error) {
	return util.RequestQuotaCost(data, gasTable)
}

func (md *MethodDeFiNewLoan) GetReceiveQuota(gasTable *util.QuotaTable) uint64 {
	return gasTable.DeFiNewLoanQuota
}

func (md *MethodDeFiNewLoan) DoSend(db vm_db.VmDb, block *ledger.AccountBlock) error {
	var err error
	param := new(defi.ParamNewLoan)
	if err = cabi.ABIDeFi.UnpackMethod(param, md.MethodName, block.Data); err != nil {
		return err
	}
	return defi.CheckLoanParam(param)
}

func (md *MethodDeFiNewLoan) DoReceive(db vm_db.VmDb, block *ledger.AccountBlock, sendBlock *ledger.AccountBlock, vm vmEnvironment) ([]*ledger.AccountBlock, error) {
	var (
		param = new(defi.ParamNewLoan)
		err   error
	)
	cabi.ABIDeFi.UnpackMethod(param, md.MethodName, sendBlock.Data)
	interest := defi.CalculateInterest(param.Shares, param.ShareAmount, param.DayRate, param.ExpireDays)
	if _, err = defi.OnAccNewLoan(db, sendBlock.AccountAddress, interest); err != nil {
		return handleDeFiReceiveErr(deFiLogger, md.MethodName, err, sendBlock)
	} else {
		loan := defi.NewLoan(sendBlock.AccountAddress, db, param, interest)
		defi.SaveLoan(db, loan)
		defi.AddNewLoanEvent(db, loan)
	}
	return nil, nil
}

type MethodDeFiCancelLoan struct {
	MethodName string
}

func (md *MethodDeFiCancelLoan) GetFee(block *ledger.AccountBlock) (*big.Int, error) {
	return big.NewInt(0), nil
}

func (md *MethodDeFiCancelLoan) GetRefundData(sendBlock *ledger.AccountBlock, sbHeight uint64) ([]byte, bool) {
	return []byte{}, false
}

func (md *MethodDeFiCancelLoan) GetSendQuota(data []byte, gasTable *util.QuotaTable) (uint64, error) {
	return util.RequestQuotaCost(data, gasTable)
}

func (md *MethodDeFiCancelLoan) GetReceiveQuota(gasTable *util.QuotaTable) uint64 {
	return gasTable.DeFiCancelLoanQuota
}

func (md *MethodDeFiCancelLoan) DoSend(db vm_db.VmDb, block *ledger.AccountBlock) error {
	return cabi.ABIDeFi.UnpackMethod(new(uint64), md.MethodName, block.Data)
}

func (md *MethodDeFiCancelLoan) DoReceive(db vm_db.VmDb, block *ledger.AccountBlock, sendBlock *ledger.AccountBlock, vm vmEnvironment) ([]*ledger.AccountBlock, error) {
	loanId := new(uint64)
	cabi.ABIDeFi.UnpackMethod(loanId, md.MethodName, sendBlock.Data)
	if loan, ok := defi.GetLoanById(db, *loanId); !ok {
		return handleDeFiReceiveErr(deFiLogger, md.MethodName, defi.LoanNotExistsErr, sendBlock)
	} else {
		if !bytes.Equal(loan.Address, sendBlock.AccountAddress.Bytes()) {
			return handleDeFiReceiveErr(deFiLogger, md.MethodName, defi.OnlyOwnerAllowErr, sendBlock)
		} else if loan.Status != defi.Open || loan.SubscribedShares != 0 {
			return handleDeFiReceiveErr(deFiLogger, md.MethodName, defi.InvalidLoanStatusForCancelErr, sendBlock)
		} else {
			defi.OnAccLoanCancelled(db, sendBlock.AccountAddress, loan.Interest)
			loan.Status = defi.Failed
			loan.Updated = defi.GetDeFiTimestamp(db)
			defi.DeleteLoan(db, loan)
			defi.AddLoanUpdateEvent(db, loan)
		}
	}
	return nil, nil
}

type MethodDeFiSubscribe struct {
	MethodName string
}

func (md *MethodDeFiSubscribe) GetFee(block *ledger.AccountBlock) (*big.Int, error) {
	return big.NewInt(0), nil
}

func (md *MethodDeFiSubscribe) GetRefundData(sendBlock *ledger.AccountBlock, sbHeight uint64) ([]byte, bool) {
	return []byte{}, false
}

func (md *MethodDeFiSubscribe) GetSendQuota(data []byte, gasTable *util.QuotaTable) (uint64, error) {
	return util.RequestQuotaCost(data, gasTable)
}

func (md *MethodDeFiSubscribe) GetReceiveQuota(gasTable *util.QuotaTable) uint64 {
	return gasTable.DeFiSubscribeQuota
}

func (md *MethodDeFiSubscribe) DoSend(db vm_db.VmDb, block *ledger.AccountBlock) error {
	var err error
	param := new(defi.ParamSubscribe)
	if err = cabi.ABIDeFi.UnpackMethod(param, md.MethodName, block.Data); err != nil {
		return err
	} else {
		if param.LoanId == 0 || param.Shares <= 0 {
			return defi.InvalidInputParamErr
		}
	}
	return nil
}

func (md *MethodDeFiSubscribe) DoReceive(db vm_db.VmDb, block *ledger.AccountBlock, sendBlock *ledger.AccountBlock, vm vmEnvironment) ([]*ledger.AccountBlock, error) {
	var (
		param = new(defi.ParamSubscribe)
		loan  *defi.Loan
		sub   *defi.Subscription
		ok    bool
		err   error
	)
	cabi.ABIDeFi.UnpackMethod(param, md.MethodName, sendBlock.Data)
	if loan, ok = defi.GetLoanById(db, param.LoanId); !ok || loan.Status != defi.Open {
		return handleDexReceiveErr(deFiLogger, md.MethodName, defi.LoanNotExistsErr, sendBlock)
	} else if defi.IsOpenLoanSubscribeFail(db, loan) {
		return handleDexReceiveErr(deFiLogger, md.MethodName, defi.LoanSubscribeFailed, sendBlock)
	}
	leavedShares := loan.Shares - loan.SubscribedShares
	if param.Shares > leavedShares {
		param.Shares = leavedShares
	}
	amount := defi.CalculateAmount(param.Shares, loan.ShareAmount)
	if _, err = defi.OnAccSubscribe(db, sendBlock.AccountAddress, amount); err != nil {
		return handleDexReceiveErr(deFiLogger, md.MethodName, err, sendBlock)
	}
	if sub, ok = defi.GetSubscription(db, param.LoanId, sendBlock.AccountAddress.Bytes()); !ok {
		sub = defi.NewSubscription(sendBlock.AccountAddress, db, param, loan)
	} else {
		sub.Shares = sub.Shares + param.Shares
		sub.Updated = defi.GetDeFiTimestamp(db)
	}
	defi.SaveSubscription(db, sub)

	defi.AddNewSubscriptionEvent(db, sub)
	defi.DoSubscribe(db, vm.GlobalStatus(), loan, param.Shares)
	return nil, nil
}

type MethodDeFiInvest struct {
	MethodName string
}

func (md *MethodDeFiInvest) GetFee(block *ledger.AccountBlock) (*big.Int, error) {
	return big.NewInt(0), nil
}

func (md *MethodDeFiInvest) GetRefundData(sendBlock *ledger.AccountBlock, sbHeight uint64) ([]byte, bool) {
	return []byte{}, false
}

func (md *MethodDeFiInvest) GetSendQuota(data []byte, gasTable *util.QuotaTable) (uint64, error) {
	return util.RequestQuotaCost(data, gasTable)
}

func (md *MethodDeFiInvest) GetReceiveQuota(gasTable *util.QuotaTable) uint64 {
	return gasTable.DeFiInvestQuota
}

func (md *MethodDeFiInvest) DoSend(db vm_db.VmDb, block *ledger.AccountBlock) error {
	var err error
	param := new(defi.ParamInvest)
	if err = cabi.ABIDeFi.UnpackMethod(param, md.MethodName, block.Data); err != nil {
		return err
	} else {
		if param.LoanId == 0 || param.BizType < defi.StakeForMining || param.BizType > defi.StakeForQuota || param.BizType == defi.RegistSBP {
			return defi.InvalidInputParamErr
		}
	}
	return nil
}

func (md *MethodDeFiInvest) DoReceive(db vm_db.VmDb, block *ledger.AccountBlock, sendBlock *ledger.AccountBlock, vm vmEnvironment) ([]*ledger.AccountBlock, error) {
	var (
		param                      = new(defi.ParamInvest)
		loanInvested, baseInvested *big.Int
		durationHeight             uint64
		loan                       *defi.Loan
		err                        error
		ok                         bool
	)
	cabi.ABIDeFi.UnpackMethod(param, md.MethodName, sendBlock.Data)
	if loan, ok = defi.GetLoanById(db, param.LoanId); !ok || loan.Status != defi.Success {
		return handleDexReceiveErr(deFiLogger, md.MethodName, defi.LoanNotExistsErr, sendBlock)
	}
	leavedAmount, availableHeight := defi.GetLoanAvailable(vm.GlobalStatus(), loan)
	if loanInvested, baseInvested, durationHeight, err = defi.DoInvest(db, sendBlock.AccountAddress, param, leavedAmount, stakeAmountMin, availableHeight, nodeConfig.params.StakeHeight, nodeConfig.params.DexSuperVipStakeHeight); err != nil {
		return handleDeFiReceiveErr(deFiLogger, md.MethodName, err, sendBlock)
	}
	invest := defi.NewInvest(db, vm.GlobalStatus(), sendBlock.AccountAddress, loan, param, loanInvested, baseInvested, durationHeight)
	defi.SaveInvest(db, invest)
	switch param.BizType {
	case defi.StakeForMining:

	case defi.StakeForSVIP:

	case defi.StakeForQuota:

	}
	return nil, nil
}

func handleDeFiReceiveErr(logger log15.Logger, method string, err error, sendBlock *ledger.AccountBlock) ([]*ledger.AccountBlock, error) {
	logger.Error("deFi receive with err", "error", err.Error(), "method", method, "sendBlockHash", sendBlock.Hash.String(), "sendAddress", sendBlock.AccountAddress.String())
	return nil, err
}
