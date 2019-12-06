package contracts

import (
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
	defi.OnDeposit(db, sendBlock.AccountAddress, sendBlock.TokenId, sendBlock.Amount)
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
	if _, err = defi.OnWithdraw(db, sendBlock.AccountAddress, param.Token.Bytes(), param.Amount); err != nil {
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
	if param.Amount.Sign() <= 0 || block.TokenId != ledger.ViteTokenId {
		return defi.InvalidInputParamErr
	}
	return nil
}

func (md *MethodDeFiNewLoan) DoReceive(db vm_db.VmDb, block *ledger.AccountBlock, sendBlock *ledger.AccountBlock, vm vmEnvironment) ([]*ledger.AccountBlock, error) {
	var (
		param = new(defi.ParamNewLoan)
		err   error
	)
	cabi.ABIDeFi.UnpackMethod(param, md.MethodName, sendBlock.Data)
	if _, err = defi.OnWithdraw(db, sendBlock.AccountAddress, param.Token.Bytes(), param.Amount); err != nil {
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

func handleDeFiReceiveErr(logger log15.Logger, method string, err error, sendBlock *ledger.AccountBlock) ([]*ledger.AccountBlock, error) {
	logger.Error("deFi receive with err", "error", err.Error(), "method", method, "sendBlockHash", sendBlock.Hash.String(), "sendAddress", sendBlock.AccountAddress.String())
	return nil, err
}
