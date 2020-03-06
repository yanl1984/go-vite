package client

import (
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/crypto/ed25519"
	"github.com/vitelabs/go-vite/vm/contracts/abi"
	"github.com/vitelabs/go-vite/vm/util"
	"github.com/vitelabs/go-vite/wallet"
	"gotest.tools/assert"
	"testing"
)

var rawUrlForGovernanceTest = "http://127.0.0.1:48101"

var rpcCli RpcClient
var cli Client

var (
	s3       = types.HexToAddressPanic("vite_40996a2ba285ad38930e09a43ee1bd0d84f756f65318e8073a")
	s3Key, _ = ed25519.HexToPrivateKey("e52012b8a25a5abaa54cc0b1ae4fb8f854dfe34d42b46afc6e97d5a0127c16d03bf5f8a221db5ff0abe3a0382a93df22fce8179cec0b498799ac38515ac12752")
	s2       = types.HexToAddressPanic("vite_ce18b99b46c70c8e6bf34177d0c5db956a8c3ea7040a1c1e25")
	s2Key, _ = ed25519.HexToPrivateKey("52c030e64ff26c8bbf9d67f049450ae5b66ac945aca1b98b3ef2164b2ab8441de0de77ffdc2719eb1d8e89139da9747bd413bfe59781c43fc078bb37d8cbd77a")
	s1       = types.HexToAddressPanic("vite_360232b0378111b122685a15e612143dc9a89cfa7e803f4b5a")
	s1Key, _ = ed25519.HexToPrivateKey("1407fc559c4809c47e0569826e71580317de78e4fcaba3c9c2c8e3ef62d42c473fc5224e59433bff4f48c83c0eb4edea0e4c42ea697e04cdec717d03e50d5200")
)

func before() {
	var err error
	rpcCli, err = NewRpcClient(rawUrlForGovernanceTest)
	util.AssertNil(err)
	cli, err = NewClient(rpcCli)
	util.AssertNil(err)

}

/**
s3:
0:vite_40996a2ba285ad38930e09a43ee1bd0d84f756f65318e8073a : e52012b8a25a5abaa54cc0b1ae4fb8f854dfe34d42b46afc6e97d5a0127c16d03bf5f8a221db5ff0abe3a0382a93df22fce8179cec0b498799ac38515ac12752
1:vite_cdedd375b6001c2081e829d80f1b336c0564fb96f1e2d2e93d
2:vite_06501648d1d4f6e3f55b6e54d5d4cafa1d22567067e829d860


s1:
0:vite_360232b0378111b122685a15e612143dc9a89cfa7e803f4b5a : 1407fc559c4809c47e0569826e71580317de78e4fcaba3c9c2c8e3ef62d42c473fc5224e59433bff4f48c83c0eb4edea0e4c42ea697e04cdec717d03e50d5200

s2
0:vite_ce18b99b46c70c8e6bf34177d0c5db956a8c3ea7040a1c1e25 : 52c030e64ff26c8bbf9d67f049450ae5b66ac945aca1b98b3ef2164b2ab8441de0de77ffdc2719eb1d8e89139da9747bd413bfe59781c43fc078bb37d8cbd77a
1:vite_999fb65a4553fcec4f68e7262a8014678ea0f594c49d843b83
*/
func walletInit() *wallet.Manager {
	w := wallet.New(&wallet.Config{
		DataDir:        "/Users/jie/go/src/github.com/vitelabs/cluster/ledger_datas/wallet",
		MaxSearchIndex: 100000,
	})
	w.Start()

	return w
}

func Test_GetState(t *testing.T) {
	before()
	state, groupInfos, err := rpcCli.GetGovernanceState()
	assert.NilError(t, err)
	t.Log(util.JsonString(state))
	t.Log(util.JsonString(groupInfos))
}

func Test_Register(t *testing.T) {
	before()
	proposer := s3
	key := s3Key
	proposerName := "s3"

	param := abi.ParamRegister{
		SbpName:               "s5",
		BlockProducingAddress: types.HexToAddressPanic("vite_cdedd375b6001c2081e829d80f1b336c0564fb96f1e2d2e93d"),
		OwnerAddress:          proposer,
		ProposerSbpName:       proposerName,
	}
	governanceClient := NewGovernanceClient(rpcCli, proposer, key)
	_, err := governanceClient.Register(param, nil)

	assert.NilError(t, err)

}

func Test_Revoke(t *testing.T) {
	before()

	param := abi.ParamRevoke{
		SbpName:         "s5",
		ProposerSbpName: "s2",
	}

	governanceClient := NewGovernanceClient(rpcCli, s2, s2Key)
	_, err := governanceClient.Revoke(param, nil)

	assert.NilError(t, err)
}

func Test_Vote(t *testing.T) {
	before()
	{
		param := abi.ParamVote{
			SbpName:         "s5",
			VoteType:        1,
			Approval: true,
			ProposerSbpName: "s5",
		}
		governanceClient := NewGovernanceClient(rpcCli, s3, s3Key)
		_, err := governanceClient.Vote(param, nil)
		assert.NilError(t, err)
	}
	{
		param := abi.ParamVote{
			SbpName:         "s5",
			VoteType:        1,
			Approval:        true,
			ProposerSbpName: "s2",
		}
		governanceClient := NewGovernanceClient(rpcCli, s2, s2Key)
		_, err := governanceClient.Vote(param, nil)
		assert.NilError(t, err)
	}
}

func Test_UpdateProducingAddress(t *testing.T) {
	before()
	{
		param := abi.ParamUpdateProducingAddress{
			SbpName:               "s3",
			BlockProducingAddress: types.HexToAddressPanic("vite_999fb65a4553fcec4f68e7262a8014678ea0f594c49d843b83"),
		}
		governanceClient := NewGovernanceClient(rpcCli, s3, s3Key)
		_, err := governanceClient.UpdateProducingAddress(param, nil)
		assert.NilError(t, err)
	}
}


func Test_PriKey(t *testing.T) {
	w := walletInit()
	files := w.ListAllEntropyFiles()
	for _, file := range files {

		manager, err := w.GetEntropyStoreManager(file)
		manager.Unlock("123456")
		assert.NilError(t, err)
		{
			_, key, err := manager.DeriveForIndexPath(0)
			assert.NilError(t, err)
			t.Log(key.Address())

			privateKey, err := key.PrivateKey()
			assert.NilError(t, err)
			t.Log(privateKey.Hex())
		}
		{
			_, key, err := manager.DeriveForIndexPath(1)
			assert.NilError(t, err)
			address, _ := key.Address()
			t.Log(manager.GetPrimaryAddr(), address)
		}
	}

	t.Log(types.AddressGovernance)
}
