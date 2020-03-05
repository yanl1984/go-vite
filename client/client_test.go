package client

import (
	"fmt"
	"math/big"
	"os/user"
	"path"
	"testing"

	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/ledger"
	"github.com/vitelabs/go-vite/rpcapi/api"
	"github.com/vitelabs/go-vite/wallet"
	"github.com/vitelabs/go-vite/wallet/entropystore"
)

var WalletDir string

func init() {
	current, _ := user.Current()
	home := current.HomeDir
	WalletDir = path.Join(home, "Library/GVite/devdata/wallet")
}

var Wallet2 *entropystore.Manager

func PreTest() error {
	w := wallet.New(&wallet.Config{
		DataDir:        WalletDir,
		MaxSearchIndex: 100000,
	})
	w.Start()

	w2, err := w.RecoverEntropyStoreFromMnemonic("warfare woman sustain earth alarm actress street solution dirt exact embark cover ripple luxury unhappy enable hood orphan unique what motor plug level bike", "123456")
	if err != nil {
		fmt.Errorf("wallet error, %+v", err)
		return err
	}
	err = w2.Unlock("123456")
	if err != nil {

		fmt.Errorf("wallet error, %+v", err)
		return err
	}

	Wallet2 = w2
	return nil
}

func TestWallet2(t *testing.T) {
	w := wallet.New(&wallet.Config{
		DataDir:        WalletDir,
		MaxSearchIndex: 100000,
	})
	w.Start()

	mnemonic, _, err := w.NewMnemonicAndEntropyStore("123456")
	if err != nil {
		panic(err)
	}
	t.Log(mnemonic)
}

func TestWallet3(t *testing.T) {
	w := wallet.New(&wallet.Config{
		DataDir:        WalletDir,
		MaxSearchIndex: 100000,
	})
	w.Start()

	em, err := w.RecoverEntropyStoreFromMnemonic("warfare woman sustain earth alarm actress street solution dirt exact embark cover ripple luxury unhappy enable hood orphan unique what motor plug level bike", "123456")
	if err != nil {
		panic(err)
	}
	em.Unlock("123456")
	_, key, err := em.DeriveForIndexPath(0)
	if err != nil {
		panic(err)
	}
	address, err := key.Address()
	if err != nil {
		panic(err)
	}
	t.Log(address)
}

func TestWallet(t *testing.T) {
	err := PreTest()
	if err != nil {
		t.Error(err)
		t.FailNow()
	}

	/**
	vite_165a295e214421ef1276e79990533953e901291d29b2d4851f
	vite_2ca3c5f1f18b38f865eb47196027ae0c50d0c21e67774abdda
	vite_e7e7fd6c532d38d0a8c1158ea89d41f7ffddeef3b6e11309b9
	vite_fcdb46a9ce7c4fd1d3321636e707660255d70e39062cd16460
	vite_78290408d7ec2293e2315cb7d98260629ede76882333e77161
	vite_3dfa8bd841bc4ed351953a12f4edbd8364ed388908136f50a1
	vite_73c6b08e401608bca17272e7c59508f2e549c221ae7efccd53
	vite_9d2dafb40aec2d287fa660e879a65b108cae355bf85e6d9ae6
	vite_260033138517d251cfa3a907e5a4f0c673d656909108e1c832
	vite_8d5de7117bbf8c1fb911ba68759b5c34dea7f63987771662ba
	*/
	t.Log("----------------------Wallet2----------------------")
	for i := uint32(0); i < 10; i++ {
		_, key, err := Wallet2.DeriveForIndexPath(i)
		if err != nil {
			t.Error(err)
			return
		}
		addr, _ := key.Address()
		fmt.Println(addr)
	}
	t.Log("----------------------Wallet2----------------------")
}

func TestClient_SubmitRequestTx(t *testing.T) {
	if err := PreTest(); err != nil {
		t.Error(err)
		t.FailNow()
	}
	rpc, err := NewRpcClient(RawUrl)
	if err != nil {
		t.Error(err)
		return
	}

	client, e := NewClient(rpc)
	if e != nil {
		t.Error(e)
		return
	}
	self, err := types.HexToAddress("vite_ab24ef68b84e642c0ddca06beec81c9acb1977bbd7da27a87a")
	if err != nil {
		t.Error(err)
		return
	}
	to, err := types.HexToAddress("vite_021b3bae0d06bb7619c146da2580b4f5bc5ec8421f49fa9c56")
	if err != nil {
		t.Error(err)
		return
	}

	block, err := client.BuildNormalRequestBlock(RequestTxParams{
		ToAddr:   to,
		SelfAddr: self,
		Amount:   big.NewInt(10000),
		TokenId:  ledger.ViteTokenId,
		Data:     []byte("hello pow"),
	}, nil)
	if err != nil {
		t.Fatal(err)
	}

	err = client.SignData(Wallet2, block)
	if err != nil {
		t.Fatal(err)
	}

	err = rpc.SendRawTx(block)
	if err != nil {
		t.Fatal(err)
	}

	t.Log("submit request tx success.", block.Hash, block.Height)
}

func TestClient_CreateContract(t *testing.T) {
	if err := PreTest(); err != nil {
		t.Error(err)
		t.FailNow()
	}
	rpc, err := NewRpcClient(RawUrl)
	if err != nil {
		t.Error(err)
		return
	}

	client, e := NewClient(rpc)
	if e != nil {
		t.Error(e)
		return
	}
	self, err := types.HexToAddress("vite_021b3bae0d06bb7619c146da2580b4f5bc5ec8421f49fa9c56")
	if err != nil {
		t.Error(err)
		return
	}
	definition := `[{"constant":false,"inputs":[{"name":"data","type":"bytes32"},{"name":"t","type":"int8"}],"name":"Store","outputs":[],"payable":false,"stateMutability":"nonpayable","type":"function"},{"constant":true,"inputs":[{"name":"data","type":"bytes32"}],"name":"getData","outputs":[{"name":"","type":"int8"}],"payable":false,"stateMutability":"view","type":"offchain"},{"constant":true,"inputs":[{"name":"t","type":"int8"}],"name":"getCnt","outputs":[{"name":"","type":"int32"}],"payable":false,"stateMutability":"view","type":"offchain"},{"inputs":[],"payable":false,"stateMutability":"nonpayable","type":"constructor"},{"anonymous":false,"inputs":[{"indexed":false,"name":"fromhash","type":"bytes32"},{"indexed":true,"name":"data","type":"bytes32"},{"indexed":false,"name":"t","type":"int8"}],"name":"_store","type":"event"}]`
	code := `608060405234801561001057600080fd5b5033600260006101000a81548174ffffffffffffffffffffffffffffffffffffffffff021916908374ffffffffffffffffffffffffffffffffffffffffff1602179055506101d8806100636000396000f3fe608060405260043610610041576000357c0100000000000000000000000000000000000000000000000000000000900463ffffffff1680630c67323614610046575b600080fd5b34801561005257600080fd5b5061008c6004803603604081101561006957600080fd5b8101908080359060200190929190803560000b906020019092919050505061008e565b005b60008160000b1315156100a057600080fd5b600080600084815260200190815260200160002160009054906101000a900460000b60000b1415156100d157600080fd5b60018060008360000b60000b815260200190815260200160002160009054906101000a900460030b01600160008360000b60000b815260200190815260200160002160006101000a81548163ffffffff021916908360030b63ffffffff1602179055508060008084815260200190815260200160002160006101000a81548160ff021916908360000b60ff160217905550817f740083ee4ad1d06c2bb11dceca13ff405c05929637a1efe89e2af61a88988f1a4983604051808381526020018260000b60000b81526020019250505060405180910390a2505056fea165627a7a723058209610fc9c352f9a73ff07a9f30e561cf6efabfdb5d4780d5e66e4debb420171630029`
	block, err := client.BuildRequestCreateContractBlock(RequestCreateContractParams{
		SelfAddr: self,
		abiStr:   definition,
		metaParams: api.CreateContractDataParam{
			Gid:         types.DELEGATE_GID,
			ConfirmTime: 0,
			SeedCount:   0,
			QuotaRatio:  10,
			HexCode:     code,
		},
	}, nil)
	if err != nil {
		t.Fatal(err)
	}

	err = client.SignData(Wallet2, block)
	if err != nil {
		t.Fatal(err)
	}

	err = rpc.SendRawTx(block)
	if err != nil {
		t.Fatal(err)
	}

	t.Log("submit request tx success.", block.Hash, block.Height)
}

func TestClient_CallContract(t *testing.T) {
	if err := PreTest(); err != nil {
		t.Error(err)
		t.FailNow()
	}
	rpc, err := NewRpcClient(RawUrl)
	if err != nil {
		t.Error(err)
		return
	}

	client, e := NewClient(rpc)
	if e != nil {
		t.Error(e)
		return
	}
	self, err := types.HexToAddress("vite_165a295e214421ef1276e79990533953e901291d29b2d4851f")
	if err != nil {
		t.Error(err)
		return
	}
	contract, err := types.HexToAddress("vite_350dde5a5405c89d5b78f07a9f7c8a6284b4e0a874580223e6")
	if err != nil {
		t.Error(err)
		return
	}

	abi := `[{"constant":false,"inputs":[{"name":"data","type":"bytes32"},{"name":"t","type":"int8"}],"name":"Store","outputs":[],"payable":false,"stateMutability":"nonpayable","type":"function"},{"constant":true,"inputs":[{"name":"data","type":"bytes32"}],"name":"getData","outputs":[{"name":"","type":"int8"}],"payable":false,"stateMutability":"view","type":"offchain"},{"constant":true,"inputs":[{"name":"t","type":"int8"}],"name":"getCnt","outputs":[{"name":"","type":"int32"}],"payable":false,"stateMutability":"view","type":"offchain"},{"inputs":[],"payable":false,"stateMutability":"nonpayable","type":"constructor"},{"anonymous":false,"inputs":[{"indexed":false,"name":"fromhash","type":"bytes32"},{"indexed":true,"name":"data","type":"bytes32"},{"indexed":false,"name":"t","type":"int8"}],"name":"_store","type":"event"}]`

	abiCli, err := GetAbiCli(rpc, abi, contract)
	data, err := abiCli.BuildCallMethodData("Store", types.HexToHashPanic("7465e0b455e06de36e8c440e15df14b667ad4289684a95f8672c71b6cfa8d0da"), int8(1))
	if err != nil {
		t.Error(err)
		return
	}

	block, err := client.BuildNormalRequestBlock(RequestTxParams{
		ToAddr:   contract,
		SelfAddr: self,
		Amount:   big.NewInt(0),
		TokenId:  ledger.ViteTokenId,
		Data:     data,
	}, nil)
	if err != nil {
		t.Fatal(err)
	}

	err = client.SignData(Wallet2, block)
	if err != nil {
		t.Fatal(err)
	}

	err = rpc.SendRawTx(block)
	if err != nil {
		t.Fatal(err)
	}

	t.Log("submit request tx success.", block.Hash, block.Height)
}

func TestClient_SubmitResponseTx(t *testing.T) {
	PreTest()
	to, err := types.HexToAddress("vite_2ca3c5f1f18b38f865eb47196027ae0c50d0c21e67774abdda")
	if err != nil {
		t.Error(err)
		return
	}

	t.Log(to)
	rpc, err := NewRpcClient(RawUrl)
	if err != nil {
		t.Error(err)
		return
	}

	client, e := NewClient(rpc)
	if e != nil {
		t.Error(e)
		return
	}

	requestHash := types.HexToHashPanic("1058ac419ffa5f8cfa8bf3a19e8f4cf870ec0956025dc4ebc17793344fd2e67e")
	t.Log("receive request.", requestHash)
	block, err := client.BuildResponseBlock(ResponseTxParams{
		SelfAddr:    to,
		RequestHash: requestHash,
	}, nil)

	if err != nil {
		t.Fatal(err)
	}

	t.Log("receive request.", requestHash, block.Amount)

	err = client.SignData(Wallet2, block)
	if err != nil {
		t.Fatal(err)
	}
	err = rpc.SendRawTx(block)

	if err != nil {
		t.Fatal(err)
	}
}

func TestClient_QueryOnroad(t *testing.T) {
	rpc, err := NewRpcClient(RawUrl)
	if err != nil {
		t.Fatal(err)
	}

	addr, err := types.HexToAddress("vite_2ca3c5f1f18b38f865eb47196027ae0c50d0c21e67774abdda")
	if err != nil {
		t.Fatal(err)
	}

	blocks, err := rpc.GetOnroadBlocksByAddress(addr, 0, 100)
	if err != nil {
		t.Fatal(err)
	}

	if len(blocks) > 0 {
		for _, v := range blocks {
			t.Log(v.Height, v.Address, v.ToAddress, *v.Amount, v.Hash)
		}
	}
}

func TestClient_GetBalanceAll(t *testing.T) {
	rpc, err := NewRpcClient(RawUrl)
	if err != nil {
		t.Error(err)
		return
	}

	client, e := NewClient(rpc)
	if e != nil {
		t.Error(e)
		return
	}

	addr, err := types.HexToAddress("vite_ab24ef68b84e642c0ddca06beec81c9acb1977bbd7da27a87a")
	if err != nil {
		t.Error(err)
		return
	}

	balance, onroad, err := client.GetBalanceAll(addr)
	if err != nil {
		t.Fatal(err)
	}
	for k, v := range balance.TokenBalanceInfoMap {
		t.Log(k, "balance", v.TokenInfo.TokenSymbol, v.TotalAmount)
	}
	for k, v := range onroad.TokenBalanceInfoMap {
		t.Log(k, "onroad", v.TokenInfo.TokenSymbol, v.TotalAmount)
	}
}

func TestClient_GetBalance(t *testing.T) {
	rpc, err := NewRpcClient(RawUrl)
	if err != nil {
		t.Error(err)
		return
	}

	client, e := NewClient(rpc)
	if e != nil {
		t.Error(e)
		return
	}

	addr, err := types.HexToAddress("vite_1b351d987dd194ea7f8146a45e7b2625c1d9d483505fc524e8")
	if err != nil {
		t.Error(err)
		return
	}

	balance, onroad, err := client.GetBalance(addr, ledger.ViteTokenId)
	if err != nil {
		t.Fatal(err)
	}
	t.Log("balance", balance.String())
	t.Log("onroad", onroad.String())
}

func TestName(t *testing.T) {
	hash, _ := types.HexToHash("")
	t.Log(hash)
}
