package client

import (
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/crypto/ed25519"
	"github.com/vitelabs/go-vite/ledger"
	"github.com/vitelabs/go-vite/rpcapi/api"
	aabi "github.com/vitelabs/go-vite/vm/abi"
	"github.com/vitelabs/go-vite/vm/contracts/abi"
	"math/big"
)

type GovernanceClient interface {
	Register(param abi.ParamRegister, prev *ledger.HashHeight) (*api.AccountBlock, error)
	Revoke(param abi.ParamRevoke, prev *ledger.HashHeight) (*api.AccountBlock, error)
	Vote(param abi.ParamVote, prev *ledger.HashHeight) (*api.AccountBlock, error)
	UpdateProducingAddress(param abi.ParamUpdateProducingAddress, prev *ledger.HashHeight) (*api.AccountBlock, error)
}

type governanceCli struct {
	rpcCli RpcClient
	cli    Client

	contractAbi  aabi.ABIContract
	contractAddr types.Address
	key          ed25519.PrivateKey

	self types.Address
}

func (c governanceCli) UpdateProducingAddress(param abi.ParamUpdateProducingAddress, prev *ledger.HashHeight) (*api.AccountBlock, error) {
	return c.callGovernanceMethod(abi.MethodNameUpdateBlockProducingAddress, param, prev)
}

func (c governanceCli) Register(param abi.ParamRegister, prev *ledger.HashHeight) (*api.AccountBlock, error) {
	return c.callGovernanceMethod(abi.MethodNameRegister, param, prev)
}

func (c governanceCli) Revoke(param abi.ParamRevoke, prev *ledger.HashHeight) (*api.AccountBlock, error) {
	return c.callGovernanceMethod(abi.MethodNameRevoke, param, prev)
}

func (c governanceCli) Vote(param abi.ParamVote, prev *ledger.HashHeight) (*api.AccountBlock, error) {
	return c.callGovernanceMethod(abi.MethodNameVote, param, prev)
}

func (c governanceCli) callGovernanceMethod(methodName string, param abi.Serializable, prev *ledger.HashHeight) (*api.AccountBlock, error) {
	data, err := c.contractAbi.PackMethod(methodName, param.Serialize()...)
	if err != nil {
		return nil, err
	}
	block, err := c.cli.BuildNormalRequestBlock(RequestTxParams{
		ToAddr:   types.AddressGovernance,
		SelfAddr: c.self,
		Amount:   big.NewInt(0),
		TokenId:  ledger.ViteTokenId,
		Data:     data,
	}, prev)
	if err != nil {
		return nil, err
	}
	err = c.cli.SignDataWithPriKey2(c.key, block)
	if err != nil {
		return nil, err
	}

	err = c.rpcCli.SendRawTx(block)
	return block, err
}

func NewGovernanceClient(rpcCli RpcClient, selfAddress types.Address, key ed25519.PrivateKey) GovernanceClient {
	cli, _ := NewClient(rpcCli)

	g := governanceCli{
		rpcCli:       rpcCli,
		contractAbi:  abi.ABIGovernance,
		contractAddr: types.AddressGovernance,
		self:         selfAddress,
		cli:          cli,
		key:          key,
	}
	return g
}
