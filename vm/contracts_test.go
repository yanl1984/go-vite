package vm

import (
	"bytes"
	"encoding/hex"
	"fmt"
	"github.com/vitelabs/go-vite/common/helper"
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/consensus/core"
	"github.com/vitelabs/go-vite/crypto/ed25519"
	"github.com/vitelabs/go-vite/ledger"
	"github.com/vitelabs/go-vite/vm/contracts/abi"
	"github.com/vitelabs/go-vite/vm/util"
	"math/big"
	"regexp"
	"strconv"
	"testing"
	"time"
)

func TestContractsRefund(t *testing.T) {
	// prepare db
	viteTotalSupply := new(big.Int).Mul(big.NewInt(1e9), big.NewInt(1e18))
	db, addr1, _, hash12, snapshot2, _ := prepareDb(viteTotalSupply)

	addr2 := types.AddressGovernance
	sbpName := "s1"
	locHashRegister, _ := types.BytesToHash(abi.GetRegistrationInfoKey(sbpName, types.SNAPSHOT_GID))
	registrationDataOld := db.storageMap[addr2][ToKey(locHashRegister.Bytes())]
	db.addr = addr2
	contractBalance, _ := db.GetBalance(&ledger.ViteTokenId)
	// register with an existed super node name, get refund
	balance1 := new(big.Int).Set(viteTotalSupply)
	addr6, _, _ := types.CreateAddress()
	db.accountBlockMap[addr6] = make(map[types.Hash]*ledger.AccountBlock)
	block13Data, err := abi.ABIGovernance.PackMethod(abi.MethodNameRegister, types.SNAPSHOT_GID, sbpName, addr6)
	if err != nil {
		panic(err)
	}
	hash13 := types.DataHash([]byte{1, 3})
	block13 := &ledger.AccountBlock{
		Height:         3,
		ToAddress:      addr2,
		AccountAddress: addr1,
		BlockType:      ledger.BlockTypeSendCall,
		PrevHash:       hash12,
		Amount:         new(big.Int).Mul(big.NewInt(1e6), big.NewInt(1e18)),
		Fee:            big.NewInt(0),
		Data:           block13Data,
		TokenId:        ledger.ViteTokenId,
		Hash:           hash13,
	}
	vm := NewVM(nil)
	//vm.Debug = true
	db.addr = addr1
	sendRegisterBlock, isRetry, err := vm.RunV2(db, block13, nil, nil)
	balance1.Sub(balance1, block13.Amount)
	if sendRegisterBlock == nil ||
		len(sendRegisterBlock.AccountBlock.SendBlockList) != 0 || isRetry || err != nil ||
		sendRegisterBlock.AccountBlock.Quota != vm.gasTable.RegisterQuota ||
		sendRegisterBlock.AccountBlock.Quota != sendRegisterBlock.AccountBlock.QuotaUsed ||
		!bytes.Equal(sendRegisterBlock.AccountBlock.Data, block13Data) ||
		db.balanceMap[addr1][ledger.ViteTokenId].Cmp(balance1) != 0 {
		t.Fatalf("send register transaction error")
	}
	db.accountBlockMap[addr1][hash13] = sendRegisterBlock.AccountBlock

	hash21 := types.DataHash([]byte{2, 1})
	block21 := &ledger.AccountBlock{
		Height:         1,
		AccountAddress: addr2,
		BlockType:      ledger.BlockTypeReceive,
		FromBlockHash:  hash13,
		Hash:           hash21,
	}
	vm = NewVM(nil)
	//vm.Debug = true
	db.addr = addr2
	receiveRegisterBlock, isRetry, err := vm.RunV2(db, block21, sendRegisterBlock.AccountBlock, NewTestGlobalStatus(0, snapshot2))
	contractBalance.Add(contractBalance, block13.Amount)
	newBalance, _ := db.GetBalance(&ledger.ViteTokenId)
	if receiveRegisterBlock == nil ||
		len(receiveRegisterBlock.AccountBlock.SendBlockList) != 1 || isRetry || err == nil ||
		db.balanceMap[addr1][ledger.ViteTokenId].Cmp(balance1) != 0 ||
		!bytes.Equal(db.storageMap[addr2][ToKey(locHashRegister.Bytes())], registrationDataOld) ||
		receiveRegisterBlock.AccountBlock.Quota != 0 ||
		receiveRegisterBlock.AccountBlock.Quota != receiveRegisterBlock.AccountBlock.QuotaUsed ||
		len(receiveRegisterBlock.AccountBlock.Data) != 33 ||
		receiveRegisterBlock.AccountBlock.Data[32] != byte(1) ||
		receiveRegisterBlock.AccountBlock.SendBlockList[0].TokenId != block13.TokenId ||
		receiveRegisterBlock.AccountBlock.SendBlockList[0].Amount.Cmp(block13.Amount) != 0 ||
		receiveRegisterBlock.AccountBlock.SendBlockList[0].BlockType != ledger.BlockTypeSendCall ||
		receiveRegisterBlock.AccountBlock.SendBlockList[0].AccountAddress != block13.ToAddress ||
		receiveRegisterBlock.AccountBlock.SendBlockList[0].ToAddress != block13.AccountAddress ||
		newBalance.Cmp(contractBalance) != 0 ||
		!bytes.Equal(receiveRegisterBlock.AccountBlock.SendBlockList[0].Data, []byte{}) {
		t.Fatalf("receive register transaction error")
	}
	db.accountBlockMap[addr2] = make(map[types.Hash]*ledger.AccountBlock)
	db.accountBlockMap[addr2][hash21] = receiveRegisterBlock.AccountBlock
	hash22 := types.DataHash([]byte{2, 2})
	receiveRegisterBlock.AccountBlock.SendBlockList[0].Hash = hash22
	receiveRegisterBlock.AccountBlock.SendBlockList[0].PrevHash = hash21
	db.accountBlockMap[addr2][hash22] = receiveRegisterBlock.AccountBlock.SendBlockList[0]

	hash14 := types.DataHash([]byte{1, 4})
	block14 := &ledger.AccountBlock{
		Height:         4,
		AccountAddress: addr1,
		BlockType:      ledger.BlockTypeReceive,
		FromBlockHash:  hash22,
		Hash:           hash14,
	}
	vm = NewVM(nil)
	//vm.Debug = true
	db.addr = addr1
	receiveRegisterRefuncBlock, isRetry, err := vm.RunV2(db, block14, receiveRegisterBlock.AccountBlock.SendBlockList[0], nil)
	balance1.Add(balance1, block13.Amount)
	if receiveRegisterRefuncBlock == nil ||
		len(receiveRegisterRefuncBlock.AccountBlock.SendBlockList) != 0 || isRetry || err != nil ||
		db.balanceMap[addr1][ledger.ViteTokenId].Cmp(balance1) != 0 ||
		receiveRegisterRefuncBlock.AccountBlock.Quota != 21000 ||
		receiveRegisterRefuncBlock.AccountBlock.Quota != receiveRegisterRefuncBlock.AccountBlock.QuotaUsed {
		t.Fatalf("receive register refund transaction error")
	}
	db.accountBlockMap[addr1][hash14] = receiveRegisterRefuncBlock.AccountBlock
}

func TestContractsRegister(t *testing.T) {
	// prepare db
	viteTotalSupply := new(big.Int).Mul(big.NewInt(1e9), big.NewInt(1e18))
	db, addr1, _, hash12, snapshot2, timestamp := prepareDb(viteTotalSupply)

	reader := util.NewVMConsensusReader(newConsensusReaderTest(db.GetGenesisSnapshotBlock().Timestamp.Unix(), 24*3600, nil))
	// register
	balance1 := new(big.Int).Set(viteTotalSupply)
	addr6, privateKey6, _ := types.CreateAddress()
	addr7, _, _ := types.CreateAddress()
	publicKey6 := ed25519.PublicKey(privateKey6.PubByte())
	db.accountBlockMap[addr6] = make(map[types.Hash]*ledger.AccountBlock)
	db.accountBlockMap[addr7] = make(map[types.Hash]*ledger.AccountBlock)
	addr2 := types.AddressGovernance
	sbpName := "super1"
	block13Data, err := abi.ABIGovernance.PackMethod(abi.MethodNameRegister, types.SNAPSHOT_GID, sbpName, addr7)
	hash13 := types.DataHash([]byte{1, 3})
	block13 := &ledger.AccountBlock{
		Height:         3,
		ToAddress:      addr2,
		AccountAddress: addr1,
		BlockType:      ledger.BlockTypeSendCall,
		PrevHash:       hash12,
		Amount:         new(big.Int).Mul(big.NewInt(5e5), big.NewInt(1e18)),
		Fee:            big.NewInt(0),
		Data:           block13Data,
		TokenId:        ledger.ViteTokenId,
		Hash:           hash13,
	}
	vm := NewVM(reader)
	//vm.Debug = true
	db.addr = addr1
	sendRegisterBlock, isRetry, err := vm.RunV2(db, block13, nil, nil)
	balance1.Sub(balance1, block13.Amount)
	if sendRegisterBlock == nil ||
		len(sendRegisterBlock.AccountBlock.SendBlockList) != 0 || isRetry || err != nil ||
		sendRegisterBlock.AccountBlock.Quota != vm.gasTable.RegisterQuota ||
		sendRegisterBlock.AccountBlock.Quota != sendRegisterBlock.AccountBlock.QuotaUsed ||
		!bytes.Equal(sendRegisterBlock.AccountBlock.Data, block13Data) ||
		db.balanceMap[addr1][ledger.ViteTokenId].Cmp(balance1) != 0 {
		t.Fatalf("send register transaction error")
	}
	db.accountBlockMap[addr1][hash13] = sendRegisterBlock.AccountBlock

	hash21 := types.DataHash([]byte{2, 1})
	block21 := &ledger.AccountBlock{
		Height:         1,
		AccountAddress: addr2,
		BlockType:      ledger.BlockTypeReceive,
		FromBlockHash:  hash13,
		Hash:           hash21,
	}
	vm = NewVM(reader)
	//vm.Debug = true
	locHashRegister := abi.GetRegistrationInfoKey(sbpName, types.SNAPSHOT_GID)
	hisAddrList := []types.Address{addr7}
	expirationHeight := snapshot2.Height + 3600*24*90
	registrationData, _ := abi.ABIGovernance.PackVariable(abi.VariableNameRegistrationInfo, sbpName, addr7, addr1, block13.Amount, expirationHeight, snapshot2.Timestamp.Unix(), int64(0), hisAddrList)
	db.addr = addr2
	receiveRegisterBlock, isRetry, err := vm.RunV2(db, block21, sendRegisterBlock.AccountBlock, NewTestGlobalStatus(0, snapshot2))
	if receiveRegisterBlock == nil ||
		len(receiveRegisterBlock.AccountBlock.SendBlockList) != 0 || isRetry || err != nil ||
		db.balanceMap[addr1][ledger.ViteTokenId].Cmp(balance1) != 0 ||
		!bytes.Equal(db.storageMap[addr2][ToKey(locHashRegister)], registrationData) ||
		len(receiveRegisterBlock.AccountBlock.Data) != 33 ||
		receiveRegisterBlock.AccountBlock.Data[32] != byte(0) ||
		receiveRegisterBlock.AccountBlock.Quota != 0 ||
		receiveRegisterBlock.AccountBlock.Quota != receiveRegisterBlock.AccountBlock.QuotaUsed {
		t.Fatalf("receive register transaction error")
	}
	db.accountBlockMap[addr2] = make(map[types.Hash]*ledger.AccountBlock)
	db.accountBlockMap[addr2][hash21] = receiveRegisterBlock.AccountBlock

	// update registration
	block14Data, err := abi.ABIGovernance.PackMethod(abi.MethodNameUpdateBlockProducingAddress, types.SNAPSHOT_GID, sbpName, addr6)
	hash14 := types.DataHash([]byte{1, 4})
	block14 := &ledger.AccountBlock{
		Height:         4,
		ToAddress:      addr2,
		AccountAddress: addr1,
		BlockType:      ledger.BlockTypeSendCall,
		PrevHash:       hash13,
		Data:           block14Data,
		Amount:         big.NewInt(0),
		Fee:            big.NewInt(0),
		TokenId:        ledger.ViteTokenId,
		Hash:           hash14,
	}
	vm = NewVM(reader)
	//vm.Debug = true
	db.addr = addr1
	sendRegisterBlock2, isRetry, err := vm.RunV2(db, block14, nil, nil)
	if sendRegisterBlock2 == nil ||
		len(sendRegisterBlock2.AccountBlock.SendBlockList) != 0 || isRetry || err != nil ||
		sendRegisterBlock2.AccountBlock.Quota != vm.gasTable.UpdateBlockProducingAddressQuota ||
		sendRegisterBlock2.AccountBlock.Quota != sendRegisterBlock2.AccountBlock.QuotaUsed ||
		!bytes.Equal(sendRegisterBlock2.AccountBlock.Data, block14Data) ||
		db.balanceMap[addr1][ledger.ViteTokenId].Cmp(balance1) != 0 {
		t.Fatalf("send update registration transaction error")
	}
	db.accountBlockMap[addr1][hash14] = sendRegisterBlock2.AccountBlock

	hash22 := types.DataHash([]byte{2, 2})
	block22 := &ledger.AccountBlock{
		Height:         2,
		AccountAddress: addr2,
		BlockType:      ledger.BlockTypeReceive,
		FromBlockHash:  hash14,
		PrevHash:       hash21,
		Hash:           hash22,
	}
	vm = NewVM(reader)
	//vm.Debug = true
	hisAddrList = append(hisAddrList, addr6)
	registrationData, _ = abi.ABIGovernance.PackVariable(abi.VariableNameRegistrationInfo, sbpName, addr6, addr1, block13.Amount, expirationHeight, snapshot2.Timestamp.Unix(), int64(0), hisAddrList)
	db.addr = addr2
	receiveRegisterBlock2, isRetry, err := vm.RunV2(db, block22, sendRegisterBlock2.AccountBlock, NewTestGlobalStatus(0, snapshot2))
	if receiveRegisterBlock2 == nil ||
		len(receiveRegisterBlock2.AccountBlock.SendBlockList) != 0 || isRetry || err != nil ||
		db.balanceMap[addr1][ledger.ViteTokenId].Cmp(balance1) != 0 ||
		!bytes.Equal(db.storageMap[addr2][ToKey(locHashRegister)], registrationData) ||
		len(receiveRegisterBlock2.AccountBlock.Data) != 33 ||
		receiveRegisterBlock2.AccountBlock.Data[32] != byte(0) ||
		receiveRegisterBlock2.AccountBlock.Quota != 0 ||
		receiveRegisterBlock2.AccountBlock.Quota != receiveRegisterBlock2.AccountBlock.QuotaUsed {
		t.Fatalf("receive update registration transaction error")
	}
	db.accountBlockMap[addr2][hash22] = receiveRegisterBlock2.AccountBlock

	// get contracts data
	db.addr = types.AddressGovernance
	if registerList, _ := abi.GetCandidateList(db, types.SNAPSHOT_GID); len(registerList) != 3 || len(registerList[0].Name) == 0 {
		t.Fatalf("get register list failed")
	}

	// cancel register
	time3 := time.Unix(timestamp+1, 0)
	snapshot3 := &ledger.SnapshotBlock{Height: 3, Timestamp: &time3, Hash: types.DataHash([]byte{10, 3}), PublicKey: publicKey6}
	db.snapshotBlockList = append(db.snapshotBlockList, snapshot3)
	time4 := time.Unix(timestamp+2, 0)
	snapshot4 := &ledger.SnapshotBlock{Height: 4, Timestamp: &time4, Hash: types.DataHash([]byte{10, 4}), PublicKey: publicKey6}
	db.snapshotBlockList = append(db.snapshotBlockList, snapshot4)
	time5 := time.Unix(timestamp+1+3600*24*90, 0)
	snapshot5 := &ledger.SnapshotBlock{Height: 3 + 3600*24*90, Timestamp: &time5, Hash: types.DataHash([]byte{10, 5})}
	db.snapshotBlockList = append(db.snapshotBlockList, snapshot5)

	hash15 := types.DataHash([]byte{1, 5})
	block15Data, _ := abi.ABIGovernance.PackMethod(abi.MethodNameRevoke, types.SNAPSHOT_GID, sbpName)
	block15 := &ledger.AccountBlock{
		Height:         5,
		ToAddress:      addr2,
		AccountAddress: addr1,
		Amount:         big.NewInt(0),
		TokenId:        ledger.ViteTokenId,
		BlockType:      ledger.BlockTypeSendCall,
		Fee:            big.NewInt(0),
		PrevHash:       hash13,
		Data:           block15Data,
		Hash:           hash15,
	}
	vm = NewVM(reader)
	//vm.Debug = true
	db.addr = addr1
	sendCancelRegisterBlock, isRetry, err := vm.RunV2(db, block15, nil, nil)
	if sendCancelRegisterBlock == nil ||
		len(sendCancelRegisterBlock.AccountBlock.SendBlockList) != 0 || isRetry || err != nil ||
		sendCancelRegisterBlock.AccountBlock.Quota != vm.gasTable.RevokeQuota ||
		sendCancelRegisterBlock.AccountBlock.Quota != sendCancelRegisterBlock.AccountBlock.QuotaUsed ||
		!bytes.Equal(sendCancelRegisterBlock.AccountBlock.Data, block15Data) ||
		db.balanceMap[addr1][ledger.ViteTokenId].Cmp(balance1) != 0 {
		t.Fatalf("send cancel register transaction error")
	}
	db.accountBlockMap[addr1][hash15] = sendCancelRegisterBlock.AccountBlock

	hash23 := types.DataHash([]byte{2, 3})
	block23 := &ledger.AccountBlock{
		Height:         3,
		AccountAddress: addr2,
		BlockType:      ledger.BlockTypeReceive,
		PrevHash:       hash21,
		FromBlockHash:  hash15,
	}
	vm = NewVM(reader)
	//vm.Debug = true
	db.addr = addr2
	receiveCancelRegisterBlock, isRetry, err := vm.RunV2(db, block23, sendCancelRegisterBlock.AccountBlock, NewTestGlobalStatus(0, snapshot5))
	registrationData, _ = abi.ABIGovernance.PackVariable(abi.VariableNameRegistrationInfoV2, sbpName, addr6, addr1, addr1, helper.Big0, uint64(0), snapshot2.Timestamp.Unix(), snapshot5.Timestamp.Unix(), hisAddrList)
	if receiveCancelRegisterBlock == nil ||
		len(receiveCancelRegisterBlock.AccountBlock.SendBlockList) != 1 || isRetry || err != nil ||
		db.balanceMap[addr2][ledger.ViteTokenId].Cmp(helper.Big0) != 0 ||
		db.balanceMap[addr1][ledger.ViteTokenId].Cmp(balance1) != 0 ||
		!bytes.Equal(db.storageMap[addr2][ToKey(locHashRegister)], registrationData) ||
		len(receiveCancelRegisterBlock.AccountBlock.Data) != 33 ||
		receiveCancelRegisterBlock.AccountBlock.Data[32] != byte(0) ||
		receiveCancelRegisterBlock.AccountBlock.Quota != 0 ||
		receiveCancelRegisterBlock.AccountBlock.Quota != receiveCancelRegisterBlock.AccountBlock.QuotaUsed ||
		receiveCancelRegisterBlock.AccountBlock.SendBlockList[0].AccountAddress != addr2 ||
		receiveCancelRegisterBlock.AccountBlock.SendBlockList[0].ToAddress != addr1 ||
		receiveCancelRegisterBlock.AccountBlock.SendBlockList[0].BlockType != ledger.BlockTypeSendCall {
		t.Fatalf("receive cancel register transaction error")
	}
	db.accountBlockMap[addr2][hash23] = receiveCancelRegisterBlock.AccountBlock
	hash24 := types.DataHash([]byte{2, 4})
	receiveCancelRegisterBlock.AccountBlock.SendBlockList[0].Hash = hash24
	receiveCancelRegisterBlock.AccountBlock.SendBlockList[0].PrevHash = hash23
	db.accountBlockMap[addr2][hash24] = receiveCancelRegisterBlock.AccountBlock.SendBlockList[0]

	hash16 := types.DataHash([]byte{1, 6})
	block16 := &ledger.AccountBlock{
		Height:         6,
		AccountAddress: addr1,
		BlockType:      ledger.BlockTypeReceive,
		PrevHash:       hash16,
		FromBlockHash:  hash23,
		Hash:           hash16,
	}
	vm = NewVM(reader)
	//vm.Debug = true
	db.addr = addr1
	balance1.Add(balance1, block13.Amount)
	receiveCancelRegisterRefundBlock, isRetry, err := vm.RunV2(db, block16, receiveCancelRegisterBlock.AccountBlock.SendBlockList[0], nil)
	if receiveCancelRegisterRefundBlock == nil ||
		len(receiveCancelRegisterRefundBlock.AccountBlock.SendBlockList) != 0 || isRetry || err != nil ||
		db.balanceMap[addr2][ledger.ViteTokenId].Cmp(helper.Big0) != 0 ||
		db.balanceMap[addr1][ledger.ViteTokenId].Cmp(balance1) != 0 ||
		receiveCancelRegisterRefundBlock.AccountBlock.Quota != 21000 ||
		receiveCancelRegisterRefundBlock.AccountBlock.Quota != receiveCancelRegisterRefundBlock.AccountBlock.QuotaUsed {
		t.Fatalf("receive cancel register refund transaction error")
	}
	db.accountBlockMap[addr1][hash16] = receiveCancelRegisterRefundBlock.AccountBlock

	// TODO reward
	// Reward
	hash17 := types.DataHash([]byte{1, 7})
	block17Data, _ := abi.ABIGovernance.PackMethod(abi.MethodNameWithdrawReward, types.SNAPSHOT_GID, sbpName, addr1)
	block17 := &ledger.AccountBlock{
		Height:         7,
		ToAddress:      addr2,
		AccountAddress: addr1,
		Amount:         big.NewInt(0),
		TokenId:        ledger.ViteTokenId,
		BlockType:      ledger.BlockTypeSendCall,
		Fee:            big.NewInt(0),
		PrevHash:       hash16,
		Data:           block17Data,
		Hash:           hash17,
	}
	vm = NewVM(nil)
	//vm.Debug = true
	db.addr = addr1
	sendRewardBlock, isRetry, err := vm.RunV2(db, block17, nil, nil)
	if sendRewardBlock == nil ||
		len(sendRewardBlock.AccountBlock.SendBlockList) != 0 || isRetry || err != nil ||
		sendRewardBlock.AccountBlock.Quota != vm.gasTable.WithdrawRewardQuota ||
		sendRewardBlock.AccountBlock.Quota != sendRewardBlock.AccountBlock.QuotaUsed ||
		!bytes.Equal(sendRewardBlock.AccountBlock.Data, block17Data) ||
		db.balanceMap[addr1][ledger.ViteTokenId].Cmp(balance1) != 0 {
		t.Fatalf("send cancel register transaction error")
	}
	db.accountBlockMap[addr1][hash17] = sendRewardBlock.AccountBlock

	hash25 := types.DataHash([]byte{2, 5})
	block25 := &ledger.AccountBlock{
		Height:         5,
		AccountAddress: addr2,
		BlockType:      ledger.BlockTypeReceive,
		PrevHash:       hash24,
		FromBlockHash:  hash17,
	}
	vm = NewVM(reader)
	//vm.Debug = true
	db.addr = addr2
	receiveRewardBlock, isRetry, err := vm.RunV2(db, block25, sendRewardBlock.AccountBlock, NewTestGlobalStatus(0, snapshot5))
	registrationData, _ = abi.ABIGovernance.PackVariable(abi.VariableNameRegistrationInfoV2, sbpName, addr6, addr1, addr1, helper.Big0, uint64(0), int64(snapshot5.Timestamp.Unix()-2-24*3600), snapshot5.Timestamp.Unix(), hisAddrList)
	if receiveRewardBlock == nil ||
		len(receiveRewardBlock.AccountBlock.SendBlockList) != 0 || isRetry || err != nil ||
		db.balanceMap[addr2][ledger.ViteTokenId].Cmp(helper.Big0) != 0 ||
		db.balanceMap[addr1][ledger.ViteTokenId].Cmp(balance1) != 0 ||
		!bytes.Equal(db.storageMap[addr2][ToKey(locHashRegister)], registrationData) ||
		len(receiveRewardBlock.AccountBlock.Data) != 33 ||
		receiveRewardBlock.AccountBlock.Data[32] != byte(0) ||
		receiveRewardBlock.AccountBlock.Quota != 0 ||
		receiveRewardBlock.AccountBlock.Quota != receiveRewardBlock.AccountBlock.QuotaUsed {
		t.Fatalf("receive reward transaction error")
	}
	db.accountBlockMap[addr2][hash25] = receiveRewardBlock.AccountBlock
}

func TestContractsVote(t *testing.T) {
	// prepare db
	viteTotalSupply := new(big.Int).Mul(big.NewInt(2e6), big.NewInt(1e18))
	db, addr1, _, hash12, snapshot2, _ := prepareDb(viteTotalSupply)
	// vote
	addr3 := types.AddressGovernance
	sbpName := "s1"
	block13Data, _ := abi.ABIGovernance.PackMethod(abi.MethodNameVote, types.SNAPSHOT_GID, sbpName)
	hash13 := types.DataHash([]byte{1, 3})
	block13 := &ledger.AccountBlock{
		Height:         3,
		ToAddress:      addr3,
		AccountAddress: addr1,
		PrevHash:       hash12,
		Amount:         big.NewInt(0),
		TokenId:        ledger.ViteTokenId,
		BlockType:      ledger.BlockTypeSendCall,
		Fee:            big.NewInt(0),
		Data:           block13Data,
		Hash:           hash13,
	}
	vm := NewVM(nil)
	//vm.Debug = true
	db.addr = addr1
	sendVoteBlock, isRetry, err := vm.RunV2(db, block13, nil, nil)
	if sendVoteBlock == nil ||
		len(sendVoteBlock.AccountBlock.SendBlockList) != 0 || isRetry || err != nil ||
		!bytes.Equal(sendVoteBlock.AccountBlock.Data, block13Data) ||
		sendVoteBlock.AccountBlock.Quota != vm.gasTable.VoteQuota ||
		sendVoteBlock.AccountBlock.Quota != sendVoteBlock.AccountBlock.QuotaUsed {
		t.Fatalf("send vote transaction error")
	}
	db.accountBlockMap[addr1][hash13] = sendVoteBlock.AccountBlock

	hash31 := types.DataHash([]byte{3, 1})
	block31 := &ledger.AccountBlock{
		Height:         1,
		AccountAddress: addr3,
		BlockType:      ledger.BlockTypeReceive,
		FromBlockHash:  hash13,
		Hash:           hash31,
	}
	vm = NewVM(nil)
	//vm.Debug = true
	db.addr = addr3
	receiveVoteBlock, isRetry, err := vm.RunV2(db, block31, sendVoteBlock.AccountBlock, NewTestGlobalStatus(0, snapshot2))
	voteKey := abi.GetVoteInfoKey(addr1, types.SNAPSHOT_GID)
	voteData, _ := abi.ABIGovernance.PackVariable(abi.VariableNameVoteInfo, sbpName)
	if receiveVoteBlock == nil ||
		len(receiveVoteBlock.AccountBlock.SendBlockList) != 0 || isRetry || err != nil ||
		!bytes.Equal(db.storageMap[addr3][ToKey(voteKey)], voteData) ||
		len(receiveVoteBlock.AccountBlock.Data) != 33 ||
		receiveVoteBlock.AccountBlock.Data[32] != byte(0) ||
		receiveVoteBlock.AccountBlock.Quota != 0 ||
		receiveVoteBlock.AccountBlock.Quota != receiveVoteBlock.AccountBlock.QuotaUsed {
		t.Fatalf("receive vote transaction error")
	}
	db.accountBlockMap[addr3] = make(map[types.Hash]*ledger.AccountBlock)
	db.accountBlockMap[addr3][hash31] = receiveVoteBlock.AccountBlock

	addr4, _ := types.BytesToAddress(helper.HexToBytes("e5bf58cacfb74cf8c49a1d5e59d3919c9a4cb9ed"))
	db.accountBlockMap[addr4] = make(map[types.Hash]*ledger.AccountBlock)
	sbpName2 := "s2"
	block14Data, _ := abi.ABIGovernance.PackMethod(abi.MethodNameVote, types.SNAPSHOT_GID, sbpName2)
	hash14 := types.DataHash([]byte{1, 4})
	block14 := &ledger.AccountBlock{
		Height:         4,
		ToAddress:      addr3,
		AccountAddress: addr1,
		Amount:         big.NewInt(0),
		TokenId:        ledger.ViteTokenId,
		BlockType:      ledger.BlockTypeSendCall,
		Fee:            big.NewInt(0),
		PrevHash:       hash13,
		Data:           block14Data,
		Hash:           hash14,
	}
	vm = NewVM(nil)
	//vm.Debug = true
	db.addr = addr1
	sendVoteBlock2, isRetry, err := vm.RunV2(db, block14, nil, nil)
	if sendVoteBlock2 == nil ||
		len(sendVoteBlock2.AccountBlock.SendBlockList) != 0 || isRetry || err != nil ||
		!bytes.Equal(sendVoteBlock2.AccountBlock.Data, block14Data) ||
		sendVoteBlock2.AccountBlock.Quota != vm.gasTable.VoteQuota ||
		sendVoteBlock2.AccountBlock.Quota != sendVoteBlock2.AccountBlock.QuotaUsed {
		t.Fatalf("send vote transaction 2 error")
	}
	db.accountBlockMap[addr1][hash14] = sendVoteBlock2.AccountBlock

	hash32 := types.DataHash([]byte{3, 2})
	block32 := &ledger.AccountBlock{
		Height:         2,
		AccountAddress: addr3,
		BlockType:      ledger.BlockTypeReceive,
		PrevHash:       hash31,
		FromBlockHash:  hash14,
		Hash:           hash32,
	}
	vm = NewVM(nil)
	//vm.Debug = true
	db.addr = addr3
	receiveVoteBlock2, isRetry, err := vm.RunV2(db, block32, sendVoteBlock2.AccountBlock, NewTestGlobalStatus(0, snapshot2))
	voteData, _ = abi.ABIGovernance.PackVariable(abi.VariableNameVoteInfo, sbpName2)
	if receiveVoteBlock2 == nil ||
		len(receiveVoteBlock2.AccountBlock.SendBlockList) != 0 || isRetry || err != nil ||
		!bytes.Equal(db.storageMap[addr3][ToKey(voteKey)], voteData) ||
		len(receiveVoteBlock2.AccountBlock.Data) != 33 ||
		receiveVoteBlock2.AccountBlock.Data[32] != byte(0) ||
		receiveVoteBlock2.AccountBlock.Quota != 0 ||
		receiveVoteBlock2.AccountBlock.Quota != receiveVoteBlock2.AccountBlock.QuotaUsed {
		t.Fatalf("receive vote transaction 2 error")
	}
	db.accountBlockMap[addr3][hash32] = receiveVoteBlock2.AccountBlock

	// get contracts data
	db.addr = types.AddressGovernance
	if voteList, _ := abi.GetVoteList(db, types.SNAPSHOT_GID); len(voteList) != 1 || voteList[0].SbpName != sbpName2 {
		t.Fatalf("get vote list failed")
	}

	// cancel vote
	block15Data, _ := abi.ABIGovernance.PackMethod(abi.MethodNameCancelVote, types.SNAPSHOT_GID)
	hash15 := types.DataHash([]byte{1, 5})
	block15 := &ledger.AccountBlock{
		Height:         5,
		ToAddress:      addr3,
		AccountAddress: addr1,
		Amount:         big.NewInt(0),
		TokenId:        ledger.ViteTokenId,
		BlockType:      ledger.BlockTypeSendCall,
		Fee:            big.NewInt(0),
		PrevHash:       hash14,
		Data:           block15Data,
		Hash:           hash15,
	}
	vm = NewVM(nil)
	//vm.Debug = true
	db.addr = addr1
	sendCancelVoteBlock, isRetry, err := vm.RunV2(db, block15, nil, nil)
	if sendCancelVoteBlock == nil ||
		len(sendCancelVoteBlock.AccountBlock.SendBlockList) != 0 || isRetry || err != nil ||
		!bytes.Equal(sendCancelVoteBlock.AccountBlock.Data, block15Data) ||
		sendCancelVoteBlock.AccountBlock.Quota != vm.gasTable.CancelVoteQuota ||
		sendCancelVoteBlock.AccountBlock.Quota != sendCancelVoteBlock.AccountBlock.QuotaUsed {
		t.Fatalf("send cancel vote transaction error")
	}
	db.accountBlockMap[addr1][hash15] = sendCancelVoteBlock.AccountBlock

	hash33 := types.DataHash([]byte{3, 3})
	block33 := &ledger.AccountBlock{
		Height:         3,
		AccountAddress: addr3,
		BlockType:      ledger.BlockTypeReceive,
		PrevHash:       hash32,
		FromBlockHash:  hash15,
		Hash:           hash33,
	}
	vm = NewVM(nil)
	//vm.Debug = true
	db.addr = addr3
	receiveCancelVoteBlock, isRetry, err := vm.RunV2(db, block33, sendCancelVoteBlock.AccountBlock, NewTestGlobalStatus(0, snapshot2))
	if receiveCancelVoteBlock == nil ||
		len(receiveCancelVoteBlock.AccountBlock.SendBlockList) != 0 || isRetry || err != nil ||
		len(db.storageMap[addr3][ToKey(voteKey)]) != 0 ||
		len(receiveCancelVoteBlock.AccountBlock.Data) != 33 ||
		receiveCancelVoteBlock.AccountBlock.Data[32] != byte(0) ||
		receiveCancelVoteBlock.AccountBlock.Quota != 0 ||
		receiveCancelVoteBlock.AccountBlock.Quota != receiveCancelVoteBlock.AccountBlock.QuotaUsed {
		t.Fatalf("receive cancel vote transaction error")
	}
	db.accountBlockMap[addr3][hash33] = receiveCancelVoteBlock.AccountBlock
}

func TestCheckTokenName(t *testing.T) {
	tests := []struct {
		data string
		exp  bool
	}{
		{"", false},
		{" ", false},
		{"a", true},
		{"ab", true},
		{"ab ", false},
		{"a b", true},
		{"a  b", false},
		{"a _b", true},
		{"_a", true},
		{"_a b c", true},
		{"_a bb c", true},
		{"_a bb cc", true},
		{"_a bb  cc", false},
	}
	for _, test := range tests {
		if ok, _ := regexp.MatchString("^([0-9a-zA-Z_]+[ ]?)*[0-9a-zA-Z_]$", test.data); ok != test.exp {
			t.Fatalf("match string error, [%v] expected %v, got %v", test.data, test.exp, ok)
		}
	}
}

func TestGenesisBlockData(t *testing.T) {
	tokenName := "ViteToken"
	tokenSymbol := "ViteToken"
	decimals := uint8(18)
	totalSupply := new(big.Int).Mul(big.NewInt(1e18), big.NewInt(1e9))
	viteAddress, _, _ := types.CreateAddress()
	issueData, err := abi.ABIAsset.PackVariable(abi.VariableNameTokenInfo, tokenName, tokenSymbol, totalSupply, decimals, viteAddress, true, helper.Tt256m1, false, uint16(0))
	if err != nil {
		t.Fatalf("pack issue data error, %v", err)
	}
	fmt.Println("-------------mintage genesis block-------------")
	fmt.Printf("address: %v\n", hex.EncodeToString(types.AddressAsset.Bytes()))
	fmt.Printf("AccountBlock{\n\tBlockType: %v\n\tAccountAddress: %v,\n\tHeight: %v,\n\tAmount: %v,\n\tTokenId:ledger.ViteTokenId,\n\tQuota:0,\n\tFee:%v\n}\n",
		ledger.BlockTypeReceive, hex.EncodeToString(types.AddressAsset.Bytes()), 1, big.NewInt(0), big.NewInt(0))
	fmt.Printf("Storage:{\n\t%v:%v\n}\n", hex.EncodeToString(abi.GetTokenInfoKey(ledger.ViteTokenId)), hex.EncodeToString(issueData))

	fmt.Println("-------------vite owner genesis block-------------")
	fmt.Println("address: viteAddress")
	fmt.Printf("AccountBlock{\n\tBlockType: %v,\n\tAccountAddress: viteAddress,\n\tHeight: %v,\n\tAmount: %v,\n\tTokenId:ledger.ViteTokenId,\n\tQuota:0,\n\tFee:%v,\n\tData:%v,\n}\n",
		ledger.BlockTypeReceive, 1, totalSupply, big.NewInt(0), []byte{})
	fmt.Printf("Storage:{\n\t$balance:ledger.ViteTokenId:%v\n}\n", totalSupply)

	conditionRegisterData, err := abi.ABIGovernance.PackVariable(abi.VariableNameRegisterStakeParam, new(big.Int).Mul(big.NewInt(1e5), util.AttovPerVite), ledger.ViteTokenId, uint64(3600*24*90))
	if err != nil {
		t.Fatalf("pack register condition variable error, %v", err)
	}
	snapshotConsensusGroupData, err := abi.ABIGovernance.PackVariable(abi.VariableNameConsensusGroupInfo,
		uint8(25),
		int64(1),
		int64(3),
		uint8(2),
		uint8(50),
		uint16(1),
		uint8(0),
		ledger.ViteTokenId,
		uint8(1),
		conditionRegisterData,
		uint8(1),
		[]byte{},
		viteAddress,
		big.NewInt(0),
		uint64(1))
	if err != nil {
		t.Fatalf("pack consensus group data variable error, %v", err)
	}
	commonConsensusGroupData, err := abi.ABIGovernance.PackVariable(abi.VariableNameConsensusGroupInfo,
		uint8(25),
		int64(3),
		int64(1),
		uint8(2),
		uint8(50),
		uint16(48),
		uint8(1),
		ledger.ViteTokenId,
		uint8(1),
		conditionRegisterData,
		uint8(1),
		[]byte{},
		viteAddress,
		big.NewInt(0),
		uint64(1))
	if err != nil {
		t.Fatalf("pack consensus group data variable error, %v", err)
	}
	fmt.Println("-------------snapshot consensus group and common consensus group genesis block-------------")
	fmt.Printf("address:%v\n", hex.EncodeToString(types.AddressGovernance.Bytes()))
	fmt.Printf("AccountBlock{\n\tBlockType: %v,\n\tAccountAddress: %v,\n\tHeight: %v,\n\tAmount: %v,\n\tTokenId:ledger.ViteTokenId,\n\tQuota:0,\n\tFee:%v,\n\tData:%v,\n}\n",
		ledger.BlockTypeReceive, hex.EncodeToString(types.AddressGovernance.Bytes()), 1, big.NewInt(0), big.NewInt(0), []byte{})
	fmt.Printf("Storage:{\n\t%v:%v,\n\t%v:%v}\n", hex.EncodeToString(abi.GetConsensusGroupInfoKey(types.SNAPSHOT_GID)), hex.EncodeToString(snapshotConsensusGroupData), hex.EncodeToString(abi.GetConsensusGroupInfoKey(types.DELEGATE_GID)), hex.EncodeToString(commonConsensusGroupData))

	fmt.Println("-------------snapshot consensus group and common consensus group register genesis block-------------")
	fmt.Printf("address:%v\n", hex.EncodeToString(types.AddressGovernance.Bytes()))
	fmt.Printf("AccountBlock{\n\tBlockType: %v,\n\tAccountAddress: %v,\n\tHeight: %v,\n\tAmount: %v,\n\tTokenId:ledger.ViteTokenId,\n\tQuota:0,\n\tFee:%v,\n\tData:%v,\n}\n",
		ledger.BlockTypeReceive, hex.EncodeToString(types.AddressGovernance.Bytes()), 1, big.NewInt(0), big.NewInt(0), []byte{})
	fmt.Printf("Storage:{\n")
	for i := 1; i <= 25; i++ {
		addr, _, _ := types.CreateAddress()
		registerData, err := abi.ABIGovernance.PackVariable(abi.VariableNameRegistrationInfo, "node"+strconv.Itoa(i), addr, addr, helper.Big0, uint64(1), int64(1), int64(0), []types.Address{addr})
		if err != nil {
			t.Fatalf("pack registration variable error, %v", err)
		}
		snapshotKey := abi.GetRegistrationInfoKey("snapshotNode1", types.SNAPSHOT_GID)
		fmt.Printf("\t%v: %v\n", hex.EncodeToString(snapshotKey), hex.EncodeToString(registerData))
	}
	fmt.Println("}")
}

type emptyConsensusReaderTest struct {
	ti        timeIndex
	detailMap map[uint64]map[string]*ConsensusDetail
}

type ConsensusDetail struct {
	BlockNum         uint64
	ExpectedBlockNum uint64
	VoteCount        *big.Int
}

func newConsensusReaderTest(genesisTime int64, interval int64, detailMap map[uint64]map[string]*ConsensusDetail) *emptyConsensusReaderTest {
	return &emptyConsensusReaderTest{timeIndex{time.Unix(genesisTime, 0), time.Second * time.Duration(interval)}, detailMap}
}

func (r *emptyConsensusReaderTest) DayStats(startIndex uint64, endIndex uint64) ([]*core.DayStats, error) {
	list := make([]*core.DayStats, 0)
	if len(r.detailMap) == 0 {
		return list, nil
	}
	for i := startIndex; i <= endIndex; i++ {
		if i > endIndex {
			break
		}
		m, ok := r.detailMap[i]
		if !ok {
			continue
		}
		blockNum := uint64(0)
		expectedBlockNum := uint64(0)
		voteCount := big.NewInt(0)
		statusMap := make(map[string]*core.SbpStats, len(m))
		for name, detail := range m {
			blockNum = blockNum + detail.BlockNum
			expectedBlockNum = expectedBlockNum + detail.ExpectedBlockNum
			voteCount.Add(voteCount, detail.VoteCount)
			statusMap[name] = &core.SbpStats{i, detail.BlockNum, detail.ExpectedBlockNum, &core.BigInt{detail.VoteCount}, name}
		}
		list = append(list, &core.DayStats{Index: i, Stats: statusMap, VoteSum: &core.BigInt{voteCount}, BlockTotal: blockNum})
	}
	return list, nil
}
func (r *emptyConsensusReaderTest) GetDayTimeIndex() core.TimeIndex {
	return r.ti
}

type timeIndex struct {
	GenesisTime time.Time
	Interval    time.Duration
}

func (ti timeIndex) Index2Time(index uint64) (time.Time, time.Time) {
	sTime := ti.GenesisTime.Add(ti.Interval * time.Duration(index))
	eTime := ti.GenesisTime.Add(ti.Interval * time.Duration(index+1))
	return sTime, eTime
}
func (ti timeIndex) Time2Index(t time.Time) uint64 {
	subSec := int64(t.Sub(ti.GenesisTime).Seconds())
	i := uint64(subSec) / uint64(ti.Interval.Seconds())
	return i
}
