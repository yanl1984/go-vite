package nodemanager

import (
	"bytes"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"github.com/vitelabs/go-vite/chain"
	"github.com/vitelabs/go-vite/common/helper"
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/ledger"
	"github.com/vitelabs/go-vite/trie"
	"github.com/vitelabs/go-vite/vm/abi"
	cabi "github.com/vitelabs/go-vite/vm/contracts/abi"
	"github.com/vitelabs/go-vite/vm/util"
	"math/big"
	"strconv"
	"strings"
)

var (
	STORAGE_KEY_BALANCE = []byte("$balance")
	STORAGE_KEY_CODE    = []byte("$code")
	emptyBalance        = big.NewInt(0)
)

func isBalanceOrCode(key []byte) bool {
	return bytes.HasPrefix(key, STORAGE_KEY_CODE) || bytes.HasPrefix(key, STORAGE_KEY_BALANCE)
}

var registerNameMap = make(map[string]string)

func exportContractBalanceAndStorage(m map[types.Address]*big.Int, g *Genesis, addr types.Address, balance *big.Int, trie *trie.Trie, c chain.Chain) (map[types.Address]*big.Int, *Genesis, error) {
	if addr == types.AddressRegister {
		m, g = exportRegisterBalanceAndStorage(m, g, trie)
		return m, g, nil
	} else if addr == types.AddressPledge {
		m, g = exportPledgeBalanceAndStorage(m, g, trie)
		return m, g, nil
	} else if addr == types.AddressMintage {
		m, g = exportMintageBalanceAndStorage(m, g, trie)
		return m, g, nil
	} else if addr == types.AddressVote {
		m, g = exportVoteBalanceAndStorage(m, g, trie)
		return m, g, nil
	} else if addr == types.AddressConsensusGroup {
		m, g = exportConsensusGroupBalanceAndStorage(m, g, trie)
		return m, g, nil
	} else {
		// for other contract, return to creator
		responseBlock, err := c.GetAccountBlockByHeight(&addr, 1)
		if err != nil {
			return m, g, err
		}
		requestBlock, err := c.GetAccountBlockByHash(&responseBlock.FromBlockHash)
		if err != nil {
			return m, g, err
		}
		m = updateBalance(m, requestBlock.AccountAddress, new(big.Int).Add(requestBlock.Fee, balance))
		return m, g, err
	}
}

var (
	registerPledgeAmount       = new(big.Int).Mul(big.NewInt(100000), big.NewInt(1e18))
	registerRefundPledgeAmount = new(big.Int).Mul(big.NewInt(400000), big.NewInt(1e18))
	registerWithdrawHeight     = uint64(1)
	registerRewardTime         = int64(1)
	registerCancelTime         = int64(0)
)

func exportRegisterBalanceAndStorage(m map[types.Address]*big.Int, g *Genesis, trie *trie.Trie) (map[types.Address]*big.Int, *Genesis) {
	if g.ConsensusGroupInfo == nil {
		g.ConsensusGroupInfo = &ConsensusGroupContractInfo{}
	}
	g.ConsensusGroupInfo.RegistrationInfoMap = make(map[string]map[string]RegistrationInfo)
	g.ConsensusGroupInfo.HisNameMap = make(map[string]map[string]string)
	iter := trie.NewIterator(nil)
	for {
		key, value, ok := iter.Next()
		if !ok {
			break
		}
		if isBalanceOrCode(key) || len(value) == 0 {
			continue
		}
		if cabi.IsRegisterKey(key) {
			old := new(types.Registration)
			cabi.ABIRegister.UnpackVariable(old, cabi.VariableNameRegistration, value)
			if !old.IsActive() {
				continue
			}
			registerNameMap[old.Name] = ""
			gid := cabi.GetGidFromRegisterKey(key)
			gidStr := gid.String()
			if _, ok := g.ConsensusGroupInfo.RegistrationInfoMap[gidStr]; !ok {
				g.ConsensusGroupInfo.RegistrationInfoMap[gidStr] = make(map[string]RegistrationInfo)
			}
			g.ConsensusGroupInfo.RegistrationInfoMap[gidStr][old.Name] = RegistrationInfo{
				old.NodeAddr, old.PledgeAddr, registerPledgeAmount, registerWithdrawHeight, registerRewardTime, registerCancelTime, []types.Address{old.NodeAddr},
			}
			if _, ok := g.ConsensusGroupInfo.HisNameMap[gidStr]; !ok {
				g.ConsensusGroupInfo.HisNameMap[gidStr] = make(map[string]string)
			}
			g.ConsensusGroupInfo.HisNameMap[gidStr][old.NodeAddr.String()] = old.Name
			m = updateBalance(m, old.PledgeAddr, registerRefundPledgeAmount)
		}
	}
	return m, g
}

var pledgeWithdrawHeight = uint64(1)

func exportPledgeBalanceAndStorage(m map[types.Address]*big.Int, g *Genesis, trie *trie.Trie) (map[types.Address]*big.Int, *Genesis) {
	g.PledgeInfo = &PledgeContractInfo{}
	g.PledgeInfo.PledgeInfoMap = make(map[string][]PledgeInfo)
	g.PledgeInfo.PledgeBeneficialMap = make(map[string]*big.Int)
	iter := trie.NewIterator(nil)
	for {
		key, value, ok := iter.Next()
		if !ok {
			break
		}
		if isBalanceOrCode(key) || len(value) == 0 {
			continue
		}
		if cabi.IsPledgeKey(key) {
			old := new(cabi.PledgeInfo)
			if err := cabi.ABIPledge.UnpackVariable(old, cabi.VariableNamePledgeInfo, value); err == nil {
				pledgeAddr := cabi.GetPledgeAddrFromPledgeKey(key)
				pledgeAddrStr := pledgeAddr.String()
				if _, ok := g.PledgeInfo.PledgeInfoMap[pledgeAddrStr]; !ok {
					g.PledgeInfo.PledgeInfoMap[pledgeAddrStr] = make([]PledgeInfo, 0)
				}
				g.PledgeInfo.PledgeInfoMap[pledgeAddrStr] = append(g.PledgeInfo.PledgeInfoMap[pledgeAddrStr],
					PledgeInfo{old.Amount, pledgeWithdrawHeight, cabi.GetBeneficialFromPledgeKey(key)})
				m = updateBalance(m, pledgeAddr, emptyBalance)
			}
		} else {
			amount := new(cabi.VariablePledgeBeneficial)
			if err := cabi.ABIPledge.UnpackVariable(amount, cabi.VariableNamePledgeBeneficial, value); err == nil && amount.Amount != nil && amount.Amount.Sign() > 0 {
				g.PledgeInfo.PledgeBeneficialMap[cabi.GetBeneficialFromPledgeBeneficialKey(key).String()] = amount.Amount
			}

		}
	}
	return m, g
}

var mintageWithdrawHeight = uint64(1)

func exportMintageBalanceAndStorage(m map[types.Address]*big.Int, g *Genesis, trie *trie.Trie) (map[types.Address]*big.Int, *Genesis) {
	g.MintageInfo = &MintageContractInfo{}
	g.MintageInfo.TokenInfoMap = make(map[string]TokenInfo)
	g.MintageInfo.LogList = make([]GenesisVmLog, 0)
	iter := trie.NewIterator(nil)
	for {
		key, value, ok := iter.Next()
		if !ok {
			break
		}
		if isBalanceOrCode(key) || len(value) == 0 {
			continue
		}
		if !cabi.IsMintageKey(key) {
			continue
		}
		tokenId := cabi.GetTokenIdFromMintageKey(key)
		old, err := cabi.ParseTokenInfo(value)
		if err != nil {
			continue
		}
		if tokenId == ledger.ViteTokenId {
			g.MintageInfo.TokenInfoMap[tokenId.String()] = TokenInfo{old.TokenName, old.TokenSymbol, old.TotalSupply, old.Decimals, types.AddressConsensusGroup, old.PledgeAmount, old.PledgeAddr, mintageWithdrawHeight, helper.Tt256m1, false, true}
		} else {
			if old.MaxSupply == nil {
				old.MaxSupply = big.NewInt(0)
			}
			g.MintageInfo.TokenInfoMap[tokenId.String()] = TokenInfo{old.TokenName, old.TokenSymbol, old.TotalSupply, old.Decimals, old.Owner, old.PledgeAmount, old.PledgeAddr, mintageWithdrawHeight, old.MaxSupply, old.OwnerBurnOnly, old.IsReIssuable}
		}
		log := util.NewLog(ABIMintageNew, "mint", tokenId)
		g.MintageInfo.LogList = append(g.MintageInfo.LogList, GenesisVmLog{hex.EncodeToString(log.Data), log.Topics})
		m = updateBalance(m, old.PledgeAddr, emptyBalance)
	}
	return m, g
}

func exportVoteBalanceAndStorage(m map[types.Address]*big.Int, g *Genesis, trie *trie.Trie) (map[types.Address]*big.Int, *Genesis) {
	if g.ConsensusGroupInfo == nil {
		g.ConsensusGroupInfo = &ConsensusGroupContractInfo{}
	}
	g.ConsensusGroupInfo.VoteStatusMap = make(map[string]map[string]string)
	gidStr := types.SNAPSHOT_GID.String()
	g.ConsensusGroupInfo.VoteStatusMap[gidStr] = make(map[string]string)
	iterator := trie.NewIterator(nil)
	for {
		key, value, ok := iterator.Next()
		if !ok {
			break
		}
		if isBalanceOrCode(key) || len(value) == 0 {
			continue
		}
		voterAddr := cabi.GetAddrFromVoteKey(key)
		nodeName := new(string)
		if err := cabi.ABIVote.UnpackVariable(nodeName, cabi.VariableNameVoteStatus, value); err == nil {
			g.ConsensusGroupInfo.VoteStatusMap[gidStr][voterAddr.String()] = *nodeName
		}
	}
	return m, g
}

func exportConsensusGroupBalanceAndStorage(m map[types.Address]*big.Int, g *Genesis, trie *trie.Trie) (map[types.Address]*big.Int, *Genesis) {
	if g.ConsensusGroupInfo == nil {
		g.ConsensusGroupInfo = &ConsensusGroupContractInfo{}
	}
	g.ConsensusGroupInfo.ConsensusGroupInfoMap = make(map[string]ConsensusGroupInfo)
	iterator := trie.NewIterator(nil)
	for {
		key, value, ok := iterator.Next()
		if !ok {
			break
		}
		if isBalanceOrCode(key) || len(value) == 0 {
			continue
		}
		old := new(types.ConsensusGroupInfo)
		cabi.ABIConsensusGroup.UnpackVariable(old, cabi.VariableNameConsensusGroupInfo, value)
		oldParam := new(cabi.VariableConditionRegisterOfPledge)
		cabi.ABIConsensusGroup.UnpackVariable(oldParam, cabi.VariableNameConditionRegisterOfPledge, old.RegisterConditionParam)
		m = updateBalance(m, old.Owner, emptyBalance)
		if gid := cabi.GetGidFromConsensusGroupKey(key); gid == types.SNAPSHOT_GID {
			g.ConsensusGroupInfo.ConsensusGroupInfoMap[gid.String()] = ConsensusGroupInfo{
				old.NodeCount,
				old.Interval,
				old.PerCount,
				old.RandCount,
				old.RandRank,
				uint16(1),
				uint8(0),
				old.CountingTokenId,
				old.RegisterConditionId,
				RegisterConditionParam{registerPledgeAmount, oldParam.PledgeToken, oldParam.PledgeHeight},
				old.VoteConditionId,
				VoteConditionParam{},
				old.Owner,
				old.PledgeAmount,
				old.WithdrawHeight}
		} else {
			g.ConsensusGroupInfo.ConsensusGroupInfoMap[gid.String()] = ConsensusGroupInfo{
				old.NodeCount,
				old.Interval,
				old.PerCount,
				old.RandCount,
				old.RandRank,
				uint16(48),
				uint8(1),
				old.CountingTokenId,
				old.RegisterConditionId,
				RegisterConditionParam{registerPledgeAmount, oldParam.PledgeToken, oldParam.PledgeHeight},
				old.VoteConditionId,
				VoteConditionParam{},
				old.Owner,
				old.PledgeAmount,
				old.WithdrawHeight}
		}
	}
	return m, g
}

func updateBalance(m map[types.Address]*big.Int, addr types.Address, balance *big.Int) map[types.Address]*big.Int {
	if v, ok := m[addr]; ok {
		v = v.Add(v, balance)
		m[addr] = v
	} else {
		m[addr] = new(big.Int).Set(balance)
	}
	return m
}

const (
	jsonMintage = `
	[
		{"type":"function","name":"CancelMintPledge","inputs":[{"name":"tokenId","type":"tokenId"}]},
		{"type":"function","name":"Mint","inputs":[{"name":"isReIssuable","type":"bool"},{"name":"tokenId","type":"tokenId"},{"name":"tokenName","type":"string"},{"name":"tokenSymbol","type":"string"},{"name":"totalSupply","type":"uint256"},{"name":"decimals","type":"uint8"},{"name":"maxSupply","type":"uint256"},{"name":"ownerBurnOnly","type":"bool"}]},
		{"type":"function","name":"Issue","inputs":[{"name":"tokenId","type":"tokenId"},{"name":"amount","type":"uint256"},{"name":"beneficial","type":"address"}]},
		{"type":"function","name":"Burn","inputs":[]},
		{"type":"function","name":"TransferOwner","inputs":[{"name":"tokenId","type":"tokenId"},{"name":"newOwner","type":"address"}]},
		{"type":"function","name":"ChangeTokenType","inputs":[{"name":"tokenId","type":"tokenId"}]},
		{"type":"variable","name":"mintage","inputs":[{"name":"tokenName","type":"string"},{"name":"tokenSymbol","type":"string"},{"name":"totalSupply","type":"uint256"},{"name":"decimals","type":"uint8"},{"name":"owner","type":"address"},{"name":"pledgeAmount","type":"uint256"},{"name":"withdrawHeight","type":"uint64"}]},
		{"type":"variable","name":"tokenInfo","inputs":[{"name":"tokenName","type":"string"},{"name":"tokenSymbol","type":"string"},{"name":"totalSupply","type":"uint256"},{"name":"decimals","type":"uint8"},{"name":"owner","type":"address"},{"name":"pledgeAmount","type":"uint256"},{"name":"withdrawHeight","type":"uint64"},{"name":"pledgeAddr","type":"address"},{"name":"isReIssuable","type":"bool"},{"name":"maxSupply","type":"uint256"},{"name":"ownerBurnOnly","type":"bool"}]},
		{"type":"event","name":"mint","inputs":[{"name":"tokenId","type":"tokenId","indexed":true}]},
		{"type":"event","name":"issue","inputs":[{"name":"tokenId","type":"tokenId","indexed":true}]},
		{"type":"event","name":"burn","inputs":[{"name":"tokenId","type":"tokenId","indexed":true},{"name":"address","type":"address"},{"name":"amount","type":"uint256"}]},
		{"type":"event","name":"transferOwner","inputs":[{"name":"tokenId","type":"tokenId","indexed":true},{"name":"owner","type":"address"}]},
		{"type":"event","name":"changeTokenType","inputs":[{"name":"tokenId","type":"tokenId","indexed":true}]}
	]`
	jsonPledge = `
	[
		{"type":"function","name":"Pledge", "inputs":[{"name":"beneficial","type":"address"}]},
		{"type":"function","name":"CancelPledge","inputs":[{"name":"beneficial","type":"address"},{"name":"amount","type":"uint256"}]},
		{"type":"variable","name":"pledgeInfo","inputs":[{"name":"amount","type":"uint256"},{"name":"withdrawHeight","type":"uint64"}]},
		{"type":"variable","name":"pledgeBeneficial","inputs":[{"name":"amount","type":"uint256"}]}
	]`
	jsonConsensusGroup = `
	[
		{"type":"function","name":"CreateConsensusGroup", "inputs":[{"name":"gid","type":"gid"},{"name":"nodeCount","type":"uint8"},{"name":"interval","type":"int64"},{"name":"perCount","type":"int64"},{"name":"randCount","type":"uint8"},{"name":"randRank","type":"uint8"},{"name":"repeat","type":"uint16"},{"name":"checkLevel","type":"uint8"},{"name":"countingTokenId","type":"tokenId"},{"name":"registerConditionId","type":"uint8"},{"name":"registerConditionParam","type":"bytes"},{"name":"voteConditionId","type":"uint8"},{"name":"voteConditionParam","type":"bytes"}]},
		{"type":"function","name":"CancelConsensusGroup", "inputs":[{"name":"gid","type":"gid"}]},
		{"type":"function","name":"ReCreateConsensusGroup", "inputs":[{"name":"gid","type":"gid"}]},
		{"type":"variable","name":"consensusGroupInfo","inputs":[{"name":"nodeCount","type":"uint8"},{"name":"interval","type":"int64"},{"name":"perCount","type":"int64"},{"name":"randCount","type":"uint8"},{"name":"randRank","type":"uint8"},{"name":"repeat","type":"uint16"},{"name":"checkLevel","type":"uint8"},{"name":"countingTokenId","type":"tokenId"},{"name":"registerConditionId","type":"uint8"},{"name":"registerConditionParam","type":"bytes"},{"name":"voteConditionId","type":"uint8"},{"name":"voteConditionParam","type":"bytes"},{"name":"owner","type":"address"},{"name":"pledgeAmount","type":"uint256"},{"name":"withdrawHeight","type":"uint64"}]},
		{"type":"variable","name":"registerOfPledge","inputs":[{"name":"pledgeAmount","type":"uint256"},{"name":"pledgeToken","type":"tokenId"},{"name":"pledgeHeight","type":"uint64"}]},
		
		{"type":"function","name":"Register", "inputs":[{"name":"gid","type":"gid"},{"name":"name","type":"string"},{"name":"nodeAddr","type":"address"}]},
		{"type":"function","name":"UpdateRegistration", "inputs":[{"name":"gid","type":"gid"},{"Name":"name","type":"string"},{"name":"nodeAddr","type":"address"}]},
		{"type":"function","name":"CancelRegister","inputs":[{"name":"gid","type":"gid"}, {"name":"name","type":"string"}]},
		{"type":"function","name":"Reward","inputs":[{"name":"gid","type":"gid"},{"name":"name","type":"string"},{"name":"beneficialAddr","type":"address"}]},
		{"type":"variable","name":"registration","inputs":[{"name":"name","type":"string"},{"name":"nodeAddr","type":"address"},{"name":"pledgeAddr","type":"address"},{"name":"amount","type":"uint256"},{"name":"withdrawHeight","type":"uint64"},{"name":"rewardTime","type":"int64"},{"name":"cancelTime","type":"int64"},{"name":"hisAddrList","type":"address[]"}]},
		{"type":"variable","name":"hisName","inputs":[{"name":"name","type":"string"}]},
		
		{"type":"function","name":"Vote", "inputs":[{"name":"gid","type":"gid"},{"name":"nodeName","type":"string"}]},
		{"type":"function","name":"CancelVote","inputs":[{"name":"gid","type":"gid"}]},
		{"type":"variable","name":"voteStatus","inputs":[{"name":"nodeName","type":"string"}]}
	]`
)

var (
	ABIMintageNew, _        = abi.JSONToABIContract(strings.NewReader(jsonMintage))
	ABIPledgeNew, _         = abi.JSONToABIContract(strings.NewReader(jsonPledge))
	ABIConsensusGroupNew, _ = abi.JSONToABIContract(strings.NewReader(jsonConsensusGroup))
)

func filterGenesis(g *Genesis, m map[types.Address]*big.Int) (*Genesis, map[types.Address]*big.Int) {
	gidStr := types.SNAPSHOT_GID.String()
	if g.ConsensusGroupInfo != nil && len(g.ConsensusGroupInfo.VoteStatusMap) > 0 && len(g.ConsensusGroupInfo.VoteStatusMap[types.SNAPSHOT_GID.String()]) > 0 {
		for voteAddr, nodeName := range g.ConsensusGroupInfo.VoteStatusMap[gidStr] {
			if _, ok := registerNameMap[nodeName]; !ok {
				delete(g.ConsensusGroupInfo.VoteStatusMap[gidStr], voteAddr)
			} else {
				addr, _ := types.HexToAddress(voteAddr)
				m = updateBalance(m, addr, emptyBalance)
			}
		}
	}
	return g, m
}

func printGenesis(g *Genesis) {
	v, _ := json.MarshalIndent(g, "", "\t")
	fmt.Println(string(v))
}

func printGenesisSummary(g *Genesis) {
	if g == nil {
		return
	}
	fmt.Println("genesis summary: ")
	consensusGroupCount := 0
	voteCount := 0
	sbpCount := 0
	sbpHisNameCount := 0
	pledgeCount := 0
	pledgeBeneficialCount := 0
	if g.ConsensusGroupInfo != nil {
		consensusGroupCount = len(g.ConsensusGroupInfo.ConsensusGroupInfoMap)
		voteCount = len(g.ConsensusGroupInfo.VoteStatusMap[types.SNAPSHOT_GID.String()])
		sbpCount = len(g.ConsensusGroupInfo.RegistrationInfoMap[types.SNAPSHOT_GID.String()])
		sbpHisNameCount = len(g.ConsensusGroupInfo.HisNameMap[types.SNAPSHOT_GID.String()])
		if sbpCount != sbpHisNameCount {
			fmt.Println("【data error】registration his name count error, expected " + strconv.Itoa(sbpCount) + ", got " + strconv.Itoa(sbpHisNameCount))
		}
		for _, info := range g.ConsensusGroupInfo.ConsensusGroupInfoMap {
			if _, ok := g.AccountBalanceMap[info.Owner.String()]; !ok {
				fmt.Println("【data error】consensus group owner account balance map nil, address " + info.Owner.String())
			}
		}
		for _, info := range g.ConsensusGroupInfo.RegistrationInfoMap[types.SNAPSHOT_GID.String()] {
			if _, ok := g.AccountBalanceMap[info.PledgeAddr.String()]; !ok {
				fmt.Println("【data error】registration owner account balance map nil, address " + info.PledgeAddr.String())
			}
		}
		for addr, _ := range g.ConsensusGroupInfo.VoteStatusMap[types.SNAPSHOT_GID.String()] {
			if _, ok := g.AccountBalanceMap[addr]; !ok {
				fmt.Println("【data error】vote account balance map nil, address " + addr)
			}
		}
	}
	fmt.Println("consensus group count: " + strconv.Itoa(consensusGroupCount))
	fmt.Println("vote count: " + strconv.Itoa(voteCount))
	fmt.Println("sbp count: " + strconv.Itoa(sbpCount))
	fmt.Println("sbp his name count: " + strconv.Itoa(sbpHisNameCount))
	pledgeAmountTotal := big.NewInt(0)
	pledgeAmountBeneficialTotal := big.NewInt(0)
	beneficialMap := make(map[string]interface{}, pledgeBeneficialCount)
	if g.PledgeInfo != nil {
		pledgeCount = len(g.PledgeInfo.PledgeInfoMap)
		pledgeBeneficialCount = len(g.PledgeInfo.PledgeBeneficialMap)
		for pledgeAddr, list := range g.PledgeInfo.PledgeInfoMap {
			for _, info := range list {
				beneficialMap[info.BeneficialAddr.String()] = struct{}{}
				pledgeAmountTotal.Add(pledgeAmountTotal, info.Amount)
			}
			if _, ok := g.AccountBalanceMap[pledgeAddr]; !ok {
				fmt.Println("【data error】pledge account balance map nil, address " + pledgeAddr)
			}
		}
		for _, amount := range g.PledgeInfo.PledgeBeneficialMap {
			pledgeAmountBeneficialTotal.Add(pledgeAmountBeneficialTotal, amount)
		}
	}
	fmt.Println("pledge addr count: " + strconv.Itoa(pledgeCount))
	fmt.Println("pledge beneficial count: " + strconv.Itoa(pledgeBeneficialCount))
	if expected := len(beneficialMap); expected != pledgeBeneficialCount {
		fmt.Println("【data error】pledge beneficial count not match, expected " + strconv.Itoa(expected) + ", got " + strconv.Itoa(pledgeBeneficialCount))
	}
	if pledgeAmountTotal.Cmp(pledgeAmountBeneficialTotal) != 0 {
		fmt.Println("【data error】pledge amount total not match, pledge amount total " + pledgeAmountTotal.String() + ", pledge amount beneficial total " + pledgeAmountBeneficialTotal.String())
	}
	tokenCount := 0
	logCount := 0
	if g.MintageInfo != nil {
		tokenCount = len(g.MintageInfo.TokenInfoMap)
		logCount = len(g.MintageInfo.LogList)
		if tokenCount != logCount {
			fmt.Println("【data error】mintage log count not match, expected " + strconv.Itoa(tokenCount) + ", got " + strconv.Itoa(logCount))
		}
		for _, info := range g.MintageInfo.TokenInfoMap {
			if _, ok := g.AccountBalanceMap[info.PledgeAddr.String()]; !ok {
				fmt.Println("【data error】mintage pledge account balance map nil, address " + info.PledgeAddr.String())
			}
		}
	}
	fmt.Println("token count: " + strconv.Itoa(tokenCount))
	fmt.Println("token log count: " + strconv.Itoa(logCount))

	balanceTotalMap := make(map[string]*big.Int)
	if g.AccountBalanceMap != nil {
		for _, m := range g.AccountBalanceMap {
			for tokenId, amount := range m {
				if origin, ok := balanceTotalMap[tokenId]; !ok {
					balanceTotalMap[tokenId] = amount
				} else {
					balanceTotalMap[tokenId] = origin.Add(origin, amount)
				}
			}
		}
	}
	for tokenId, amount := range balanceTotalMap {
		fmt.Println("balance total of " + tokenId + " : " + amount.String())
	}
	fmt.Println("balance total of pledge : " + pledgeAmountTotal.String())

	totalViteAmount := big.NewInt(0)
	totalViteAmount.Add(totalViteAmount, balanceTotalMap[ledger.ViteTokenId.String()])
	totalViteAmount.Add(totalViteAmount, pledgeAmountTotal)
	totalViteAmount.Add(totalViteAmount, new(big.Int).Mul(new(big.Int).Mul(big.NewInt(1e3), big.NewInt(1e18)), big.NewInt(int64(tokenCount-1))))
	totalViteAmount.Add(totalViteAmount, new(big.Int).Mul(new(big.Int).Mul(big.NewInt(1e5), big.NewInt(1e18)), big.NewInt(int64(sbpCount))))
	totalSupply := new(big.Int).Mul(big.NewInt(1e9), big.NewInt(1e18))
	if totalViteAmount.Cmp(totalSupply) != 0 {
		fmt.Println("【data error】vite token total amount not match, expected " + totalSupply.String() + ", got " + totalViteAmount.String())
	}

	vcpTotalSupply := big.NewInt(1e10)
	if balanceTotalMap["tti_251a3e67a41b5ea2373936c8"].Cmp(vcpTotalSupply) != 0 {
		fmt.Println("【data error】vcp token total amount not match, expected " + vcpTotalSupply.String() + ", got " + balanceTotalMap["tti_251a3e67a41b5ea2373936c8"].String())
	}
}
