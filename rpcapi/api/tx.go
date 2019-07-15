package api

import (
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"math/big"
	"math/rand"
	"strconv"
	"time"

	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/consensus"
	"github.com/vitelabs/go-vite/crypto/ed25519"
	"github.com/vitelabs/go-vite/generator"
	"github.com/vitelabs/go-vite/ledger"
	"github.com/vitelabs/go-vite/verifier"
	"github.com/vitelabs/go-vite/vite"
	"github.com/vitelabs/go-vite/vite/net"
	"github.com/vitelabs/go-vite/vm"
	"github.com/vitelabs/go-vite/vm/quota"
	"github.com/vitelabs/go-vite/vm/util"
	"github.com/vitelabs/go-vite/vm_db"
	"go.uber.org/atomic"
)

type Tx struct {
	vite   *vite.Vite
	autoTx bool
}

func NewTxApi(vite *vite.Vite) *Tx {
	tx := &Tx{
		vite:   vite,
		autoTx: true,
	}
	tx.autoSend()
	return tx
}

func (t Tx) SendRawTx(block *AccountBlock) error {
	log.Info("SendRawTx")
	if block == nil {
		return errors.New("empty block")
	}
	lb, err := block.RpcToLedgerBlock()
	if err != nil {
		return err
	}

	latestSb := t.vite.Chain().GetLatestSnapshotBlock()
	if latestSb == nil {
		return errors.New("failed to get latest snapshotBlock")
	}
	nowTime := time.Now()
	if nowTime.Before(latestSb.Timestamp.Add(-10*time.Minute)) || nowTime.After(latestSb.Timestamp.Add(10*time.Minute)) {
		return IllegalNodeTime
	}

	v := verifier.NewVerifier(nil, verifier.NewAccountVerifier(t.vite.Chain(), t.vite.Consensus()))
	err = v.VerifyNetAb(lb)
	if err != nil {
		return err
	}
	result, err := v.VerifyRPCAccBlock(lb, latestSb)
	if err != nil {
		return err
	}

	if result != nil {
		return t.vite.Pool().AddDirectAccountBlock(result.AccountBlock.AccountAddress, result)
	} else {
		return errors.New("generator gen an empty block")
	}
}

func (t Tx) checkSnapshotValid(latestSb *ledger.SnapshotBlock) error {
	nowTime := time.Now()
	if nowTime.Before(latestSb.Timestamp.Add(-10*time.Minute)) || nowTime.After(latestSb.Timestamp.Add(10*time.Minute)) {
		return IllegalNodeTime
	}
	return nil
}

func (t Tx) SendTxWithPrivateKey(param SendTxWithPrivateKeyParam) (*AccountBlock, error) {

	if param.Amount == nil {
		return nil, errors.New("amount is nil")
	}

	if param.SelfAddr == nil {
		return nil, errors.New("selfAddr is nil")
	}

	if param.ToAddr == nil && param.BlockType != ledger.BlockTypeSendCreate {
		return nil, errors.New("toAddr is nil")
	}

	if param.PrivateKey == nil {
		return nil, errors.New("privateKey is nil")
	}

	var d *big.Int = nil
	if param.Difficulty != nil {
		t, ok := new(big.Int).SetString(*param.Difficulty, 10)
		if !ok {
			return nil, ErrStrToBigInt
		}
		d = t
	}

	amount, ok := new(big.Int).SetString(*param.Amount, 10)
	if !ok {
		return nil, ErrStrToBigInt
	}

	var blockType byte
	if param.BlockType > 0 {
		blockType = param.BlockType
	} else {
		blockType = ledger.BlockTypeSendCall
	}

	msg := &generator.IncomingMessage{
		BlockType:      blockType,
		AccountAddress: *param.SelfAddr,
		ToAddress:      param.ToAddr,
		TokenId:        &param.TokenTypeId,
		Amount:         amount,
		Fee:            nil,
		Data:           param.Data,
		Difficulty:     d,
	}

	addrState, err := generator.GetAddressStateForGenerator(t.vite.Chain(), &msg.AccountAddress)
	if err != nil || addrState == nil {
		return nil, errors.New(fmt.Sprintf("failed to get addr state for generator, err:%v", err))
	}
	g, e := generator.NewGenerator(t.vite.Chain(), t.vite.Consensus(), msg.AccountAddress, addrState.LatestSnapshotHash, addrState.LatestAccountHash)
	if e != nil {
		return nil, e
	}
	result, e := g.GenerateWithMessage(msg, &msg.AccountAddress, func(addr types.Address, data []byte) (signedData, pubkey []byte, err error) {
		var privkey ed25519.PrivateKey
		privkey, e := ed25519.HexToPrivateKey(*param.PrivateKey)
		if e != nil {
			return nil, nil, e
		}
		signData := ed25519.Sign(privkey, data)
		pubkey = privkey.PubByte()
		return signData, pubkey, nil
	})
	if e != nil {
		return nil, e
	}
	if result.Err != nil {
		return nil, result.Err
	}
	if result.VMBlock != nil {
		if err := t.vite.Pool().AddDirectAccountBlock(msg.AccountAddress, result.VMBlock); err != nil {
			return nil, err
		}
		return ledgerToRpcBlock(t.vite.Chain(), result.VMBlock.AccountBlock)
	} else {
		return nil, errors.New("generator gen an empty block")
	}
}

type SendTxWithPrivateKeyParam struct {
	SelfAddr     *types.Address    `json:"selfAddr"`
	ToAddr       *types.Address    `json:"toAddr"`
	TokenTypeId  types.TokenTypeId `json:"tokenTypeId"`
	PrivateKey   *string           `json:"privateKey"` //hex16
	Amount       *string           `json:"amount"`
	Data         []byte            `json:"data"` //base64
	Difficulty   *string           `json:"difficulty,omitempty"`
	PreBlockHash *types.Hash       `json:"preBlockHash,omitempty"`
	BlockType    byte              `json:"blockType"`
}

type CalcPoWDifficultyParam struct {
	SelfAddr types.Address `json:"selfAddr"`
	PrevHash types.Hash    `json:"prevHash"`

	BlockType byte           `json:"blockType"`
	ToAddr    *types.Address `json:"toAddr"`
	Data      []byte         `json:"data"`

	UsePledgeQuota bool `json:"usePledgeQuota"`

	Multiple uint16 `json:"multipleOnCongestion"`
}

var multipleDivision = big.NewInt(10)

type CalcPoWDifficultyResult struct {
	QuotaRequired uint64 `json:"quota"`
	Difficulty    string `json:"difficulty"`
	Qc            string `json:"qc"`
	IsCongestion  bool   `json:"isCongestion"`
}

func (t Tx) CalcPoWDifficulty(param CalcPoWDifficultyParam) (result *CalcPoWDifficultyResult, err error) {
	latestBlock, err := t.vite.Chain().GetLatestAccountBlock(param.SelfAddr)
	if err != nil {
		return nil, err
	}
	if (latestBlock == nil && !param.PrevHash.IsZero()) ||
		(latestBlock != nil && latestBlock.Hash != param.PrevHash) {
		return nil, util.ErrChainForked
	}
	// get quota required
	block := &ledger.AccountBlock{
		BlockType:      param.BlockType,
		AccountAddress: param.SelfAddr,
		PrevHash:       param.PrevHash,
		Data:           param.Data,
	}
	if param.ToAddr != nil {
		block.ToAddress = *param.ToAddr
	} else if param.BlockType == ledger.BlockTypeSendCall {
		return nil, errors.New("toAddr is nil")
	}
	sb := t.vite.Chain().GetLatestSnapshotBlock()
	db, err := vm_db.NewVmDb(t.vite.Chain(), &param.SelfAddr, &sb.Hash, &param.PrevHash)
	if err != nil {
		return nil, err
	}
	quotaRequired, err := vm.GasRequiredForBlock(db, block, util.GasTableByHeight(sb.Height), sb.Height)
	if err != nil {
		return nil, err
	}

	qc, _, isCongestion := quota.CalcQc(db, sb.Height)
	qcStr := strconv.FormatFloat(qc, 'g', 18, 64)

	// get current quota
	var pledgeAmount *big.Int
	var q types.Quota
	if param.UsePledgeQuota {
		pledgeAmount, err = t.vite.Chain().GetPledgeBeneficialAmount(param.SelfAddr)
		if err != nil {
			return nil, err
		}
		q, err := quota.GetPledgeQuota(db, param.SelfAddr, pledgeAmount, sb.Height)
		if err != nil {
			return nil, err
		}
		if q.Current() >= quotaRequired {
			return &CalcPoWDifficultyResult{quotaRequired, "", qcStr, isCongestion}, nil
		}
	} else {
		pledgeAmount = big.NewInt(0)
		q = types.NewQuota(0, 0, 0, 0, false)
	}
	// calc difficulty if current quota is not enough
	canPoW := quota.CanPoW(db, block.AccountAddress)
	if !canPoW {
		return nil, util.ErrCalcPoWTwice
	}
	d, err := quota.CalcPoWDifficulty(db, quotaRequired, q, sb.Height)
	if err != nil {
		return nil, err
	}
	if isCongestion && param.Multiple > uint16(multipleDivision.Uint64()) {
		d.Mul(d, multipleDivision)
		d.Div(d, big.NewInt(int64(param.Multiple)))
	}
	return &CalcPoWDifficultyResult{quotaRequired, d.String(), qcStr, isCongestion}, nil
}

func (tx Tx) autoSend() {
	if !tx.autoTx {
		return
	}
	var InitContractAddr = []string{
		"vite_def8c040e89f6039391fc43186c17f4ebb3c9aabf29041591a",
		"vite_b64dd6e7fb8fdf5f554627fa01f5ed01b734cf06a5d1a16bfa",
		"vite_5b2d01d8ea831131a9f06d53eb580445214750168de6824beb",
		"vite_54f476e6dfd071709eeed7b99ed64d76d9c63239efdab3bb24",
		"vite_c617004ffe880f5e1bf6d3845e59c561b46dad4c96b9ddb20a",
		"vite_765710426e441830d65abdfeb8ce5722802918cd1045f15520",
		"vite_14894b06de6e16d2f1aa5d8260a2f78275e4b7795062c1fc89",
		"vite_259a86611e4804025f2e6788a4c5f9347e4d72d00faaa316d3",
		"vite_56493c94d37cb4c5b58b906f924ccd1bc6ff50e0d1fa6cffee",
		"vite_bb6364c4eaf7286eefa90af616b621d49062b990879f250b1a",
		"vite_34a4417a8b95f977fad794372b6ac46f3e7140562d9433a946",
		"vite_dff6d616d81bae5358a2db10ccb1e01a1b5e5f1a11e404486a",
		"vite_eced4a6e518768bcbb8cebb63094e742c746d7575deb5ea192",
		"vite_0899312577c9385402f4c5ccb7cf4eda959238a8866093a537",
		"vite_c2b5f3c0401f8bebe828fadb349ea6126eaa708430242beed9",
		"vite_e8f5cfb0db2607a2c8304e4dbfedc3d1e7f9a4d670909d987f",
		"vite_f2fa075ec03c0b8c79cb523a0e5dfdab793ee6a7f82f138311",
		"vite_1e8a7df3c21cf2db81ef175aa406c0b57da806ab6f10f2108e",
		"vite_1231ebd5d95e420826ff14683a81c54979b9cc9d38a4a8064b",
		"vite_66f8c1e61f20d4bad36c9cf059a5187f05882ba8aa1481f668",
		"vite_5a72e512632f96126faca2c5babdb670628c4aac1c08666f2c",
		"vite_2c8e711af8fbea93e67059d0c50ed0ab74bd202208d149c052",
		"vite_468d2204036e54995bf419b75c1479ce79211628540f6b5e4c",
		"vite_bc5a41ee0f42dfcc77cf04173b4c567ec5f1e6e27cb4f81525",
		"vite_0431b553f31f254962f5891fa3d8942c1b94f00e658f35ad2e",
		"vite_99fe0c6e9aa835f873ee368be9a3b8bf949b0f4109d4a462f7",
		"vite_cc342425b2b05674c030b7f2c238f36761e3c7c2d6ebf0c70a",
		"vite_16668c24b0def8134a9556a9e2c14e7a6511bd746580d4e1f0",
		"vite_bbc1e03e370d24fd3a415a44d947d4857ab69ac549378474e6",
		"vite_0254434fa9821d8ea2d8a9e1b19a1ed53e08f4eff81827bd43",
		"vite_d92212099c2e52c2408e25aaf1bc46aa07dbe8832debed82ed",
		"vite_d793ae7e370beb572d9bf1c00fc7fd2814e0e8e42d7432369a",
		"vite_29d22e783b50bcb7243a7de2a205d5cf5f03484bd97784608b",
		"vite_375673f0bc966bc1dcf8611d5da19f4a0b9f19425b5c2f0844",
		"vite_d0004f95e992b9de83f5a5f2c68acfccc1e5775dd6a628fdf5",
		"vite_f196a2b63ed6f32b73469a047515030082e5c6e28b74ec21ca",
		"vite_cbdf1dc2560304ec2735d8362b7f5aa7977c28c3ea25e68790",
		"vite_d88131d8bfda20a77e138b6a71ee48058bfe81708cde1b7495",
		"vite_0aa846b66eb12e66790ef216dcf81db3f838e143c2b9571a22",
		"vite_f1b30260d2b0ec3164126f76cd55ccb11d27bfb77e19126d97",
		"vite_09222c571b90b6f6423cf1d56413f5e9532e6341fcfcf48c7e",
		"vite_610e31f662103d8882497d89f0cff6a62a60e1631036c8e38c",
		"vite_995c7b55f19e10e69a98ee10a9bf4c86911730508184cdab12",
		"vite_84f07a481d3b8f277978197703760237eb4fc1133ff43da208",
		"vite_7e67ecc6b901cc2cca5f3816e1016e6654b83d44abfcb16556",
		"vite_276824b30754750fa4ee5faa8f1e4cd207b2a95a414d189926",
		"vite_a6c8d497e37e68f2db2eff7c2e4e8b161cbb677586c3181553",
		"vite_ea059f6c5cdd879746df8cf8696d42656e5aca68d3c840764e",
		"vite_27f2ff4ff807ab994b9d8275cf7b53b634bf0bc0546d758751",
		"vite_f9a5b8d807310d8728cc885f6ad9daca54b09bda6ebc8771e9",
		"vite_99f653b05d484aed7b87216cce817cc08cb1519c48a213a216",
		"vite_f9157ca2abdb743f8ad7b5ed3a00488c55e881b2825c09b400",
		"vite_ef194381906e4faf20330fd21498090dad5d7b72d3d2fb38c4",
		"vite_fa386c952d7e0d333a1d2a75070715b775d113ddc0a20cfb9c",
		"vite_e00c64319d0f482100b4902045444f17fa28b5f50c618acd3a",
		"vite_8b790a974bdf09ac577c38744d405de49c5026d9138f14a70f",
		"vite_8be300361fbf9ce216627385f0b59807255a919d6d508b5d37",
		"vite_d47d99c14630ee4a55fcab11ff42d9f8e3049db3e726f83115",
		"vite_595292636cd0fec8542e26b222c8bb8524e848d544ebe7de9b",
		"vite_56e893c8198a3e6d2838a863657c6337ee283df776b0f40d61",
		"vite_cabb0c9afc29fec6f5324fb1c086f94205f8015dde74d51c6c",
		"vite_5e16816d0803406649b8485052f9206cbc9d9bf50344186f1c",
		"vite_2904f22981a3ba7081f87517ec6daeb2064859ee6ed46e5712",
		"vite_28c53ecfad37e7af2402a7c49d65cf1c2111d0fc2849fd430d",
		"vite_c972e8bd67efbc1903eb9fbcd665b8d8aee1c019f9e0a0dda0",
		"vite_0215112c5a0784f4df97b99abd2c76f750118268e5f40e458c",
		"vite_059331fdb994ad8eb51841fa6ee84f9dad921cb3ef3082dd70",
		"vite_2c0fd9e46d07077ce8d9943f832f980b16d9644b861adde46e",
		"vite_4412ff23fe9a8e8cb761b290a9805c3165cf3b2199cfe31059",
		"vite_8b46b4c59c0c0155193800f1372fc692e6450c6850a63b1a5d",
		"vite_a1f88c63415344a83121a7263886e5c79b897c12154a797c64",
		"vite_6d67192f3f3b97d81970c60129e10ada0475e24aad30ae1c85",
		"vite_d16ea816d565d200f9a046ef137d17b2f0de6a01951b2597a9",
		"vite_20746d0153f666069e9bec369ad80eee4327b2efd4bf83620b",
		"vite_3493baea515231d941575a7aecb3c9cb2b0528fccd5b69fcaf",
		"vite_90bda57b930fdb7679299c91dfc8b395a748b8eb5586d9238f",
		"vite_7121fabe460a9ee61cb2ff2872fda28980a41827f6e7ffaa98",
		"vite_c23411c34722ff2274f8c123c50aba21d9992238ea0957bfc3",
		"vite_64fdb8854ba5fff4675aac017fb541faca3bb287452997316a",
		"vite_70906878ee449edf285db9266f1ae8b64d8bbee6a6df23b301",
		"vite_4fccd4945ff3dd19ee020f9b08b098cb3283d137d5376edc1e",
		"vite_35b93c9325bacf34e49f38d4c238c47e65fa8746e96aa279cb",
		"vite_3df00b7df29e137057b78a4c36fd915977583c3dd9a445d836",
		"vite_d2610aa816990bad796bf6b43174f34228c83d4079308ef32c",
		"vite_ba62a92ae667a12c588ebd91a08fdfbe0f122bf6ccdec9b135",
		"vite_800fac0252960000c62ae4c229e86a7a150be28b723f5254aa",
		"vite_b93cc01d73812ce5360f7d9fa6c9de6dfeb600c112c69f3fad",
		"vite_5b14c328cb0fc638951a244c27caa34ac6b2da0e807c29f2ea",
		"vite_9f74909131ffcd7f4c6df8afa42db43d7e4c94ebecdf7ab2a2",
		"vite_6588c292a21a717486c91ee4b8cd9c748de2fb1bca5ef4f2c5",
		"vite_828a0e7735e4a95607f5375151512b2bef850c09485fe0f7c6",
		"vite_d0702a4b89d0be30191ea746d3ed50189512993d69b416997c",
		"vite_e4b918a4ebee121826423a74dc583ba05259b278c1718a9a5d",
		"vite_e40afe3c22dda545197db0404f5ca4e6b40b444f22ac539d34",
		"vite_cce8e195b4f2e8262f8be5f9957d3d1c0ccaf77c3d86b95001",
		"vite_ea6ef3c0d11f51cdfe6ca84eedc8eeeba7abe379f9c8717074",
		"vite_8e95458b0c128c046504a4aaf7173e7b38c014ee12e1433071",
		"vite_f471a48ce1c85ac2a4842684b5de795bc94470fc4798100c5d",
		"vite_148fa9ffe5c63ec072e6021fa5a432a4f7b01501b3cc5d9505",
		"vite_1bbaac067c6cb8943166099393cc0e44956af307b93b939e8e",
	}

	if tx.vite.Producer() == nil {
		return
	}

	M := 4
	N := 0
	coinbase := tx.vite.Producer().GetCoinBase()

	manager, err := tx.vite.WalletManager().GetEntropyStoreManager(coinbase.String())
	if err != nil {
		panic(err)
	}

	var fromAddrs []types.Address
	var fromHexPrivKeys []string

	{
		for i := uint32(0); i < uint32(5); i++ {
			_, key, err := manager.DeriveForIndexPath(i)
			if err != nil {
				panic(err)
			}
			binKey, err := key.PrivateKey()
			if err != nil {
				panic(err)
			}

			pubKey, err := key.PublicKey()
			if err != nil {
				panic(err)
			}

			address, err := key.Address()
			if err != nil {
				panic(err)
			}

			fmt.Printf("%s hex public key:%s\n", address, hex.EncodeToString(pubKey))

			hexKey := hex.EncodeToString(binKey)
			fromHexPrivKeys = append(fromHexPrivKeys, hexKey)
			fromAddrs = append(fromAddrs, *address)

		}
	}

	toAddr := types.AddressConsensusGroup
	amount := string("0")

	ss := []string{
		"/cF/JQAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAABAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAEAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAnMxAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA",
		"/cF/JQAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAABAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAEAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAnMyAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA",
		"/cF/JQAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAABAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAEAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAnMzAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA",
		"/cF/JQAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAABAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAEAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAnM0AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA",
		"/cF/JQAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAABAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAEAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAnM1AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA",
	}
	var datas [][]byte
	for _, v := range ss {
		byts, err := base64.StdEncoding.DecodeString(v)
		if err != nil {
			panic(err)
		}
		datas = append(datas, byts)
	}

	num := atomic.NewUint32(0)

	fmt.Println("auto send...................................")
	tx.vite.Consensus().Subscribe(types.SNAPSHOT_GID, "api-auto-send", nil, func(e consensus.Event) {
		log.Info("api-auto-send trigger", "time", e.Timestamp)
		if num.Load() > 0 {
			fmt.Printf("something is loading[return].%s\n", time.Now())
			return
		}
		num.Add(1)
		defer num.Sub(1)
		snapshotBlock := tx.vite.Chain().GetLatestSnapshotBlock()
		if snapshotBlock.Height < 10 {
			fmt.Println("latest height must >= 10.")
			return
		}

		state := tx.vite.Net().Status().State
		if state != net.SyncDone {
			fmt.Printf("sync state: %s \n", state)
			return
		}

		for i := 0; i < N; i++ {
			for k, v := range fromAddrs {
				addr := v
				key := fromHexPrivKeys[k]

				block, err := tx.SendTxWithPrivateKey(SendTxWithPrivateKeyParam{
					SelfAddr:     &addr,
					ToAddr:       &toAddr,
					TokenTypeId:  ledger.ViteTokenId,
					PrivateKey:   &key,
					Amount:       &amount,
					Data:         datas[rand.Intn(len(datas))],
					Difficulty:   nil,
					PreBlockHash: nil,
					BlockType:    2,
				})
				if err != nil {
					log.Error(fmt.Sprintf("send block err:%v\n", err))
					return
				} else {
					log.Info(fmt.Sprintf("send block:%s,%s,%s\n", block.AccountAddress, block.Height, block.Hash))
				}
			}

		}

		for i := 0; i < M; i++ {
			for k, v := range fromAddrs {
				addr := v
				key := fromHexPrivKeys[k]

				mToAddr := types.HexToAddressPanic(InitContractAddr[rand.Intn(len(InitContractAddr))])
				block, err := tx.SendTxWithPrivateKey(SendTxWithPrivateKeyParam{
					SelfAddr:     &addr,
					ToAddr:       &mToAddr,
					TokenTypeId:  ledger.ViteTokenId,
					PrivateKey:   &key,
					Amount:       &amount,
					Difficulty:   nil,
					PreBlockHash: nil,
					BlockType:    2,
				})
				if err != nil {
					log.Error(fmt.Sprintf("send block err:%v\n", err))
					return
				} else {
					log.Info(fmt.Sprintf("send block:%s,%s,%s\n", block.AccountAddress, block.Height, block.Hash))
				}
			}
		}
	})
}
