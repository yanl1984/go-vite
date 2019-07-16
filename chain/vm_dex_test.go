package chain

import (
	"encoding/base64"
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
	dexAddr, _                = types.HexToAddress("vite_56fd05b23ff26cd7b0a40957fb77bde60c9fd6ebc35f809c23")
	tradeToken, _             = types.HexToTokenTypeId("tti_045e6ca837c143cd477b32f3")
	newTradeToken, _          = types.HexToTokenTypeId("tti_2736f320d7ed1c2871af1d9d")
	notExistsNewTradeToken, _ = types.HexToTokenTypeId("tti_060e61a9f5222c0fcc0c7ff5")
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

func BenchmarkVMDexNewMarketSend(b *testing.B) {
	sendBlock := makeDexNewMarketBlock(testAddr)
	benchmarkSend(b, sendBlock)
}
func BenchmarkVMDexNewMarketReceive(b *testing.B) {
	sendBlock := makeDexNewMarketBlock(testAddr)
	receiveBlock := makeReceiveBlock(types.AddressDexFund)
	benchmarkReceive(b, sendBlock, receiveBlock)
}
func makeDexNewMarketBlock(addr types.Address) *ledger.AccountBlock {
	data, err := abi.ABIDexFund.PackMethod(abi.MethodNameDexFundNewMarket, newTradeToken, ledger.ViteTokenId)
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
	fundSettles2.FundSettles = []*dexproto.FundSettle{fundSt1, fundSt2}
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
	sendBlock := makeDexDexPeriodJobBlock(dexAddr)
	benchmarkSend(b, sendBlock)
}
func BenchmarkVMDexPeriodJobReceive(b *testing.B) {
	sendBlock := makeDexDexPeriodJobBlock(dexAddr)
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
	amount := new(big.Int).Mul(big.NewInt(10000), big1e18)
	data, err := abi.ABIDexFund.PackMethod(abi.MethodNameDexFundPledgeForVip, uint8(1))
	if err != nil {
		panic(err)
	}
	return makeSendBlock(addr, types.AddressDexFund, data, amount, big0)
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
	sendBlock := makeDexCancelPledgePledgeCallBackBlock(testAddr)
	benchmarkSend(b, sendBlock)
}
func BenchmarkVMDexCancelPledgeCallBackReceive(b *testing.B) {
	sendBlock := makeDexCancelPledgePledgeCallBackBlock(testAddr)
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

func BenchmarkVMDexGetTokenInfoCallbackSend(b *testing.B) {
	sendBlock := makeDexGetTokenInfoCallBackBlock(types.AddressMintage)
	benchmarkSend(b, sendBlock)
}
func BenchmarkVMDexGetTokenInfoCallBackReceive(b *testing.B) {
	sendBlock := makeDexGetTokenInfoCallBackBlock(types.AddressMintage)
	receiveBlock := makeReceiveBlock(types.AddressDexFund)
	benchmarkReceive(b, sendBlock, receiveBlock)
}
func makeDexGetTokenInfoCallBackBlock(addr types.Address) *ledger.AccountBlock {
	data, err := abi.ABIMintage.PackCallback(abi.MethodNameGetTokenInfo, notExistsNewTradeToken, uint8(dex.GetTokenForTransferOwner), true, uint8(18), "VCC", uint16(1), testAddr)
	if err != nil {
		panic(err)
	}
	return makeSendBlock(addr, types.AddressDexFund, data, big0, big0)
}

func BenchmarkVMDexNotifyTimeSend(b *testing.B) {
	sendBlock := makeDexNotifyTimeBlock(dexAddr)
	benchmarkSend(b, sendBlock)
}
func BenchmarkVMDexNotifyTimeReceive(b *testing.B) {
	sendBlock := makeDexNotifyTimeBlock(dexAddr)
	receiveBlock := makeReceiveBlock(types.AddressDexFund)
	benchmarkReceive(b, sendBlock, receiveBlock)
}
func makeDexNotifyTimeBlock(addr types.Address) *ledger.AccountBlock {
	data, err := abi.ABIDexFund.PackMethod(abi.MethodNameDexFundNotifyTime, int64(1563187468))
	if err != nil {
		panic(err)
	}
	return makeSendBlock(addr, types.AddressDexFund, data, big0, big0)
}

func BenchmarkVMDexNewInviterSend(b *testing.B) {
	sendBlock := makeDexNewInviterBlock(testAddr)
	benchmarkSend(b, sendBlock)
}
func BenchmarkVMDexNewInviterReceive(b *testing.B) {
	sendBlock := makeDexNewInviterBlock(testAddr)
	receiveBlock := makeReceiveBlock(types.AddressDexFund)
	benchmarkReceive(b, sendBlock, receiveBlock)
}
func makeDexNewInviterBlock(addr types.Address) *ledger.AccountBlock {
	amount := new(big.Int).Mul(big.NewInt(1000), big1e18)
	data, err := abi.ABIDexFund.PackMethod(abi.MethodNameDexFundNewInviter)
	if err != nil {
		panic(err)
	}
	return makeSendBlock(addr, types.AddressDexFund, data, amount, big0)
}

func BenchmarkVMDexBindInviteCodeSend(b *testing.B) {
	sendBlock := makeDexBindInviteCodeBlock(testAddr)
	benchmarkSend(b, sendBlock)
}
func BenchmarkVMDexBindInviteCodeReceive(b *testing.B) {
	sendBlock := makeDexBindInviteCodeBlock(testAddr)
	receiveBlock := makeReceiveBlock(types.AddressDexFund)
	benchmarkReceive(b, sendBlock, receiveBlock)
}
func makeDexBindInviteCodeBlock(addr types.Address) *ledger.AccountBlock {
	data, err := abi.ABIDexFund.PackMethod(abi.MethodNameDexFundBindInviteCode, uint32(123))
	if err != nil {
		panic(err)
	}
	return makeSendBlock(addr, types.AddressDexFund, data, big0, big0)
}

func BenchmarkVMDexEndorseVxSend(b *testing.B) {
	sendBlock := makeDexEndorseVxBlock(testAddr)
	benchmarkSend(b, sendBlock)
}
func BenchmarkVMEndorseVxReceive(b *testing.B) {
	sendBlock := makeDexEndorseVxBlock(testAddr)
	receiveBlock := makeReceiveBlock(types.AddressDexFund)
	benchmarkReceive(b, sendBlock, receiveBlock)
}
func makeDexEndorseVxBlock(addr types.Address) *ledger.AccountBlock {
	amount := new(big.Int).Mul(big.NewInt(10000), big1e18)
	data, err := abi.ABIDexFund.PackMethod(abi.MethodNameDexFundEndorseVxMinePool)
	if err != nil {
		panic(err)
	}
	block := makeSendBlock(addr, types.AddressDexFund, data, amount, big0)
	block.TokenId = dex.VxTokenId
	return block
}

func BenchmarkVMDexSettleMakerMinedVxSend(b *testing.B) {
	sendBlock := makeDexSettleMakerMinedVxBlock(dexAddr)
	benchmarkSend(b, sendBlock)
}
func BenchmarkVMSettleMakerMinedVxReceive(b *testing.B) {
	sendBlock := makeDexSettleMakerMinedVxBlock(dexAddr)
	receiveBlock := makeReceiveBlock(types.AddressDexFund)
	benchmarkReceive(b, sendBlock, receiveBlock)
}
func makeDexSettleMakerMinedVxBlock(addr types.Address) *ledger.AccountBlock {
	amount := new(big.Int).Mul(big.NewInt(10), big1e18)
	actions := &dexproto.VxSettleActions{}
	actions.Period = 1
	actions.Page = 1
	action := &dexproto.VxSettleAction{}
	action.Address = testAddr.Bytes()
	action.Amount = amount.Bytes()
	action1 := &dexproto.VxSettleAction{}
	action1.Address = dexAddr.Bytes()
	action1.Amount = amount.Bytes()
	actions.Actions = []*dexproto.VxSettleAction{action, action1}
	actionsData, err := proto.Marshal(actions)
	data, err := abi.ABIDexFund.PackMethod(abi.MethodNameDexFunSettleMakerMinedVx, actionsData)
	if err != nil {
		panic(err)
	}
	return makeSendBlock(addr, types.AddressDexFund, data, amount, big0)
}

func BenchmarkVMDexTradeNewOrderSend(b *testing.B) {
	sendBlock := makeDexTradeNewOrderBlock(types.AddressDexFund)
	benchmarkSend(b, sendBlock)
}
func BenchmarkVMDexTradeNewOrderReceive(b *testing.B) {
	sendBlock := makeDexTradeNewOrderBlock(types.AddressDexFund)
	receiveBlock := makeReceiveBlock(types.AddressDexTrade)
	benchmarkReceive(b, sendBlock, receiveBlock)
}
func makeDexTradeNewOrderBlock(addr types.Address) *ledger.AccountBlock {
	order := &dex.Order{}
	order.Id, _ = base64.StdEncoding.DecodeString("AAABAP/////h//////8AXSoSsgAAEA==")
	order.Address = testAddr.Bytes()
	order.MarketId = 1
	order.Side = false
	order.Type = 1
	order.Price = dex.PriceToBytes("20")
	order.TakerFeeRate = 200
	order.MakerFeeRate = 200
	order.TakerBrokerFeeRate = 100
	order.MakerBrokerFeeRate = 100
	order.Quantity = big1e18.Bytes()
	order.Amount = big1e18.Bytes()
	order.LockedBuyFee = big0.Bytes()
	order.Status = dex.Pending
	order.Timestamp = 1563187568
	if orderInfoBytes, err := order.Serialize(); err != nil {
		panic(err)
	} else {
		if tradeBlockData, err := abi.ABIDexTrade.PackMethod(abi.MethodNameDexTradeNewOrder, orderInfoBytes); err != nil {
			panic(err)
		} else {
			return makeSendBlock(addr, types.AddressDexTrade, tradeBlockData, big0, big0)
		}
	}
}

func BenchmarkVMDexTradeCancelOrderSend(b *testing.B) {
	sendBlock := makeDexTradeCancelOrderBlock(testAddr)
	benchmarkSend(b, sendBlock)
}
func BenchmarkVMDexTradeCancelOrderReceive(b *testing.B) {
	sendBlock := makeDexTradeCancelOrderBlock(testAddr)
	receiveBlock := makeReceiveBlock(types.AddressDexTrade)
	benchmarkReceive(b, sendBlock, receiveBlock)
}
func makeDexTradeCancelOrderBlock(addr types.Address) *ledger.AccountBlock {
	orderId, _ := base64.StdEncoding.DecodeString("AAABAP/////h//////8AXSoSsgAADw==")
	data, err := abi.ABIDexTrade.PackMethod(abi.MethodNameDexTradeCancelOrder, orderId)
	if err != nil {
		panic(err)
	}
	return makeSendBlock(addr, types.AddressDexTrade, data, big0, big0)
}

func BenchmarkVMDexTradeNotifyNewMarketSend(b *testing.B) {
	sendBlock := makeDexTradeNotifyNewMarketBlock(types.AddressDexFund)
	benchmarkSend(b, sendBlock)
}
func BenchmarkVMDexTradeNotifyNewMarketReceive(b *testing.B) {
	sendBlock := makeDexTradeNotifyNewMarketBlock(types.AddressDexFund)
	receiveBlock := makeReceiveBlock(types.AddressDexTrade)
	benchmarkReceive(b, sendBlock, receiveBlock)
}
func makeDexTradeNotifyNewMarketBlock(addr types.Address) *ledger.AccountBlock {
	marketInfo := &dex.MarketInfo{}
	marketInfo.MarketId = 2
	marketInfo.MarketSymbol = "ABC_001-VITE"
	marketInfo.TradeToken = notExistsNewTradeToken.Bytes()
	marketInfo.QuoteToken = ledger.ViteTokenId.Bytes()
	marketInfo.QuoteTokenType = 1
	marketInfo.TradeTokenDecimals = 18
	marketInfo.QuoteTokenDecimals = 18
	marketInfo.TakerBrokerFeeRate = 18
	marketInfo.MakerBrokerFeeRate = 18
	marketInfo.AllowMine = true
	marketInfo.Valid = true
	marketInfo.Owner = testAddr.Bytes()
	marketInfo.Creator = testAddr.Bytes()
	marketInfo.Stopped = false
	marketInfo.Timestamp = 1563187568
	if marketBytes, err := marketInfo.Serialize(); err != nil {
		panic(err)
	} else {
		if syncData, err := abi.ABIDexTrade.PackMethod(abi.MethodNameDexTradeNotifyNewMarket, marketBytes); err != nil {
			panic(err)
		} else {
			return makeSendBlock(addr, types.AddressDexTrade, syncData, big0, big0)
		}
	}
}

func BenchmarkVMDexTradeCleanExpireOrdersSend(b *testing.B) {
	sendBlock := makeDexTradeCleanExpireOrdersBlock(dexAddr)
	benchmarkSend(b, sendBlock)
}
func BenchmarkVMDexTradeCleanExpireOrdersReceive(b *testing.B) {
	sendBlock := makeDexTradeCleanExpireOrdersBlock(dexAddr)
	receiveBlock := makeReceiveBlock(types.AddressDexTrade)
	benchmarkReceive(b, sendBlock, receiveBlock)
}
func makeDexTradeCleanExpireOrdersBlock(addr types.Address) *ledger.AccountBlock {
	orderId, _ := base64.StdEncoding.DecodeString("AAABAP/////h//////8AXSoSsgAADw==")
	idsData := make([]byte, 0, 2*dex.OrderIdBytesLength)
	idsData = append(idsData, orderId...)
	idsData = append(idsData, orderId...)
	idsData = append(idsData, orderId...)
	if syncData, err := abi.ABIDexTrade.PackMethod(abi.MethodNameDexTradeCleanExpireOrders, idsData); err != nil {
		panic(err)
	} else {
		return makeSendBlock(addr, types.AddressDexTrade, syncData, big0, big0)
	}
}
