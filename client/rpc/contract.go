//

package rpc

import (
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/rpc"
	"github.com/vitelabs/go-vite/rpcapi/api"
	"github.com/vitelabs/go-vite/vm/contracts/abi"
)

// ContractApi ...
type ContractApi interface {
	CallOffChainMethod(param api.CallOffChainMethodParam) ([]byte, error)
	GetCreateContractData(param api.CreateContractDataParam) ([]byte, error)
	GetContractStorage(addr types.Address, prefix string) (map[string]string, error)
	GetContractInfo(addr types.Address) (*api.ContractInfo, error)
	GetGovernanceState() ([]*abi.SbpInfo, []*types.ConsensusGroupInfo, error)
}

type contractApi struct {
	cc *rpc.Client
}

func NewContractApi(cc *rpc.Client) ContractApi {
	return &contractApi{cc: cc}
}

func (ci contractApi) GetCreateContractData(param api.CreateContractDataParam) (result []byte, err error) {
	err = ci.cc.Call(&result, "contract_getCreateContractData", param)
	return
}
func (ci contractApi) CallOffChainMethod(param api.CallOffChainMethodParam) (result []byte, err error) {
	err = ci.cc.Call(&result, "contract_callOffChainMethod", param)
	return
}

func (ci contractApi) GetContractStorage(addr types.Address, prefix string) (result map[string]string, err error) {
	result = make(map[string]string)
	err = ci.cc.Call(&result, "contract_getContractStorage", addr, prefix)
	return
}

func (ci contractApi) GetContractInfo(addr types.Address) (result *api.ContractInfo, err error) {
	result = &api.ContractInfo{}
	err = ci.cc.Call(&result, "contract_getContractInfo", addr)
	return
}

func (ci contractApi) GetGovernanceState() ([]*abi.SbpInfo, []*types.ConsensusGroupInfo, error) {
	result := &struct {
		SbpInfos   []*abi.SbpInfo
		GroupInfos []*types.ConsensusGroupInfo
	}{}
	err := ci.cc.Call(&result, "contract_getGovernanceState")
	if err != nil {
		return nil, nil, err
	}
	return result.SbpInfos, result.GroupInfos, err
}
