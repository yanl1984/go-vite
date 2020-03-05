package contracts

import (
	"math/big"

	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/ledger"
	"github.com/vitelabs/go-vite/vm/abi"
	cabi "github.com/vitelabs/go-vite/vm/contracts/abi"
	"github.com/vitelabs/go-vite/vm/util"
	"github.com/vitelabs/go-vite/vm_db"
)

// InitContractsConfig init params of built-in contracts.This method is
// supposed be called when the node started.
func InitContractsConfig(isTestParam bool) {
}

type vmEnvironment interface {
	GlobalStatus() util.GlobalStatus
	ConsensusReader() util.ConsensusReader
}

// BuiltinContractMethod defines interfaces of built-in contract method
// Send: GetFee->GetSendQuota->DoSend
// Receive: GetReceiveQuota->DoReceive->GetRefundData
type BuiltinContractMethod interface {
	GetFee(block *ledger.AccountBlock) (*big.Int, error)
	// quota for doSend block
	GetSendQuota(data []byte, gasTable *util.QuotaTable) (uint64, error)
	// calc and use quota, check tx data
	DoSend(db vm_db.VmDb, block *ledger.AccountBlock) error

	// receive block quota
	GetReceiveQuota(gasTable *util.QuotaTable) uint64
	// check status, update state
	DoReceive(db vm_db.VmDb, block *ledger.AccountBlock, sendBlock *ledger.AccountBlock, vm vmEnvironment) ([]*ledger.AccountBlock, error)
	// refund data at receive error
	GetRefundData(sendBlock *ledger.AccountBlock, sbHeight uint64) ([]byte, bool)
}

type builtinContract struct {
	m   map[string]BuiltinContractMethod
	abi abi.ABIContract
}

var (
	simpleContracts = newSimpleContracts()
)

func newSimpleContracts() map[types.Address]*builtinContract {
	return map[types.Address]*builtinContract{
		types.AddressGovernance: {
			map[string]BuiltinContractMethod{
				cabi.MethodNameRegister:                    &MethodRegister{cabi.MethodNameRegister},
				cabi.MethodNameRevoke:                      &MethodRevoke{cabi.MethodNameRevoke},
				cabi.MethodNameUpdateBlockProducingAddress: &MethodUpdateBlockProducingAddress{cabi.MethodNameUpdateBlockProducingAddress},
			},
			cabi.ABIGovernance,
		},
	}
}

// GetBuiltinContractMethod finds method instance of built-in contract method by address and method id
func GetBuiltinContractMethod(addr types.Address, methodSelector []byte, sbHeight uint64) (BuiltinContractMethod, bool, error) {
	var contractsMap map[types.Address]*builtinContract
	contractsMap = simpleContracts

	p, addrExists := contractsMap[addr]
	if addrExists {
		if method, err := p.abi.MethodById(methodSelector); err == nil {
			c, methodExists := p.m[method.Name]
			if methodExists {
				return c, methodExists, nil
			}
		}
		return nil, addrExists, util.ErrAbiMethodNotFound
	}
	return nil, addrExists, nil
}

// NewLog generate vm log
func NewLog(c abi.ABIContract, name string, params ...interface{}) *ledger.VmLog {
	topics, data, _ := c.PackEvent(name, params...)
	return &ledger.VmLog{Topics: topics, Data: data}
}
