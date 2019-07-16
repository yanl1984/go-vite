package chain

import (
	"github.com/golang/protobuf/proto"
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/ledger"
	"github.com/vitelabs/go-vite/vm/contracts/abi"
	"github.com/vitelabs/go-vite/vm/contracts/dex"
	dexproto "github.com/vitelabs/go-vite/vm/contracts/dex/proto"
	"math/big"
	"testing"
)

var (
	dexAddr, _ = types.HexToAddress("vite_56fd05b23ff26cd7b0a40957fb77bde60c9fd6ebc35f809c23")
	tradeToken, _ = types.HexToTokenTypeId("tti_045e6ca837c143cd477b32f3")

)

func BenchmarkVMDexDepositSend(b *testing.B) {
	sendBlock := makeDexDepositBlock(testAddr)
	benchmarkSend(b, sendBlock)
}
func BenchmarkVMDexDepositReceive(b *testing.B) {
	sendBlock := makeDexDepositBlock(testAddr)
	receiveBlock := makeReceiveBlock(types.AddressDexFund)
	benchmarkReceive(b, sendBlock, receiveBlock)
}
func makeDexDepositBlock(addr types.Address) *ledger.AccountBlock {
	data, err := abi.ABIDexFund.PackMethod(abi.MethodNameDexFundUserDeposit)
	if err != nil {
		panic(err)
	}
	return makeSendBlock(addr, types.AddressDexFund, data, big1e18, big0)
}

func BenchmarkVMDexWithdrawSend(b *testing.B) {
	sendBlock := makeDexWithdrawBlock(testAddr)
	benchmarkSend(b, sendBlock)
}
func BenchmarkVMDexWithdrawReceive(b *testing.B) {
	sendBlock := makeDexWithdrawBlock(testAddr)
	receiveBlock := makeReceiveBlock(types.AddressDexFund)
	benchmarkReceive(b, sendBlock, receiveBlock)
}
func makeDexWithdrawBlock(addr types.Address) *ledger.AccountBlock {
	data, err := abi.ABIDexFund.PackMethod(abi.MethodNameDexFundUserWithdraw, ledger.ViteTokenId, big1e18)
	if err != nil {
		panic(err)
	}
	return makeSendBlock(addr, types.AddressDexFund, data, big0, big0)
}

func BenchmarkVMDexNewOrderSend(b *testing.B) {
	sendBlock := makeDexNewOrderBlock(testAddr)
	benchmarkSend(b, sendBlock)
}
func BenchmarkVMDexNewOrderReceive(b *testing.B) {
	sendBlock := makeDexNewOrderBlock(testAddr)
	receiveBlock := makeReceiveBlock(types.AddressDexFund)
	benchmarkReceive(b, sendBlock, receiveBlock)
}
func makeDexNewOrderBlock(addr types.Address) *ledger.AccountBlock {
	data, err := abi.ABIDexFund.PackMethod(abi.MethodNameDexFundNewOrder, tradeToken, ledger.ViteTokenId, true, uint8(dex.Limited), "30", new(big.Int).Mul(new(big.Int).SetInt64(1e18), big.NewInt(400)))
	if err != nil {
		panic(err)
	}
	return makeSendBlock(addr, types.AddressDexFund, data, big1e18, big0)
}

func BenchmarkVMDexSettleOrdersSend(b *testing.B) {
	sendBlock := makeDexSettleOrdersBlock(testAddr)
	benchmarkSend(b, sendBlock)
}
func BenchmarkVMDexSettleOrdersReceive(b *testing.B) {
	sendBlock := makeDexSettleOrdersBlock(testAddr)
	receiveBlock := makeReceiveBlock(types.AddressDexFund)
	benchmarkReceive(b, sendBlock, receiveBlock)
}
func makeDexSettleOrdersBlock(addr types.Address) *ledger.AccountBlock {
	settleActions := &dexproto.SettleActions{}
	settleActions.TradeToken = tradeToken.Bytes()
	settleActions.QuoteToken = ledger.ViteTokenId.Bytes()
	fundSettles1 := &dexproto.UserFundSettle{}
	fundSt1 := &dexproto.FundSettle{}
	fundSt1.IsTradeToken = true
	fundSt1.IncAvailable = big1e18.Bytes()
	fundSt1.ReduceLocked = big1e18.Bytes()
	fundSt1.ReleaseLocked = big0.Bytes()
	fundSt2 := &dexproto.FundSettle{}
	fundSt2.IsTradeToken = false
	fundSt2.IncAvailable = big1e18.Bytes()
	fundSt2.ReduceLocked = big1e18.Bytes()
	fundSt2.ReleaseLocked = big0.Bytes()
	fundSettles1.Address = addr.Bytes()
	fundSettles1.FundSettles = []*dexproto.FundSettle{fundSt1, fundSt2}
	fundSettles2 := &dexproto.UserFundSettle{}
	fundSettles2.Address = addr.Bytes()
	fundSettles2.FundSettles =  []*dexproto.FundSettle{fundSt1, fundSt2}
	settleActions.FundActions = []*dexproto.UserFundSettle{fundSettles1, fundSettles2}

	feeSettle1 := &dexproto.UserFeeSettle{}
	feeSettle1.Address = dexAddr.Bytes()
	feeSettle1.BaseFee = big1e18.Bytes()
	feeSettle1.BrokerFee = big1e18.Bytes()

	feeSettle2 := &dexproto.UserFeeSettle{}
	feeSettle2.Address = dexAddr.Bytes()
	feeSettle2.BaseFee = big1e18.Bytes()
	feeSettle2.BrokerFee = big1e18.Bytes()
	settleActions.FeeActions = []*dexproto.UserFeeSettle{feeSettle1, feeSettle2}
	settleData, err := proto.Marshal(settleActions)
	if err != nil {
		panic(err)
	}
	data, err := abi.ABIDexFund.PackMethod(abi.MethodNameDexFundSettleOrders, settleData)
	if err != nil {
		panic(err)
	}
	return makeSendBlock(types.AddressDexTrade, types.AddressDexFund, data, big0, big0)
}

func BenchmarkVmDexPeriodJobSend(b *testing.B) {
	sendBlock := makeDexDexPeriodJobBlock(testAddr)
	benchmarkSend(b, sendBlock)
}
func BenchmarkVMDexPeriodJobReceive(b *testing.B) {
	sendBlock := makeDexDexPeriodJobBlock(testAddr)
	receiveBlock := makeReceiveBlock(types.AddressDexFund)
	benchmarkReceive(b, sendBlock, receiveBlock)
}
func makeDexDexPeriodJobBlock(addr types.Address) *ledger.AccountBlock {
	data, err := abi.ABIDexFund.PackMethod(abi.MethodNameDexFundPeriodJob, uint64(0), uint8(0))
	if err != nil {
		panic(err)
	}
	return makeSendBlock(addr, types.AddressDexFund, data, big0, big0)
}

func BenchmarkVmDexNewMarketSend(b *testing.B) {
	sendBlock := makeDexDexNewMarketBlock(testAddr)
	benchmarkSend(b, sendBlock)
}
func BenchmarkVMDexNewMarketReceive(b *testing.B) {
	sendBlock := makeDexDexNewMarketBlock(testAddr)
	receiveBlock := makeReceiveBlock(types.AddressDexFund)
	benchmarkReceive(b, sendBlock, receiveBlock)
}
func makeDexDexNewMarketBlock(addr types.Address) *ledger.AccountBlock {
	data, err := abi.ABIDexFund.PackMethod(abi.MethodNameDexFundNewMarket, uint64(0), uint8(0))
	if err != nil {
		panic(err)
	}
	return makeSendBlock(addr, types.AddressDexFund, data, big0, big0)
}

func BenchmarkVMDexPledgeForVxSend(b *testing.B) {
	sendBlock := makeDexPledgeForVxBlock(testAddr)
	benchmarkSend(b, sendBlock)
}
func BenchmarkVMDexPledgeForVxReceive(b *testing.B) {
	sendBlock := makeDexPledgeForVxBlock(testAddr)
	receiveBlock := makeReceiveBlock(types.AddressDexFund)
	benchmarkReceive(b, sendBlock, receiveBlock)
}
func makeDexPledgeForVxBlock(addr types.Address) *ledger.AccountBlock {
	amount := new(big.Int).Mul(big.NewInt(500), big1e18)
	data, err := abi.ABIDexFund.PackMethod(abi.MethodNameDexFundPledgeForVx, uint8(1), amount)
	if err != nil {
		panic(err)
	}
	return makeSendBlock(addr, types.AddressDexFund, data, big0, big0)
}

func BenchmarkVMDexPledgeForVipSend(b *testing.B) {
	sendBlock := makeDexPledgeForVipBlock(dexAddr)
	benchmarkSend(b, sendBlock)
}
func BenchmarkVMDexPledgeForVipReceive(b *testing.B) {
	sendBlock := makeDexPledgeForVipBlock(dexAddr)
	receiveBlock := makeReceiveBlock(types.AddressDexFund)
	benchmarkReceive(b, sendBlock, receiveBlock)
}
func makeDexPledgeForVipBlock(addr types.Address) *ledger.AccountBlock {
	data, err := abi.ABIDexFund.PackMethod(abi.MethodNameDexFundPledgeForVip, uint8(1))
	if err != nil {
		panic(err)
	}
	return makeSendBlock(addr, types.AddressDexFund, data, big0, big0)
}

func BenchmarkVMDexPledgeCallBackSend(b *testing.B) {
	sendBlock := makeDexPledgePledgeCallBackBlock(dexAddr)
	benchmarkSend(b, sendBlock)
}
func BenchmarkVMDexPledgeCallBackReceive(b *testing.B) {
	sendBlock := makeDexPledgePledgeCallBackBlock(dexAddr)
	receiveBlock := makeReceiveBlock(types.AddressDexFund)
	benchmarkReceive(b, sendBlock, receiveBlock)
}
func makeDexPledgePledgeCallBackBlock(addr types.Address) *ledger.AccountBlock {
	amount := new(big.Int).Mul(big.NewInt(500), big1e18)
	data, err := abi.ABIPledge.PackCallback(abi.MethodNameAgentPledge, addr, types.AddressDexFund, amount, uint8(dex.PledgeForVx), true)
	if err != nil {
		panic(err)
	}
	return makeSendBlock(types.AddressPledge, types.AddressDexFund, data, big0, big0)
}

func BenchmarkVMDexCancelPledgeCallBackSend(b *testing.B) {
	sendBlock := makeDexCancelPledgePledgeCallBackBlock(dexAddr)
	benchmarkSend(b, sendBlock)
}
func BenchmarkVMDexCancelPledgeCallBackReceive(b *testing.B) {
	sendBlock := makeDexCancelPledgePledgeCallBackBlock(dexAddr)
	receiveBlock := makeReceiveBlock(types.AddressDexFund)
	benchmarkReceive(b, sendBlock, receiveBlock)
}
func makeDexCancelPledgePledgeCallBackBlock(addr types.Address) *ledger.AccountBlock {
	amount := new(big.Int).Mul(big.NewInt(500), big1e18)
	data, err := abi.ABIPledge.PackCallback(abi.MethodNameAgentCancelPledge, addr, types.AddressDexFund, amount, uint8(dex.PledgeForVx), true)
	if err != nil {
		panic(err)
	}
	return makeSendBlock(types.AddressPledge, types.AddressDexFund, data, amount, big0)
}

func BenchmarkVMDexCancelPledgeCallBackSend(b *testing.B) {
	sendBlock := makeDexCancelPledgePledgeCallBackBlock(dexAddr)
	benchmarkSend(b, sendBlock)
}
func BenchmarkVMDexCancelPledgeCallBackReceive(b *testing.B) {
	sendBlock := makeDexCancelPledgePledgeCallBackBlock(dexAddr)
	receiveBlock := makeReceiveBlock(types.AddressDexFund)
	benchmarkReceive(b, sendBlock, receiveBlock)
}
func makeDexCancelPledgePledgeCallBackBlock(addr types.Address) *ledger.AccountBlock {
	amount := new(big.Int).Mul(big.NewInt(500), big1e18)
	data, err := abi.ABIPledge.PackCallback(abi.MethodNameAgentCancelPledge, addr, types.AddressDexFund, amount, uint8(dex.PledgeForVx), true)
	if err != nil {
		panic(err)
	}
	return makeSendBlock(types.AddressPledge, types.AddressDexFund, data, amount, big0)
}