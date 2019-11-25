package chain_genesis

import (
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"github.com/vitelabs/go-vite/vm/contracts/abi"
	"github.com/vitelabs/go-vite/vm/contracts/dex"
	dexproto "github.com/vitelabs/go-vite/vm/contracts/dex/proto"
	"github.com/vitelabs/go-vite/vm/util"
	"math/big"
	"sort"
	"strconv"

	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/config"
	"github.com/vitelabs/go-vite/ledger"
	"github.com/vitelabs/go-vite/vm_db"
)

func NewGenesisAccountBlocks(cfg *config.Genesis) []*vm_db.VmAccountBlock {
	list := make([]*vm_db.VmAccountBlock, 0)
	addrSet := make(map[types.Address]interface{})
	list, addrSet = newGenesisGovernanceContractBlocks(cfg, list, addrSet)
	list, addrSet = newGenesisAssetContractBlocks(cfg, list, addrSet)
	list, addrSet = newGenesisQuotaContractBlocks(cfg, list, addrSet)
	list = newGenesisNormalAccountBlocks(cfg, list, addrSet)
	list, addrSet = newDexFundContractBlocks(cfg, list, addrSet)
	list, addrSet = newDexTradeContractBlocks(cfg, list, addrSet)
	return list
}

func updateAccountBalanceMap(cfg *config.Genesis, addr types.Address, vmdb vm_db.VmDb) {
	if len(cfg.AccountBalanceMap) == 0 {
		return
	}
	for tokenIdStr, balance := range cfg.AccountBalanceMap[addr.String()] {
		tokenId, err := types.HexToTokenTypeId(tokenIdStr)
		dealWithError(err)
		vmdb.SetBalance(&tokenId, balance)
	}
}

func newGenesisGovernanceContractBlocks(cfg *config.Genesis, list []*vm_db.VmAccountBlock, addrSet map[types.Address]interface{}) ([]*vm_db.VmAccountBlock, map[types.Address]interface{}) {
	if cfg.GovernanceInfo != nil {
		contractAddr := types.AddressGovernance
		block := ledger.AccountBlock{
			BlockType:      ledger.BlockTypeGenesisReceive,
			Height:         1,
			AccountAddress: contractAddr,
			Amount:         big.NewInt(0),
			Fee:            big.NewInt(0),
		}
		vmdb := vm_db.NewGenesisVmDB(&contractAddr)
		for gidStr, groupInfo := range cfg.GovernanceInfo.ConsensusGroupInfoMap {
			gid, err := types.HexToGid(gidStr)
			dealWithError(err)
			var registerConditionParam []byte
			if groupInfo.RegisterConditionId == 1 {
				registerConditionParam, err = abi.ABIGovernance.PackVariable(abi.VariableNameRegisterStakeParam,
					groupInfo.RegisterConditionParam.StakeAmount,
					groupInfo.RegisterConditionParam.StakeToken,
					groupInfo.RegisterConditionParam.StakeHeight)
				dealWithError(err)
			}
			value, err := abi.ABIGovernance.PackVariable(abi.VariableNameConsensusGroupInfo,
				groupInfo.NodeCount,
				groupInfo.Interval,
				groupInfo.PerCount,
				groupInfo.RandCount,
				groupInfo.RandRank,
				groupInfo.Repeat,
				groupInfo.CheckLevel,
				groupInfo.CountingTokenId,
				groupInfo.RegisterConditionId,
				registerConditionParam,
				groupInfo.VoteConditionId,
				[]byte{},
				groupInfo.Owner,
				groupInfo.StakeAmount,
				groupInfo.ExpirationHeight)
			dealWithError(err)
			util.SetValue(vmdb, abi.GetConsensusGroupInfoKey(gid), value)
		}

		for gidStr, groupRegistrationInfoMap := range cfg.GovernanceInfo.RegistrationInfoMap {
			gid, err := types.HexToGid(gidStr)
			dealWithError(err)
			for name, registrationInfo := range groupRegistrationInfoMap {
				if len(registrationInfo.HistoryAddressList) == 0 {
					registrationInfo.HistoryAddressList = []types.Address{*registrationInfo.BlockProducingAddress}
				}
				value, err := abi.ABIGovernance.PackVariable(abi.VariableNameRegistrationInfo,
					name,
					registrationInfo.BlockProducingAddress,
					registrationInfo.StakeAddress,
					registrationInfo.Amount,
					registrationInfo.ExpirationHeight,
					registrationInfo.RewardTime,
					registrationInfo.RevokeTime,
					registrationInfo.HistoryAddressList)
				dealWithError(err)
				util.SetValue(vmdb, abi.GetRegistrationInfoKey(name, gid), value)
				if len(cfg.GovernanceInfo.HisNameMap) == 0 ||
					len(cfg.GovernanceInfo.HisNameMap[gidStr]) == 0 ||
					len(cfg.GovernanceInfo.HisNameMap[gidStr][registrationInfo.BlockProducingAddress.String()]) == 0 {
					value, err := abi.ABIGovernance.PackVariable(abi.VariableNameRegisteredHisName, name)
					dealWithError(err)
					util.SetValue(vmdb, abi.GetHisNameKey(*registrationInfo.BlockProducingAddress, gid), value)
				}
			}
		}

		for gidStr, groupHisNameMap := range cfg.GovernanceInfo.HisNameMap {
			gid, err := types.HexToGid(gidStr)
			dealWithError(err)
			for blockProducingAddrStr, name := range groupHisNameMap {
				blockProducingAddr, err := types.HexToAddress(blockProducingAddrStr)
				dealWithError(err)
				value, err := abi.ABIGovernance.PackVariable(abi.VariableNameRegisteredHisName, name)
				dealWithError(err)
				util.SetValue(vmdb, abi.GetHisNameKey(blockProducingAddr, gid), value)
			}
		}

		for gidStr, groupVoteMap := range cfg.GovernanceInfo.VoteStatusMap {
			gid, err := types.HexToGid(gidStr)
			dealWithError(err)
			for voteAddrStr, sbpName := range groupVoteMap {
				voteAddr, err := types.HexToAddress(voteAddrStr)
				dealWithError(err)
				value, err := abi.ABIGovernance.PackVariable(abi.VariableNameVoteInfo, sbpName)
				dealWithError(err)
				util.SetValue(vmdb, abi.GetVoteInfoKey(voteAddr, gid), value)
			}
		}

		updateAccountBalanceMap(cfg, contractAddr, vmdb)

		block.Hash = block.ComputeHash()
		list = append(list, &vm_db.VmAccountBlock{&block, vmdb})
		addrSet[contractAddr] = struct{}{}
	}
	return list, addrSet
}

type tokenInfoForSort struct {
	tokenId types.TokenTypeId
	config.TokenInfo
}
type byTokenId []*tokenInfoForSort

func (a byTokenId) Len() int      { return len(a) }
func (a byTokenId) Swap(i, j int) { a[i], a[j] = a[j], a[i] }
func (a byTokenId) Less(i, j int) bool {
	return a[i].tokenId.Hex() > a[j].tokenId.Hex()
}

func newGenesisAssetContractBlocks(cfg *config.Genesis, list []*vm_db.VmAccountBlock, addrSet map[types.Address]interface{}) ([]*vm_db.VmAccountBlock, map[types.Address]interface{}) {
	if cfg.AssetInfo != nil {
		nextIndexMap := make(map[string]uint16)
		contractAddr := types.AddressAsset
		block := ledger.AccountBlock{
			BlockType:      ledger.BlockTypeGenesisReceive,
			Height:         1,
			AccountAddress: contractAddr,
			Amount:         big.NewInt(0),
			Fee:            big.NewInt(0),
		}
		vmdb := vm_db.NewGenesisVmDB(&contractAddr)
		tokenList := make([]*tokenInfoForSort, 0, len(cfg.AssetInfo.TokenInfoMap))
		for tokenIdStr, tokenInfo := range cfg.AssetInfo.TokenInfoMap {
			tokenId, err := types.HexToTokenTypeId(tokenIdStr)
			dealWithError(err)
			tokenList = append(tokenList, &tokenInfoForSort{tokenId, *tokenInfo})
		}
		sort.Sort(byTokenId(tokenList))
		for _, tokenInfo := range tokenList {
			nextIndex := uint16(0)
			if index, ok := nextIndexMap[tokenInfo.TokenSymbol]; ok {
				nextIndex = index
			}
			value, err := abi.ABIAsset.PackVariable(abi.VariableNameTokenInfo,
				tokenInfo.TokenName,
				tokenInfo.TokenSymbol,
				tokenInfo.TotalSupply,
				tokenInfo.Decimals,
				tokenInfo.Owner,
				tokenInfo.IsReIssuable,
				tokenInfo.MaxSupply,
				tokenInfo.IsOwnerBurnOnly,
				nextIndex)
			dealWithError(err)
			nextIndex = nextIndex + 1
			nextIndexMap[tokenInfo.TokenSymbol] = nextIndex
			nextIndexValue, err := abi.ABIAsset.PackVariable(abi.VariableNameTokenIndex, nextIndex)
			dealWithError(err)
			util.SetValue(vmdb, abi.GetNextTokenIndexKey(tokenInfo.TokenSymbol), nextIndexValue)
			util.SetValue(vmdb, abi.GetTokenInfoKey(tokenInfo.tokenId), value)
		}

		if len(cfg.AssetInfo.LogList) > 0 {
			for _, log := range cfg.AssetInfo.LogList {
				dataBytes, err := hex.DecodeString(log.Data)
				dealWithError(err)
				vmdb.AddLog(&ledger.VmLog{Data: dataBytes, Topics: log.Topics})
			}
		}
		block.LogHash = vmdb.GetLogListHash()
		updateAccountBalanceMap(cfg, contractAddr, vmdb)
		block.Hash = block.ComputeHash()
		list = append(list, &vm_db.VmAccountBlock{&block, vmdb})
		addrSet[contractAddr] = struct{}{}
	}
	return list, addrSet
}

func newGenesisQuotaContractBlocks(cfg *config.Genesis, list []*vm_db.VmAccountBlock, addrSet map[types.Address]interface{}) ([]*vm_db.VmAccountBlock, map[types.Address]interface{}) {
	if cfg.QuotaInfo != nil {
		contractAddr := types.AddressQuota
		block := ledger.AccountBlock{
			BlockType:      ledger.BlockTypeGenesisReceive,
			Height:         1,
			AccountAddress: contractAddr,
			Amount:         big.NewInt(0),
			Fee:            big.NewInt(0),
		}
		vmdb := vm_db.NewGenesisVmDB(&contractAddr)
		for stakeAddrStr, stakeInfoList := range cfg.QuotaInfo.StakeInfoMap {
			stakeAddr, err := types.HexToAddress(stakeAddrStr)
			dealWithError(err)
			for i, stakeInfo := range stakeInfoList {
				value, err := abi.ABIQuota.PackVariable(abi.VariableNameStakeInfo,
					stakeInfo.Amount,
					stakeInfo.ExpirationHeight,
					stakeInfo.Beneficiary,
					false,
					types.ZERO_ADDRESS,
					uint8(0))
				dealWithError(err)
				util.SetValue(vmdb, abi.GetStakeInfoKey(stakeAddr, uint64(i)), value)
			}
		}

		for beneficiaryStr, amount := range cfg.QuotaInfo.StakeBeneficialMap {
			beneficiary, err := types.HexToAddress(beneficiaryStr)
			dealWithError(err)
			value, err := abi.ABIQuota.PackVariable(abi.VariableNameStakeBeneficial, amount)
			dealWithError(err)
			util.SetValue(vmdb, abi.GetStakeBeneficialKey(beneficiary), value)
		}
		updateAccountBalanceMap(cfg, contractAddr, vmdb)
		block.Hash = block.ComputeHash()
		list = append(list, &vm_db.VmAccountBlock{&block, vmdb})
		addrSet[contractAddr] = struct{}{}
	}
	return list, addrSet
}

func newGenesisNormalAccountBlocks(cfg *config.Genesis, list []*vm_db.VmAccountBlock, addrSet map[types.Address]interface{}) []*vm_db.VmAccountBlock {
	for addrStr, balanceMap := range cfg.AccountBalanceMap {
		addr, err := types.HexToAddress(addrStr)
		dealWithError(err)
		if _, ok := addrSet[addr]; ok {
			continue
		}
		block := ledger.AccountBlock{
			BlockType:      ledger.BlockTypeGenesisReceive,
			Height:         1,
			AccountAddress: addr,
			Amount:         big.NewInt(0),
			Fee:            big.NewInt(0),
		}
		vmdb := vm_db.NewGenesisVmDB(&addr)
		for tokenIdStr, balance := range balanceMap {
			tokenId, err := types.HexToTokenTypeId(tokenIdStr)
			dealWithError(err)
			vmdb.SetBalance(&tokenId, balance)
		}
		block.Hash = block.ComputeHash()
		list = append(list, &vm_db.VmAccountBlock{&block, vmdb})
	}

	return list
}

func newDexFundContractBlocks(cfg *config.Genesis, list []*vm_db.VmAccountBlock, addrSet map[types.Address]interface{}) ([]*vm_db.VmAccountBlock, map[types.Address]interface{}) {
	if cfg.DexFundInfo != nil {
		contractAddr := types.AddressDexFund
		block := ledger.AccountBlock{
			BlockType:      ledger.BlockTypeGenesisReceive,
			Height:         1,
			AccountAddress: contractAddr,
			Amount:         big.NewInt(0),
			Fee:            big.NewInt(0),
		}
		vmdb := vm_db.NewGenesisVmDB(&contractAddr)
		dex.SetOwner(vmdb, cfg.DexFundInfo.Owner)
		dex.SetTimeOracle(vmdb, cfg.DexFundInfo.Timer)
		dex.SetPeriodJobTrigger(vmdb, cfg.DexFundInfo.Trigger)
		dex.SaveMaintainer(vmdb, cfg.DexFundInfo.Maintainer)
		dex.SaveMakerMiningAdmin(vmdb, cfg.DexFundInfo.MakerMineProxy)
		dex.GenesisSetTimestamp(vmdb, cfg.DexFundInfo.NotifiedTimestamp)
		dex.SaveVxMinePool(vmdb, cfg.DexFundInfo.EndorseVxAmount)
		for tk, amt := range cfg.DexFundInfo.AccountBalanceMap {
			vmdb.SetBalance(&tk, amt)
		}
		for _, tk := range cfg.DexFundInfo.Tokens {
			tokenInfo := &dex.TokenInfo{}
			tokenInfo.TokenId = tk.TokenId.Bytes()
			tokenInfo.Decimals = tk.Decimals
			tokenInfo.Symbol = tk.Symbol
			tokenInfo.Index = tk.Index
			tokenInfo.Owner = tk.Owner.Bytes()
			tokenInfo.QuoteTokenType = tk.QuoteTokenType
			tokenInfo.TokenId = tk.TokenId.Bytes()
			dex.SaveTokenInfo(vmdb, tk.TokenId, tokenInfo)
		}
		pendings := &dex.PendingTransferTokenOwnerActions{}
		for _, ttk := range cfg.DexFundInfo.PendingTransferTokens {
			action := &dexproto.TransferTokenOwnerAction{}
			action.Token = ttk.TokenId.Bytes()
			action.Origin = ttk.Origin.Bytes()
			action.New = ttk.New.Bytes()
			pendings.PendingActions = append(pendings.PendingActions, action)
		}
		if len(cfg.DexFundInfo.Markets) > 0 {
			for _, mkif := range cfg.DexFundInfo.Markets {
				mkInfo := &dex.MarketInfo{}
				mkInfo.MarketId = mkif.MarketId
				mkInfo.MarketSymbol = mkif.MarketSymbol
				mkInfo.TradeToken = mkif.TradeToken.Bytes()
				mkInfo.QuoteToken = mkif.QuoteToken.Bytes()
				mkInfo.QuoteTokenType = mkif.QuoteTokenType
				mkInfo.TradeTokenDecimals = mkif.TradeTokenDecimals
				mkInfo.QuoteTokenDecimals = mkif.QuoteTokenDecimals
				mkInfo.TakerOperatorFeeRate = mkif.TakerBrokerFeeRate
				mkInfo.MakerOperatorFeeRate = mkif.MakerBrokerFeeRate
				mkInfo.AllowMining = mkif.AllowMine
				mkInfo.Valid = mkif.Valid
				mkInfo.Owner = mkif.Owner.Bytes()
				mkInfo.Creator = mkif.Creator.Bytes()
				mkInfo.Stopped = mkif.Stopped
				mkInfo.Timestamp = mkif.Timestamp
				dex.SaveMarketInfo(vmdb, mkInfo, mkif.TradeToken, mkif.QuoteToken)
			}
		}
		dex.SavePendingTransferTokenOwners(vmdb, pendings)
		for _, fund := range cfg.DexFundInfo.UserFunds {
			userFund := &dex.Fund{}
			userFund.Address = fund.Address.Bytes()
			for _, acc := range fund.Accounts {
				userAcc := &dexproto.Account{}
				userAcc.Token = acc.Token.Bytes()
				userAcc.Available = acc.Available.Bytes()
				userAcc.Locked = acc.Locked.Bytes()
				if acc.VxLocked != nil {
					userAcc.VxLocked = acc.VxLocked.Bytes()
				}
				if acc.VxUnlocking != nil {
					userAcc.VxUnlocking = acc.VxUnlocking.Bytes()
				}
				userFund.Accounts = append(userFund.Accounts, userAcc)
			}
			dex.SaveFund(vmdb, fund.Address, userFund)
		}
		for addr, amount := range cfg.DexFundInfo.PledgeVxs {
			dex.SaveMiningStakedAmount(vmdb, addr, amount)
		}
		for _, addr := range cfg.DexFundInfo.PledgeVips {
			pledgeInfo := &dex.VIPStaking{}
			pledgeInfo.StakedTimes = 1
			pledgeInfo.Timestamp = cfg.DexFundInfo.NotifiedTimestamp
			dex.SaveVIPStaking(vmdb, addr, pledgeInfo)
		}
		for pid, amt := range cfg.DexFundInfo.MakerMinedVxs {
			period, _ := strconv.Atoi(pid)
			dex.SaveMakerMiningPoolByPeriodId(vmdb, uint64(period), amt)
		}
		for addr, code := range cfg.DexFundInfo.Inviters {
			dex.SaveInviterByCode(vmdb, addr, code)
			dex.SaveCodeByInviter(vmdb, addr, code)
		}
		for principal, agent := range cfg.DexFundInfo.MarketAgents {
			vmdb.SetValue(dex.GetGrantedMarketToAgentKey(principal, 1), agent.Bytes())
		}
		for _, stake := range cfg.DexFundInfo.DexStakes {
			info := &dex.DelegateStakeInfo{}
			info.StakeType = stake.StakeType
			info.Address = stake.Address.Bytes()
			if stake.Principal != types.ZERO_ADDRESS {
				info.Principal = stake.Principal.Bytes()
			}
			info.Amount = stake.Amount.Bytes()
			if info.StakeType == dex.StakeForMining {
				dex.SaveMiningStakedV2Amount(vmdb, stake.Address, stake.Amount)
			}
			info.Status = stake.Status
			info.SerialNo = stake.SerialNo
			infoBytes, _ := info.Serialize()
			vmdb.SetValue(dex.GetDelegateStakeInfoKey(stake.Id.Bytes()), infoBytes)
			index := &dex.DelegateStakeAddressIndex{}
			index.StakeType = stake.StakeType
			index.Id = stake.Id.Bytes()
			indexBytes, _ := index.Serialize()
			vmdb.SetValue(dex.GetDelegateStakeAddressIndexKey(stake.Id.Bytes(), stake.SerialNo), indexBytes)
		}
		block.Hash = block.ComputeHash()
		fmt.Printf("fund block.hash %s\n", block.Hash.String())
		list = append(list, &vm_db.VmAccountBlock{&block, vmdb})
		addrSet[contractAddr] = struct{}{}
	}
	return list, addrSet
}

func newDexTradeContractBlocks(cfg *config.Genesis, list []*vm_db.VmAccountBlock, addrSet map[types.Address]interface{}) ([]*vm_db.VmAccountBlock, map[types.Address]interface{}) {
	if cfg.DexTradeInfo != nil {
		contractAddr := types.AddressDexTrade
		block := ledger.AccountBlock{
			BlockType:      ledger.BlockTypeGenesisReceive,
			Height:         1,
			AccountAddress: contractAddr,
			Amount:         big.NewInt(0),
			Fee:            big.NewInt(0),
		}
		vmdb := vm_db.NewGenesisVmDB(&contractAddr)
		dex.SetTradeTimestamp(vmdb, cfg.DexTradeInfo.Timestamp)
		for _, mkif := range cfg.DexTradeInfo.Markets {
			mkInfo := &dex.MarketInfo{}
			mkInfo.MarketId = mkif.MarketId
			mkInfo.MarketSymbol = mkif.MarketSymbol
			mkInfo.TradeToken = mkif.TradeToken.Bytes()
			mkInfo.QuoteToken = mkif.QuoteToken.Bytes()
			mkInfo.QuoteTokenType = mkif.QuoteTokenType
			mkInfo.TradeTokenDecimals = mkif.TradeTokenDecimals
			mkInfo.QuoteTokenDecimals = mkif.QuoteTokenDecimals
			mkInfo.TakerOperatorFeeRate = mkif.TakerBrokerFeeRate
			mkInfo.MakerOperatorFeeRate = mkif.MakerBrokerFeeRate
			mkInfo.AllowMining = mkif.AllowMine
			mkInfo.Valid = mkif.Valid
			mkInfo.Owner = mkif.Owner.Bytes()
			mkInfo.Creator = mkif.Creator.Bytes()
			mkInfo.Stopped = mkif.Stopped
			mkInfo.Timestamp = mkif.Timestamp
			dex.SaveMarketInfoById(vmdb, mkInfo)
		}
		for _, od := range cfg.DexTradeInfo.Orders {
			order := &dex.Order{}
			order.Id, _ = base64.StdEncoding.DecodeString(od.Id)
			order.Address = od.Address.Bytes()
			order.MarketId = od.MarketId
			order.Side = od.Side
			order.Type = od.Type
			order.Price = dex.PriceToBytes(od.Price)
			order.TakerFeeRate = od.TakerFeeRate
			order.MakerFeeRate = od.MakerFeeRate
			order.TakerOperatorFeeRate = od.TakerBrokerFeeRate
			order.MakerOperatorFeeRate = od.MakerBrokerFeeRate
			order.Quantity = od.Quantity.Bytes()
			order.Amount = od.Amount.Bytes()
			order.LockedBuyFee = od.LockedBuyFee.Bytes()
			order.Status = od.Status
			order.ExecutedQuantity = od.ExecutedQuantity.Bytes()
			order.ExecutedAmount = od.ExecutedAmount.Bytes()
			order.ExecutedBaseFee = od.ExecutedBaseFee.Bytes()
			order.ExecutedOperatorFee = od.ExecutedBrokerFee.Bytes()
			order.Timestamp = cfg.DexTradeInfo.Timestamp
			order.Agent = od.Agent.Bytes()
			order.SendHash = od.SendHash.Bytes()
			orderId := order.Id
			if data, err := order.SerializeCompact(); err != nil {
				panic(err)
			} else {
				vmdb.SetValue(orderId, data)
			}
			dex.SaveHashMapOrderId(vmdb, order.SendHash, orderId)
		}
		block.Hash = block.ComputeHash()
		fmt.Printf("trade block.hash %s\n", block.Hash.String())
		list = append(list, &vm_db.VmAccountBlock{&block, vmdb})
		addrSet[contractAddr] = struct{}{}
	}
	return list, addrSet
}

func dealWithError(err error) {
	if err != nil {
		panic(err)
	}
}
