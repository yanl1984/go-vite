package client

import (
	"testing"

	"github.com/vitelabs/go-vite/common/types"
)

func TestAbiCli_CallOffChain(t *testing.T) {
	rpc, err := NewRpcClient(RawUrl)
	if err != nil {
		t.Error(err)
		return
	}

	abi := `[{"constant":false,"inputs":[{"name":"data","type":"bytes32"},{"name":"t","type":"int8"}],"name":"Store","outputs":[],"payable":false,"stateMutability":"nonpayable","type":"function"},{"constant":true,"inputs":[{"name":"data","type":"bytes32"}],"name":"getData","outputs":[{"name":"","type":"int8"}],"payable":false,"stateMutability":"view","type":"offchain"},{"constant":true,"inputs":[{"name":"t","type":"int8"}],"name":"getCnt","outputs":[{"name":"","type":"int32"}],"payable":false,"stateMutability":"view","type":"offchain"},{"inputs":[],"payable":false,"stateMutability":"nonpayable","type":"constructor"},{"anonymous":false,"inputs":[{"indexed":false,"name":"fromhash","type":"bytes32"},{"indexed":true,"name":"data","type":"bytes32"},{"indexed":false,"name":"t","type":"int8"}],"name":"_store","type":"event"}]`
	offchainCode := `608060405260043610610050576000357c0100000000000000000000000000000000000000000000000000000000900463ffffffff16806341035cde14610052578063dce983421461009f57610050565b005b610083600480360360208110156100695760006000fd5b8101908080356000191690602001909291905050506100eb565b604051808260000b60000b815260200191505060405180910390f35b6100cf600480360360208110156100b65760006000fd5b81019080803560000b9060200190929190505050610125565b604051808260030b60030b815260200191505060405180910390f35b600060006000506000836000191660001916815260200190815260200160002160009054906101000a900460000b9050610120565b919050565b6000600160005060008360000b60000b815260200190815260200160002160009054906101000a900460030b9050610158565b91905056fea165627a7a723058209610fc9c352f9a73ff07a9f30e561cf6efabfdb5d4780d5e66e4debb420171630029`
	contract := types.HexToAddressPanic("vite_350dde5a5405c89d5b78f07a9f7c8a6284b4e0a874580223e6")

	abiCli, err := GetAbiCli(rpc, abi, contract)
	if err != nil {
		t.Fatal(err)
	}

	result, err := abiCli.CallOffChain(offchainCode, "getData", types.HexToHashPanic("b36eba71b204fdc996a9a8e7efeb2adbbe6806e9d0774cdd08f9c9cfe317eaf6"))
	if err != nil {
		t.Fatal(err)
	}
	for k, v := range result {
		t.Log(k, v)
	}

	result, err = abiCli.CallOffChain(offchainCode, "getCnt", int8(1))
	if err != nil {
		t.Fatal(err)
	}
	for k, v := range result {
		t.Log(k, v)
	}
}
