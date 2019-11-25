package chain

import (
	"encoding/base64"
	"encoding/hex"
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
	tradeToken1, _            = types.HexToTokenTypeId("tti_4d3a69b12962332e8df52701")
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

//part of NewMarket[entry]
func BenchmarkVMDexNewMarketSend(b *testing.B) {
	sendBlock := makeDexNewMarketBlock(testAddr)
	benchmarkSend(b, sendBlock)
}

//part of NewMarket
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

// part of NewOrder[entry]
func BenchmarkVMDexNewOrderSend(b *testing.B) {
	sendBlock := makeDexNewOrderBlock(testAddr)
	benchmarkSend(b, sendBlock)
}

// part of NewOrder
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

// part of NewOrder
// part of CancelOrder
// part of NewAgentOrder
// part of CancelOrderByHash
func BenchmarkVMDexSettleOrdersSend(b *testing.B) {
	sendBlock := makeDexSettleOrdersBlock(testAddr)
	benchmarkSend(b, sendBlock)
}

// part of NewOrder
// part of CancelOrder
// part of NewAgentOrder
// part of CancelOrderByHash
func BenchmarkVMDexSettleOrdersReceive(b *testing.B) {
	sendBlock := makeDexSettleOrdersBlock(testAddr)
	receiveBlock := makeReceiveBlock(types.AddressDexFund)
	benchmarkReceive(b, sendBlock, receiveBlock)
}
func makeDexSettleOrdersBlock(addr types.Address) *ledger.AccountBlock {
	settleActions := &dexproto.SettleActions{}
	settleActions.TradeToken = tradeToken.Bytes()
	settleActions.QuoteToken = ledger.ViteTokenId.Bytes()
	fundSettles1 := &dexproto.FundSettle{}
	accSt1 := &dexproto.AccountSettle{}
	accSt1.IsTradeToken = true
	accSt1.IncAvailable = big1e18.Bytes()
	accSt1.ReduceLocked = big1e18.Bytes()
	accSt1.ReleaseLocked = big0.Bytes()
	fundSt2 := &dexproto.AccountSettle{}
	fundSt2.IsTradeToken = false
	fundSt2.IncAvailable = big1e18.Bytes()
	fundSt2.ReduceLocked = big1e18.Bytes()
	fundSt2.ReleaseLocked = big0.Bytes()
	fundSettles1.Address = addr.Bytes()
	fundSettles1.AccountSettles = []*dexproto.AccountSettle{accSt1, fundSt2}
	fundSettles2 := &dexproto.FundSettle{}
	fundSettles2.Address = addr.Bytes()
	fundSettles2.AccountSettles = []*dexproto.AccountSettle{accSt1, fundSt2}
	settleActions.FundActions = []*dexproto.FundSettle{fundSettles1, fundSettles2}

	feeSettle1 := &dexproto.FeeSettle{}
	feeSettle1.Address = dexAddr.Bytes()
	feeSettle1.BaseFee = big1e18.Bytes()
	feeSettle1.OperatorFee = big1e18.Bytes()

	feeSettle2 := &dexproto.FeeSettle{}
	feeSettle2.Address = dexAddr.Bytes()
	feeSettle2.BaseFee = big1e18.Bytes()
	feeSettle2.OperatorFee = big1e18.Bytes()
	settleActions.FeeActions = []*dexproto.FeeSettle{feeSettle1, feeSettle2}
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
	sendBlock := makeDexPeriodJobBlock(dexAddr)
	benchmarkSend(b, sendBlock)
}
func BenchmarkVMDexPeriodJobReceive(b *testing.B) {
	sendBlock := makeDexPeriodJobBlock(dexAddr)
	receiveBlock := makeReceiveBlock(types.AddressDexFund)
	benchmarkReceive(b, sendBlock, receiveBlock)
}
func makeDexPeriodJobBlock(addr types.Address) *ledger.AccountBlock {
	data, err := abi.ABIDexFund.PackMethod(abi.MethodNameDexFundPeriodJob, uint64(0), uint8(0))
	if err != nil {
		panic(err)
	}
	return makeSendBlock(addr, types.AddressDexFund, data, big0, big0)
}

//part of StakeForMining[entry]
func BenchmarkVMDexStakeForMiningSend(b *testing.B) {
	sendBlock := makeDexStakeForMiningBlock(testAddr)
	benchmarkSend(b, sendBlock)
}

//part of StakeForMining
func BenchmarkVMDexStakeForMiningReceive(b *testing.B) {
	sendBlock := makeDexStakeForMiningBlock(testAddr)
	receiveBlock := makeReceiveBlock(types.AddressDexFund)
	benchmarkReceive(b, sendBlock, receiveBlock)
}
func makeDexStakeForMiningBlock(addr types.Address) *ledger.AccountBlock {
	amount := new(big.Int).Mul(big.NewInt(500), big1e18)
	data, err := abi.ABIDexFund.PackMethod(abi.MethodNameDexFundStakeForMining, uint8(1), amount)
	if err != nil {
		panic(err)
	}
	return makeSendBlock(addr, types.AddressDexFund, data, big0, big0)
}

//part of PledgeForVip[entry]
func BenchmarkVMDexPledgeForVipSend(b *testing.B) {
	sendBlock := makeDexPledgeForVipBlock(dexAddr)
	benchmarkSend(b, sendBlock)
}

//part of PledgeForVip
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

//part of StakeForMining
//part of PledgeForVip
func BenchmarkVMDexPledgeCallbackSend(b *testing.B) {
	sendBlock := makeDexPledgeCallbackBlock(dexAddr)
	benchmarkSend(b, sendBlock)
}

//part of StakeForMining
//part of PledgeForVip
func BenchmarkVMDexPledgeCallbackReceive(b *testing.B) {
	sendBlock := makeDexPledgeCallbackBlock(dexAddr)
	receiveBlock := makeReceiveBlock(types.AddressDexFund)
	benchmarkReceive(b, sendBlock, receiveBlock)
}
func makeDexPledgeCallbackBlock(addr types.Address) *ledger.AccountBlock {
	amount := new(big.Int).Mul(big.NewInt(500), big1e18)
	data, err := abi.ABIDexFund.PackMethod(abi.MethodNameDexFundDelegateStakeCallback, addr, types.AddressDexFund, amount, uint8(dex.StakeForMining), true)
	if err != nil {
		panic(err)
	}
	return makeSendBlock(types.AddressQuota, types.AddressDexFund, data, big0, big0)
}

//part of StakeForMining
//part of PledgeForVip
func BenchmarkVMDexCancelPledgeCallbackSend(b *testing.B) {
	sendBlock := makeDexCancelPledgeCallbackBlock(testAddr)
	benchmarkSend(b, sendBlock)
}

//part of StakeForMining
//part of PledgeForVip
func BenchmarkVMDexCancelPledgeCallBackReceive(b *testing.B) {
	sendBlock := makeDexCancelPledgeCallbackBlock(testAddr)
	receiveBlock := makeReceiveBlock(types.AddressDexFund)
	benchmarkReceive(b, sendBlock, receiveBlock)
}
func makeDexCancelPledgeCallbackBlock(addr types.Address) *ledger.AccountBlock {
	amount := new(big.Int).Mul(big.NewInt(500), big1e18)
	data, err := abi.ABIDexFund.PackMethod(abi.MethodNameDexFundCancelDelegateStakeCallback, addr, types.AddressDexFund, amount, uint8(dex.StakeForMining), true)
	if err != nil {
		panic(err)
	}
	return makeSendBlock(types.AddressQuota, types.AddressDexFund, data, amount, big0)
}

//part of NewMarket
func BenchmarkVMDexGetTokenInfoCallbackSend(b *testing.B) {
	sendBlock := makeDexGetTokenInfoCallbackBlock(types.AddressAsset)
	benchmarkSend(b, sendBlock)
}

//part of NewMarket
func BenchmarkVMDexGetTokenInfoCallBackReceive(b *testing.B) {
	sendBlock := makeDexGetTokenInfoCallbackBlock(types.AddressAsset)
	receiveBlock := makeReceiveBlock(types.AddressDexFund)
	benchmarkReceive(b, sendBlock, receiveBlock)
}
func makeDexGetTokenInfoCallbackBlock(addr types.Address) *ledger.AccountBlock {
	data, err := abi.ABIDexFund.PackMethod(abi.MethodNameDexFundGetTokenInfoCallback, notExistsNewTradeToken, uint8(dex.GetTokenForTransferOwner), true, uint8(18), "VCC", uint16(1), testAddr)
	if err != nil {
		panic(err)
	}
	return makeSendBlock(addr, types.AddressDexFund, data, big0, big0)
}

func BenchmarkVMDexOwnerConfigSend(b *testing.B) {
	sendBlock := makeDexOwnerConfigBlock(dexAddr)
	benchmarkSend(b, sendBlock)
}
func BenchmarkVMDexOwnerConfigReceive(b *testing.B) {
	sendBlock := makeDexOwnerConfigBlock(dexAddr)
	receiveBlock := makeReceiveBlock(types.AddressDexFund)
	benchmarkReceive(b, sendBlock, receiveBlock)
}
func makeDexOwnerConfigBlock(addr types.Address) *ledger.AccountBlock {
	data, err := abi.ABIDexFund.PackMethod(abi.MethodNameDexFundOwnerConfig, uint8(dex.AdminConfigOwner+dex.AdminConfigTimeOracle+dex.AdminConfigPeriodJobTrigger+dex.AdminConfigMakerMiningAdmin+dex.AdminConfigMaintainer), dexAddr, dexAddr, dexAddr, false, dexAddr, dexAddr)
	if err != nil {
		panic(err)
	}
	return makeSendBlock(addr, types.AddressDexFund, data, big0, big0)
}

func BenchmarkVMDexOwnerConfigTradeSend(b *testing.B) {
	sendBlock := makeDexOwnerConfigTradeBlock(dexAddr)
	benchmarkSend(b, sendBlock)
}
func BenchmarkVMDexOwnerConfigTradeReceive(b *testing.B) {
	sendBlock := makeDexOwnerConfigTradeBlock(dexAddr)
	receiveBlock := makeReceiveBlock(types.AddressDexFund)
	benchmarkReceive(b, sendBlock, receiveBlock)
}
func makeDexOwnerConfigTradeBlock(addr types.Address) *ledger.AccountBlock {
	data, err := abi.ABIDexFund.PackMethod(abi.MethodNameDexFundOwnerConfigTrade, uint8(dex.TradeAdminConfigMineMarket), tradeToken, ledger.ViteTokenId, false, ledger.ViteTokenId, uint8(1), uint8(1), big.NewInt(0), uint8(1), big.NewInt(0))
	if err != nil {
		panic(err)
	}
	return makeSendBlock(addr, types.AddressDexFund, data, big0, big0)
}

func BenchmarkVMDexMarketOwnerConfigSend(b *testing.B) {
	sendBlock := makeDexMarketOwnerConfigBlock(testAddr)
	benchmarkSend(b, sendBlock)
}
func BenchmarkVMDexMarketOwnerConfigReceive(b *testing.B) {
	sendBlock := makeDexMarketOwnerConfigBlock(testAddr)
	receiveBlock := makeReceiveBlock(types.AddressDexFund)
	benchmarkReceive(b, sendBlock, receiveBlock)
}
func makeDexMarketOwnerConfigBlock(addr types.Address) *ledger.AccountBlock {
	data, err := abi.ABIDexFund.PackMethod(abi.MethodNameDexFundMarketOwnerConfig, uint8(dex.MarketOwnerTransferOwner), tradeToken, ledger.ViteTokenId, dexAddr, int32(0), int32(0), false)
	if err != nil {
		panic(err)
	}
	return makeSendBlock(addr, types.AddressDexFund, data, big0, big0)
}

func BenchmarkVMDexTransferTokenOwnerSend(b *testing.B) {
	sendBlock := makeDexTransferTokenOwnerBlock(testAddr)
	benchmarkSend(b, sendBlock)
}
func BenchmarkVMDexTransferTokenOwnerReceive(b *testing.B) {
	sendBlock := makeDexTransferTokenOwnerBlock(testAddr)
	receiveBlock := makeReceiveBlock(types.AddressDexFund)
	benchmarkReceive(b, sendBlock, receiveBlock)
}
func makeDexTransferTokenOwnerBlock(addr types.Address) *ledger.AccountBlock {
	data, err := abi.ABIDexFund.PackMethod(abi.MethodNameDexFundTransferTokenOwner, tradeToken, dexAddr)
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
	sendBlock := makeDexNewInviterBlock(dexAddr)
	benchmarkSend(b, sendBlock)
}
func BenchmarkVMDexNewInviterReceive(b *testing.B) {
	sendBlock := makeDexNewInviterBlock(dexAddr)
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
func BenchmarkVMDexEndorseVxReceive(b *testing.B) {
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
func BenchmarkVMDexSettleMakerMinedVxReceive(b *testing.B) {
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
	data, err := abi.ABIDexFund.PackMethod(abi.MethodNameDexFundSettleMakerMinedVx, actionsData)
	if err != nil {
		panic(err)
	}
	return makeSendBlock(addr, types.AddressDexFund, data, amount, big0)
}

func BenchmarkVMDexPledgeForSuperVipSend(b *testing.B) {
	sendBlock := makeDexPledgeForSuperVipBlock(dexAddr)
	benchmarkSend(b, sendBlock)
}
func BenchmarkVMDexPledgeForSuperVipReceive(b *testing.B) {
	sendBlock := makeDexPledgeForSuperVipBlock(dexAddr)
	receiveBlock := makeReceiveBlock(types.AddressDexFund)
	benchmarkReceive(b, sendBlock, receiveBlock)
}
func makeDexPledgeForSuperVipBlock(addr types.Address) *ledger.AccountBlock {
	data, err := abi.ABIDexFund.PackMethod(abi.MethodNameDexFundStakeForSuperVip, uint8(dex.Stake))
	if err != nil {
		panic(err)
	}
	return makeSendBlock(addr, types.AddressDexFund, data, big0, big0)
}

func BenchmarkVMDexConfigMarketAgentSend(b *testing.B) {
	sendBlock := makeDexConfigMarketAgentBlock(dexAddr)
	benchmarkSend(b, sendBlock)
}
func BenchmarkVMDexConfigMarketAgentReceive(b *testing.B) {
	sendBlock := makeDexConfigMarketAgentBlock(dexAddr)
	receiveBlock := makeReceiveBlock(types.AddressDexFund)
	benchmarkReceive(b, sendBlock, receiveBlock)
}
func makeDexConfigMarketAgentBlock(addr types.Address) *ledger.AccountBlock {
	tradeTokens := []types.TokenTypeId{tradeToken, tradeToken1}
	quoteTokens := []types.TokenTypeId{ledger.ViteTokenId, ledger.ViteTokenId}
	data, err := abi.ABIDexFund.PackMethod(abi.MethodNameDexFundConfigMarketsAgent, uint8(dex.GrantAgent), dexAddr, tradeTokens, quoteTokens)
	if err != nil {
		panic(err)
	}
	return makeSendBlock(addr, types.AddressDexFund, data, big0, big0)
}

// part of NewAgentOrder[entry]
func BenchmarkVMNewAgentOrderSend(b *testing.B) {
	sendBlock := makeDexNewAgentOrderBlock(dexAddr)
	benchmarkSend(b, sendBlock)
}

// part of NewAgentOrder
func BenchmarkVMNewAgentOrderReceive(b *testing.B) {
	sendBlock := makeDexNewAgentOrderBlock(dexAddr)
	receiveBlock := makeReceiveBlock(types.AddressDexFund)
	benchmarkReceive(b, sendBlock, receiveBlock)
}
func makeDexNewAgentOrderBlock(addr types.Address) *ledger.AccountBlock {
	data, err := abi.ABIDexFund.PackMethod(abi.MethodNameDexFundNewAgentOrder, testAddr, tradeToken, ledger.ViteTokenId, true, uint8(dex.Limited), "30", new(big.Int).Mul(new(big.Int).SetInt64(1e18), big.NewInt(400)))
	if err != nil {
		panic(err)
	}
	return makeSendBlock(addr, types.AddressDexFund, data, big0, big0)
}

//part of NewOrder
//part of NewAgentOrder
func BenchmarkVMDexTradeNewOrderSend(b *testing.B) {
	sendBlock := makeDexTradeNewOrderBlock(types.AddressDexFund)
	benchmarkSend(b, sendBlock)
}

//part of NewOrder
//part of NewAgentOrder
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
	order.TakerOperatorFeeRate = 100
	order.MakerOperatorFeeRate = 100
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

//part of CancelOrder[entry]
func BenchmarkVMDexTradeCancelOrderSend(b *testing.B) {
	sendBlock := makeDexTradeCancelOrderBlock(testAddr)
	benchmarkSend(b, sendBlock)
}

//part of CancelOrder
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

//part of NewMarket
func BenchmarkVMDexTradeNotifyNewMarketSend(b *testing.B) {
	sendBlock := makeDexTradeNotifyNewMarketBlock(types.AddressDexFund)
	benchmarkSend(b, sendBlock)
}

//part of NewMarket
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
	marketInfo.TakerOperatorFeeRate = 18
	marketInfo.MakerOperatorFeeRate = 18
	marketInfo.AllowMining = true
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
	idsData = append(idsData, orderId...)
	idsData = append(idsData, orderId...)
	idsData = append(idsData, orderId...)
	idsData = append(idsData, orderId...)
	idsData = append(idsData, orderId...)
	if syncData, err := abi.ABIDexTrade.PackMethod(abi.MethodNameDexTradeCleanExpireOrders, idsData); err != nil {
		panic(err)
	} else {
		return makeSendBlock(addr, types.AddressDexTrade, syncData, big0, big0)
	}
}

//part of CancelOrderByHash[entry]
func BenchmarkVMDexTradeCancelOrderByHashSend(b *testing.B) {
	sendBlock := makeDexTradeCancelOrderByHashBlock(testAddr)
	benchmarkSend(b, sendBlock)
}

//part of CancelOrderByHash
func BenchmarkVMDexTradeCancelOrderByHashReceive(b *testing.B) {
	sendBlock := makeDexTradeCancelOrderByHashBlock(testAddr)
	receiveBlock := makeReceiveBlock(types.AddressDexTrade)
	benchmarkReceive(b, sendBlock, receiveBlock)
}
func makeDexTradeCancelOrderByHashBlock(addr types.Address) *ledger.AccountBlock {
	sendHash, _ := hex.DecodeString("ba5520be6bbc1b8a77ab83af14f2a14c86a6abbcf380d314b4c9d8e440b5ff3b")
	data, err := abi.ABIDexTrade.PackMethod(abi.MethodNameDexTradeCancelOrderByHash, sendHash)
	if err != nil {
		panic(err)
	}
	return makeSendBlock(addr, types.AddressDexTrade, data, big0, big0)
}

func BenchmarkVMDexFundLockVxForDividendSend(b *testing.B) {
	sendBlock := makeDexFundLockVxForDividendBlock(testAddr)
	benchmarkSend(b, sendBlock)
}
func BenchmarkVMDexFundLockVxForDividendReceive(b *testing.B) {
	sendBlock := makeDexFundLockVxForDividendBlock(testAddr)
	receiveBlock := makeReceiveBlock(types.AddressDexFund)
	benchmarkReceive(b, sendBlock, receiveBlock)
}
func makeDexFundLockVxForDividendBlock(addr types.Address) *ledger.AccountBlock {
	data, err := abi.ABIDexFund.PackMethod(abi.MethodNameDexFundLockVxForDividend, uint8(dex.UnlockVx), new(big.Int).Mul(new(big.Int).SetInt64(1e18), big.NewInt(10)))
	if err != nil {
		panic(err)
	}
	return makeSendBlock(addr, types.AddressDexFund, data, big0, big0)
}

func BenchmarkVMDexFundSwitchConfigSend(b *testing.B) {
	sendBlock := makeDexFundSwitchConfigBlock(testAddr)
	benchmarkSend(b, sendBlock)
}
func BenchmarkVMDexFundSwitchConfigReceive(b *testing.B) {
	sendBlock := makeDexFundSwitchConfigBlock(testAddr)
	receiveBlock := makeReceiveBlock(types.AddressDexFund)
	benchmarkReceive(b, sendBlock, receiveBlock)
}
func makeDexFundSwitchConfigBlock(addr types.Address) *ledger.AccountBlock {
	data, err := abi.ABIDexFund.PackMethod(abi.MethodNameDexFundSwitchConfig, uint8(dex.AutoLockMinedVx), true)
	if err != nil {
		panic(err)
	}
	return makeSendBlock(addr, types.AddressDexFund, data, big0, big0)
}

func BenchmarkVMDexFundStakeForPrincipalSVIPSend(b *testing.B) {
	sendBlock := makeDexFundStakeForPrincipalSVIPBlock(dexAddr)
	benchmarkSend(b, sendBlock)
}
func BenchmarkVMDexFundStakeForPrincipalSVIPReceive(b *testing.B) {
	sendBlock := makeDexFundStakeForPrincipalSVIPBlock(dexAddr)
	receiveBlock := makeReceiveBlock(types.AddressDexFund)
	benchmarkReceive(b, sendBlock, receiveBlock)
}
func makeDexFundStakeForPrincipalSVIPBlock(addr types.Address) *ledger.AccountBlock {
	data, err := abi.ABIDexFund.PackMethod(abi.MethodNameDexFundStakeForPrincipalSVIP, addr)
	if err != nil {
		panic(err)
	}
	return makeSendBlock(testAddr, types.AddressDexFund, data, big0, big0)
}

func BenchmarkVMDexFundCancelStakeByIdSend(b *testing.B) {
	sendBlock := makeDexFundCancelStakeByIdBlock()
	benchmarkSend(b, sendBlock)
}
func BenchmarkVMDexFundCancelStakeByIdReceive(b *testing.B) {
	sendBlock := makeDexFundCancelStakeByIdBlock()
	receiveBlock := makeReceiveBlock(types.AddressDexFund)
	benchmarkReceive(b, sendBlock, receiveBlock)
}
func makeDexFundCancelStakeByIdBlock() *ledger.AccountBlock {
	stakeId, _ := types.HexToHash("8aadbeb14a503c43506c9c3566fb3baa7b63f39206bc44260696ecb13ffe6a95")
	data, err := abi.ABIDexFund.PackMethod(abi.MethodNameDexFundCancelStakeById, stakeId)
	if err != nil {
		panic(err)
	}
	return makeSendBlock(testAddr, types.AddressDexFund, data, big0, big0)
}

func BenchmarkVMDexFundDelegateStakeCallbackV2Send(b *testing.B) {
	sendBlock := makeDexFundDelegateStakeCallbackV2Block()
	benchmarkSend(b, sendBlock)
}
func BenchmarkVMDexFundDelegateStakeCallbackV2Receive(b *testing.B) {
	sendBlock := makeDexFundDelegateStakeCallbackV2Block()
	receiveBlock := makeReceiveBlock(types.AddressDexFund)
	benchmarkReceive(b, sendBlock, receiveBlock)
}
func makeDexFundDelegateStakeCallbackV2Block() *ledger.AccountBlock {
	stakeId, _ := types.HexToHash("8aadbeb14a503c43506c9c3566fb3baa7b63f39206bc44260696ecb13ffe6a95")
	data, err := abi.ABIDexFund.PackMethod(abi.MethodNameDexFundDelegateStakeCallbackV2, stakeId, true)
	if err != nil {
		panic(err)
	}
	return makeSendBlock(testAddr, types.AddressDexFund, data, big0, big0)
}

func BenchmarkVMDexFundCancelDelegateStakeCallbackV2Send(b *testing.B) {
	sendBlock := makeDexFundCancelDelegateStakeCallbackV2Block()
	benchmarkSend(b, sendBlock)
}
func BenchmarkVMDexFundCancelDelegateStakeCallbackV2Receive(b *testing.B) {
	sendBlock := makeDexFundCancelDelegateStakeCallbackV2Block()
	receiveBlock := makeReceiveBlock(types.AddressDexFund)
	benchmarkReceive(b, sendBlock, receiveBlock)
}
func makeDexFundCancelDelegateStakeCallbackV2Block() *ledger.AccountBlock {
	stakeId, _ := types.HexToHash("8aadbeb14a503c43506c9c3566fb3baa7b63f39206bc44260696ecb13ffe6a95")
	data, err := abi.ABIDexFund.PackMethod(abi.MethodNameDexFundCancelDelegateStakeCallbackV2, stakeId, true)
	if err != nil {
		panic(err)
	}
	return makeSendBlock(testAddr, types.AddressDexFund, data, new(big.Int).Mul(big.NewInt(1e6), big1e18), big0)
}

func TestPrintDexBlockSize(t *testing.T) {
	printBlockSize("dexFundDeposit",
		makeDexDepositBlock(testAddr),
		makeReceiveBlock(types.AddressDexFund))
	printBlockSize("dexFundWithdraw",
		makeDexWithdrawBlock(testAddr),
		makeReceiveBlock(types.AddressDexFund))
	printBlockSize("dexFundNewMarket",
		makeDexNewMarketBlock(testAddr),
		makeReceiveBlock(types.AddressDexFund))
	printBlockSize("dexFundNewOrder",
		makeDexNewOrderBlock(testAddr),
		makeReceiveBlock(types.AddressDexFund))
	printBlockSize("dexFundSettleOrders",
		makeDexSettleOrdersBlock(testAddr),
		makeReceiveBlock(types.AddressDexFund))
	printBlockSize("dexFundPeriodJob",
		makeDexPeriodJobBlock(dexAddr),
		makeReceiveBlock(types.AddressDexFund))
	printBlockSize("dexFundStakeForMining",
		makeDexStakeForMiningBlock(testAddr),
		makeReceiveBlock(types.AddressDexFund))
	printBlockSize("dexFundPledgeForVip",
		makeDexPledgeForVipBlock(dexAddr),
		makeReceiveBlock(types.AddressDexFund))
	printBlockSize("dexFundPledgeCallback",
		makeDexPledgeCallbackBlock(testAddr),
		makeReceiveBlock(types.AddressDexFund))
	printBlockSize("dexFundCancelPledgeCallBack",
		makeDexCancelPledgeCallbackBlock(testAddr),
		makeReceiveBlock(types.AddressDexFund))
	printBlockSize("dexFundGetTokenInfoCallback",
		makeDexGetTokenInfoCallbackBlock(types.AddressAsset),
		makeReceiveBlock(types.AddressDexFund))
	printBlockSize("dexFundOwnerConfig",
		makeDexOwnerConfigBlock(dexAddr),
		makeReceiveBlock(types.AddressDexFund))
	printBlockSize("dexFundOwnerConfigTrade",
		makeDexOwnerConfigTradeBlock(dexAddr),
		makeReceiveBlock(types.AddressDexFund))
	printBlockSize("dexFundMarketOwnerConfig",
		makeDexMarketOwnerConfigBlock(testAddr),
		makeReceiveBlock(types.AddressDexFund))
	printBlockSize("dexFundTransferTokenOwner",
		makeDexTransferTokenOwnerBlock(testAddr),
		makeReceiveBlock(types.AddressDexFund))
	printBlockSize("dexFundNotifyTime",
		makeDexNotifyTimeBlock(dexAddr),
		makeReceiveBlock(types.AddressDexFund))
	printBlockSize("dexFundNewInviter",
		makeDexNewInviterBlock(dexAddr),
		makeReceiveBlock(types.AddressDexFund))
	printBlockSize("dexFundBindInviteCode",
		makeDexBindInviteCodeBlock(testAddr),
		makeReceiveBlock(types.AddressDexFund))
	printBlockSize("dexFundEndorseVx",
		makeDexEndorseVxBlock(testAddr),
		makeReceiveBlock(types.AddressDexFund))
	printBlockSize("dexFundSettleMakerMinedVx",
		makeDexSettleMakerMinedVxBlock(dexAddr),
		makeReceiveBlock(types.AddressDexFund))
	printBlockSize("dexFundPledgeForSuperVip",
		makeDexPledgeForSuperVipBlock(dexAddr),
		makeReceiveBlock(types.AddressDexFund))
	printBlockSize("dexFundConfigMarketAgent",
		makeDexConfigMarketAgentBlock(dexAddr),
		makeReceiveBlock(types.AddressDexFund))
	printBlockSize("dexFundNewAgentOrder",
		makeDexNewAgentOrderBlock(dexAddr),
		makeReceiveBlock(types.AddressDexFund))

	printBlockSize("dexTradeNewOrder",
		makeDexTradeNewOrderBlock(testAddr),
		makeReceiveBlock(types.AddressDexTrade))
	printBlockSize("dexTradeCancelOrder",
		makeDexTradeCancelOrderBlock(testAddr),
		makeReceiveBlock(types.AddressDexTrade))
	printBlockSize("dexTradeNotifyNewMarket",
		makeDexTradeNotifyNewMarketBlock(dexAddr),
		makeReceiveBlock(types.AddressDexTrade))
	printBlockSize("dexTradeCleanExpireOrders",
		makeDexTradeCleanExpireOrdersBlock(dexAddr),
		makeReceiveBlock(types.AddressDexTrade))
	printBlockSize("dexTradeCancelOrderByHash",
		makeDexTradeCancelOrderByHashBlock(dexAddr),
		makeReceiveBlock(types.AddressDexTrade))

	printBlockSize("dexFundLockVxForDividend",
		makeDexFundLockVxForDividendBlock(testAddr),
		makeReceiveBlock(types.AddressDexFund))
	printBlockSize("dexFundSwitchConfig",
		makeDexFundSwitchConfigBlock(testAddr),
		makeReceiveBlock(types.AddressDexFund))
	printBlockSize("dexFundStakeForPrincipalSVIP",
		makeDexFundStakeForPrincipalSVIPBlock(testAddr),
		makeReceiveBlock(types.AddressDexFund))
	printBlockSize("dexFundCancelStakeById",
		makeDexFundCancelStakeByIdBlock(),
		makeReceiveBlock(types.AddressDexFund))
	printBlockSize("dexFundDelegateStakeCallbackV2",
		makeDexFundDelegateStakeCallbackV2Block(),
		makeReceiveBlock(types.AddressDexFund))
	printBlockSize("dexFundCancelDelegateStakeCallbackV2",
		makeDexFundCancelDelegateStakeCallbackV2Block(),
		makeReceiveBlock(types.AddressDexFund))
}
