package contracts

import (
	"math/big"

	"github.com/vitelabs/go-vite/ledger"
	"github.com/vitelabs/go-vite/vm/contracts/abi"
	"github.com/vitelabs/go-vite/vm/util"
	"github.com/vitelabs/go-vite/vm_db"
)

type MethodRegister struct {
	MethodName string
}

func (p MethodRegister) unpackParam(data []byte) (*abi.ParamRegister, error) {
	param := new(abi.ParamRegister)
	if err := abi.ABIGovernance.UnpackMethod(param, p.MethodName, data); err != nil {
		return nil, util.ErrInvalidMethodParam
	}
	return param, nil
}

func (p MethodRegister) packParam(param *abi.ParamRegister) ([]byte, error) {
	return abi.ABIGovernance.PackMethod(p.MethodName, param.Serialize()...)
}

func (p *MethodRegister) GetFee(block *ledger.AccountBlock) (*big.Int, error) {
	return big.NewInt(0), nil
}
func (p *MethodRegister) GetRefundData(sendBlock *ledger.AccountBlock, sbHeight uint64) ([]byte, bool) {
	return []byte{}, false
}
func (p *MethodRegister) GetSendQuota(data []byte, gasTable *util.QuotaTable) (uint64, error) {
	return gasTable.RegisterQuota, nil
}
func (p *MethodRegister) GetReceiveQuota(gasTable *util.QuotaTable) uint64 {
	return 0
}

func (p *MethodRegister) DoSend(db vm_db.VmDb, block *ledger.AccountBlock) error {
	// check amount
	if block.Amount.Sign() != 0 {
		return util.ErrInvalidMethodParam
	}
	param, err := p.unpackParam(block.Data)
	util.AssertNil(err)

	//governance, err := abi.NewGovernance(db)
	//util.AssertNil(err)
	//
	//err = governance.CheckSbpExistForOwner(param.ProposerSbpName, block.AccountAddress)
	//util.AssertNil(err)
	//
	//_, err = governance.CheckSbpExistForRegister(param.SbpName, param.BlockProducingAddress, 0)

	block.Data, _ = p.packParam(param)
	return err
}

func (p *MethodRegister) DoReceive(db vm_db.VmDb, block *ledger.AccountBlock, sendBlock *ledger.AccountBlock, vm vmEnvironment) ([]*ledger.AccountBlock, error) {
	// Check param by group info
	param, err := p.unpackParam(sendBlock.Data)

	snapshotBlock := vm.GlobalStatus().SnapshotBlock()

	governance, err := abi.NewGovernance(db)
	util.AssertNil(err)

	err = governance.CheckSbpExistForOwner(param.ProposerSbpName, sendBlock.AccountAddress)
	if err != nil {
		return nil, err
	}

	err = governance.Register(param.SbpName, param.BlockProducingAddress, param.OwnerAddress, param.ProposerSbpName, snapshotBlock.Height)
	if err != nil {
		return nil, err
	}
	return nil, nil
}

type MethodVote struct {
	MethodName string
}

func (p MethodVote) unpackParam(data []byte) (*abi.ParamVote, error) {
	param := new(abi.ParamVote)
	if err := abi.ABIGovernance.UnpackMethod(param, p.MethodName, data); err != nil {
		return nil, util.ErrInvalidMethodParam
	}
	return param, nil
}

func (p MethodVote) packParam(param *abi.ParamVote) ([]byte, error) {
	return abi.ABIGovernance.PackMethod(p.MethodName, param.Serialize()...)
}

func (p *MethodVote) GetFee(block *ledger.AccountBlock) (*big.Int, error) {
	return big.NewInt(0), nil
}
func (p *MethodVote) GetRefundData(sendBlock *ledger.AccountBlock, sbHeight uint64) ([]byte, bool) {
	return []byte{}, false
}
func (p *MethodVote) GetSendQuota(data []byte, gasTable *util.QuotaTable) (uint64, error) {
	return gasTable.RegisterQuota, nil
}
func (p *MethodVote) GetReceiveQuota(gasTable *util.QuotaTable) uint64 {
	return 0
}

func (p *MethodVote) DoSend(db vm_db.VmDb, block *ledger.AccountBlock) error {
	// check amount
	if block.Amount.Sign() != 0 {
		return util.ErrInvalidMethodParam
	}
	param, err := p.unpackParam(block.Data)
	util.AssertNil(err)
	//
	//governance, err := abi.NewGovernance(db)
	//util.AssertNil(err)
	//err = governance.CheckSbpExistForOwner(param.ProposerSbpName, block.AccountAddress)
	//util.AssertNil(err)
	//
	//_, err = governance.CheckSbpExistForVoting(param.SbpName, param.VoteType, 0, param.ProposerSbpName)
	block.Data, _ = p.packParam(param)
	return err
}

func (p *MethodVote) DoReceive(db vm_db.VmDb, block *ledger.AccountBlock, sendBlock *ledger.AccountBlock, vm vmEnvironment) ([]*ledger.AccountBlock, error) {
	// Check param by group info
	param, err := p.unpackParam(sendBlock.Data)

	snapshotBlock := vm.GlobalStatus().SnapshotBlock()

	governance, err := abi.NewGovernance(db)
	util.AssertNil(err)
	err = governance.CheckSbpExistForOwner(param.ProposerSbpName, sendBlock.AccountAddress)
	if err != nil {
		return nil, err
	}

	err = governance.Voting(param.SbpName, param.VoteType, param.Approval, param.ProposerSbpName, snapshotBlock.Height)
	if err != nil {
		return nil, err
	}
	return nil, nil
}

type MethodRevoke struct {
	MethodName string
}

func (p MethodRevoke) unpackParam(data []byte) (*abi.ParamRevoke, error) {
	param := new(abi.ParamRevoke)
	if err := abi.ABIGovernance.UnpackMethod(param, p.MethodName, data); err != nil {
		return nil, util.ErrInvalidMethodParam
	}
	return param, nil
}

func (p MethodRevoke) packParam(param *abi.ParamRevoke) ([]byte, error) {
	return abi.ABIGovernance.PackMethod(p.MethodName, param.Serialize()...)
}

func (p *MethodRevoke) GetFee(block *ledger.AccountBlock) (*big.Int, error) {
	return big.NewInt(0), nil
}
func (p *MethodRevoke) GetRefundData(sendBlock *ledger.AccountBlock, sbHeight uint64) ([]byte, bool) {
	return []byte{}, false
}
func (p *MethodRevoke) GetSendQuota(data []byte, gasTable *util.QuotaTable) (uint64, error) {
	return gasTable.RevokeQuota, nil
}
func (p *MethodRevoke) GetReceiveQuota(gasTable *util.QuotaTable) uint64 {
	return 0
}

func (p *MethodRevoke) DoSend(db vm_db.VmDb, block *ledger.AccountBlock) error {
	// check amount
	if block.Amount.Sign() != 0 {
		return util.ErrInvalidMethodParam
	}
	param, err := p.unpackParam(block.Data)
	util.AssertNil(err)
	//
	//governance, err := abi.NewGovernance(db)
	//util.AssertNil(err)
	//
	//err = governance.CheckSbpExistForOwner(param.ProposerSbpName, block.AccountAddress)
	//util.AssertNil(err)
	//
	//_, err = governance.CheckSbpExistForRevoke(param.SbpName)
	block.Data, _ = p.packParam(param)
	return err
}
func (p *MethodRevoke) DoReceive(db vm_db.VmDb, block *ledger.AccountBlock, sendBlock *ledger.AccountBlock, vm vmEnvironment) ([]*ledger.AccountBlock, error) {
	// Check param by group info
	param, err := p.unpackParam(sendBlock.Data)

	snapshotBlock := vm.GlobalStatus().SnapshotBlock()

	vm.GlobalStatus().SnapshotBlock()
	governance, err := abi.NewGovernance(db)
	util.AssertNil(err)

	err = governance.CheckSbpExistForOwner(param.ProposerSbpName, sendBlock.AccountAddress)
	if err != nil {
		return nil, err
	}

	err = governance.Revoke(param.SbpName, param.ProposerSbpName, snapshotBlock.Height)
	if err != nil {
		return nil, err
	}
	return nil, nil
}

type MethodUpdateBlockProducingAddress struct {
	MethodName string
}

func (p MethodUpdateBlockProducingAddress) unpackParam(data []byte) (*abi.ParamUpdateProducingAddress, error) {
	param := new(abi.ParamUpdateProducingAddress)
	if err := abi.ABIGovernance.UnpackMethod(param, p.MethodName, data); err != nil {
		return nil, util.ErrInvalidMethodParam
	}
	return param, nil
}
func (p MethodUpdateBlockProducingAddress) packParam(param *abi.ParamUpdateProducingAddress) ([]byte, error) {
	return abi.ABIGovernance.PackMethod(p.MethodName, param.Serialize()...)
}

func (p *MethodUpdateBlockProducingAddress) GetFee(block *ledger.AccountBlock) (*big.Int, error) {
	return big.NewInt(0), nil
}

func (p *MethodUpdateBlockProducingAddress) GetRefundData(sendBlock *ledger.AccountBlock, sbHeight uint64) ([]byte, bool) {
	return []byte{}, false
}
func (p *MethodUpdateBlockProducingAddress) GetSendQuota(data []byte, gasTable *util.QuotaTable) (uint64, error) {
	return gasTable.UpdateBlockProducingAddressQuota, nil
}
func (p *MethodUpdateBlockProducingAddress) GetReceiveQuota(gasTable *util.QuotaTable) uint64 {
	return 0
}

func (p *MethodUpdateBlockProducingAddress) DoSend(db vm_db.VmDb, block *ledger.AccountBlock) error {
	// check amount
	if block.Amount.Sign() != 0 {
		return util.ErrInvalidMethodParam
	}
	param, err := p.unpackParam(block.Data)
	util.AssertNil(err)
	//
	//governance, err := abi.NewGovernance(db)
	//util.AssertNil(err)
	//
	//// check owner
	//err = governance.CheckSbpExistForOwner(param.SbpName, block.AccountAddress)
	//util.AssertNil(err)
	//// check producingAddress repeat
	//_, err = governance.CheckSbpExistForUpdateProducingAddress(param.SbpName, param.BlockProducingAddress)
	//util.AssertNil(err)

	block.Data, _ = p.packParam(param)
	return err
}
func (p *MethodUpdateBlockProducingAddress) DoReceive(db vm_db.VmDb, block *ledger.AccountBlock, sendBlock *ledger.AccountBlock, vm vmEnvironment) ([]*ledger.AccountBlock, error) {
	// Check param by group info
	param, err := p.unpackParam(sendBlock.Data)

	governance, err := abi.NewGovernance(db)
	util.AssertNil(err)

	err = governance.CheckSbpExistForOwner(param.SbpName, sendBlock.AccountAddress)
	if err != nil {
		return nil, err
	}

	err = governance.UpdateProducingAddress(param.BlockProducingAddress, param.SbpName)
	if err != nil {
		return nil, err
	}
	return nil, nil
}
