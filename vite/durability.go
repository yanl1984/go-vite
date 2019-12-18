package vite

import (
	"bytes"
	"encoding/gob"
	"errors"
	"fmt"
	"github.com/syndtr/goleveldb/leveldb/util"
	"github.com/vitelabs/go-vite/chain"
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/crypto/ed25519"
	"github.com/vitelabs/go-vite/ledger"
	"github.com/vitelabs/go-vite/vm/contracts/abi"
	"math/big"
	"math/rand"
	"sync/atomic"
	"time"
)

const (
	USERS_COUNT   = 100
	SEND_SLEEP    = 3 * time.Millisecond
	RECEIVE_SLEEP = 3 * time.Millisecond
)

type User struct {
	Address    types.Address
	PublicKey  ed25519.PublicKey
	PrivateKey ed25519.PrivateKey

	Used uint64
}

func (user *User) init() {
	user.Used = 0
}

func (user *User) use() bool {
	return atomic.CompareAndSwapUint64(&user.Used, 0, 1)
}
func (user *User) unUse() {
	atomic.SwapUint64(&user.Used, 0)
}

func parseUserConfig(config []string) *User {
	address, err := types.HexToAddress(config[0])
	if err != nil {
		panic(err)
	}
	publicKey, err := ed25519.HexToPublicKey(config[1])
	if err != nil {
		panic(err)
	}
	privateKey, err := ed25519.HexToPrivateKey(config[2])
	if err != nil {
		panic(err)
	}

	return &User{
		Address:    address,
		PublicKey:  publicKey,
		PrivateKey: privateKey,
	}
}

var richUser = parseUserConfig([]string{
	"vite_f67a1262871a637505218c7097b9c30dfe69ea4f8bcf1b824b",
	"e40a4b1a608805f2da70ec37f64391bc8e373787c5e4fc97c4cad357cbf06737",
	"4abceb5d4d3382ebd9ce32003ada42be28c1d4de479a035d783a0658fb0ccc3ce40a4b1a608805f2da70ec37f64391bc8e373787c5e4fc97c4cad357cbf06737",
})

func DurabilityTest(vite *Vite) {
	loopCheckSb(vite)

	var users = initUsers(vite, USERS_COUNT)

	fmt.Printf("has %d users\n", len(users))

	getRandomUser := func(excludeAddr string) *User {

	LOOP:
		randomIndex := rand.Intn(len(users))
		user := users[randomIndex]
		if len(excludeAddr) > 0 && user.Address.String() == excludeAddr {
			goto LOOP
		}
		if !user.use() {
			goto LOOP
		}
		return user
	}

	go func() {
		for {
			sendUser := getRandomUser("")

			receiveUser := getRandomUser(sendUser.Address.String())
			receiveAddress := receiveUser.Address

			receiveUser.unUse()

			if _, err := send(vite, sendUser, receiveAddress, rand.Intn(30)); err != nil {
				fmt.Println(err)
			}
			sendUser.unUse()

			time.Sleep(SEND_SLEEP)

		}
	}()

	go func() {
		for {
			receiveUser := getRandomUser("")
			blocks, err := vite.chain.GetOnRoadBlocksByAddr(receiveUser.Address, 0, 1)
			if err != nil {
				fmt.Printf("GetOnRoadBlocksByAddr failed: %s\n", err)

				receiveUser.unUse()

				continue
			}

			if len(blocks) > 0 {
				if _, err := receive(vite, receiveUser, blocks[0].Hash); err != nil {
					fmt.Println(err)
				}
			}

			receiveUser.unUse()

			time.Sleep(RECEIVE_SLEEP)
		}
	}()

}

func loopCheckSb(vite *Vite) {
	for {
		sb := vite.Chain().GetLatestSnapshotBlock()
		if sb.Height > 10 {
			return
		}
		time.Sleep(1 * time.Second)

	}
}

func send(vite *Vite, from *User, toAddress types.Address, amount int) (*ledger.AccountBlock, error) {
	chain := vite.Chain()

	height, prevHash := getBlockBase(chain, from)

	ab := &ledger.AccountBlock{
		BlockType:      ledger.BlockTypeSendCall,
		PrevHash:       prevHash,
		Height:         height,
		AccountAddress: from.Address,
		PublicKey:      from.PublicKey,
		ToAddress:      toAddress,
		Amount:         big.NewInt(int64(amount)),
		TokenId:        ledger.ViteTokenId,
	}

	auth(ab, from)

	if err := sendRawTransaction(vite, ab); err != nil {

		return nil, errors.New(fmt.Sprintf("Send failed %s: %s\n", from.Address, err))
	}
	return ab, nil
}

func receive(vite *Vite, user *User, fromBlockHash types.Hash) (*ledger.AccountBlock, error) {
	chain := vite.Chain()
	height, prevHash := getBlockBase(chain, user)

	ab := &ledger.AccountBlock{
		BlockType: ledger.BlockTypeReceive,
		PrevHash:  prevHash,
		Height:    height,

		AccountAddress: user.Address,
		PublicKey:      user.PublicKey,
		FromBlockHash:  fromBlockHash,
	}

	auth(ab, user)

	if err := sendRawTransaction(vite, ab); err != nil {
		return nil, errors.New(fmt.Sprintf("Receive failed %s: %s\n", user.Address, err))
	}
	return ab, nil
}

func pledge(vite *Vite, user *User, toUser *User) error {
	chain := vite.Chain()
	height, prevHash := getBlockBase(chain, user)

	data, err := abi.ABIQuota.PackMethod(abi.MethodNameStakeV3, toUser.Address)
	if err != nil {
		return err
	}

	// 50w
	amount, _ := big.NewInt(0).SetString("5000000000000000000000000", 10)

	ab := &ledger.AccountBlock{
		BlockType: ledger.BlockTypeSendCall,
		PrevHash:  prevHash,
		Height:    height,

		AccountAddress: user.Address,
		ToAddress:      types.AddressQuota,

		Amount:  amount,
		TokenId: ledger.ViteTokenId,

		Data:      data,
		PublicKey: user.PublicKey,
	}

	auth(ab, user)

	if err := sendRawTransaction(vite, ab); err != nil {

		return errors.New(fmt.Sprintf("pledge failed %s: %s\n", user.Address, err))
	}

	for {
		rb, err := chain.GetReceiveAbBySendAb(ab.Hash)
		if err != nil {
			return err
		}
		if rb != nil {
			confirmedTimes, err := chain.GetConfirmedTimes(rb.Hash)
			if err != nil {
				return err
			}
			if confirmedTimes > 1 {
				break
			}
		}

		time.Sleep(150 * time.Millisecond)
	}
	return nil
}

func sendRawTransaction(vite *Vite, block *ledger.AccountBlock) error {
	latestSb := vite.Chain().GetLatestSnapshotBlock()

	result, err := vite.Verifier().VerifyRPCAccountBlock(block, latestSb)
	if err != nil {
		return err
	}

	if result != nil {
		return vite.Pool().AddDirectAccountBlock(result.AccountBlock.AccountAddress, result)
	}
	return errors.New("generator gen an empty block")
}

func getBlockBase(chain chain.Chain, user *User) (uint64, types.Hash) {
	latestAb, err := chain.GetLatestAccountBlock(user.Address)
	if err != nil {
		panic(err)
	}

	height := uint64(1)
	var prevHash types.Hash
	if latestAb != nil {
		height = latestAb.Height + 1
		prevHash = latestAb.Hash
	}
	return height, prevHash
}

func auth(block *ledger.AccountBlock, user *User) {
	block.Hash = block.ComputeHash()
	block.Signature = ed25519.Sign(user.PrivateKey, block.Hash.Bytes())
}

const (
	USER_PREFIX = byte(0)
)

func initUsers(vite *Vite, minSize int) []*User {
	c := vite.Chain()
	testDB, err := c.NewDb("durability_test")
	if err != nil {
		panic(err)
	}
	iter := testDB.NewIterator(util.BytesPrefix([]byte{USER_PREFIX}), nil)
	defer iter.Release()

	var users []*User

	for iter.Next() {
		var data = bytes.NewBuffer(iter.Value())

		dec := gob.NewDecoder(data)

		var user = User{}
		if err := dec.Decode(&user); err != nil {
			panic(err)
		}
		user.init()
		users = append(users, &user)
	}

	if len(users) < minSize {
		var gap = minSize - len(users)
		for i := 0; i < gap; i++ {
			addr, privateKey, err := types.CreateAddress()
			if err != nil {
				panic(err)
			}
			publicKey := ed25519.PublicKey(privateKey.PubByte())

			var newUser = &User{
				Address:    addr,
				PublicKey:  publicKey,
				PrivateKey: privateKey,
			}

			// pledge
			fmt.Printf("Pledge %s\n", newUser.Address)
			if err := pledge(vite, richUser, newUser); err != nil {
				panic(err)
			}

			sendBlock, err := send(vite, richUser, newUser.Address, 100000000000)
			if err != nil {
				panic(err)
			}

			// sendMoney
			if _, err := receive(vite, newUser, sendBlock.Hash); err != nil {
				panic(err)
			}

			// save
			var data = bytes.NewBuffer(nil)
			enc := gob.NewEncoder(data)
			if err := enc.Encode(newUser); err != nil {
				panic(err)
			}

			if err := testDB.Put(append([]byte{USER_PREFIX}, addr.Bytes()...), data.Bytes(), nil); err != nil {
				panic(err)
			}

			users = append(users, newUser)
			fmt.Printf("Create user[%d]: %s\n", len(users), newUser.Address)

		}
	}

	return users
}
