package nodemanager

import (
	"fmt"
	"github.com/shopspring/decimal"
	"github.com/tealeg/xlsx"
	"github.com/vitelabs/go-vite/chain"
	"github.com/vitelabs/go-vite/cmd/utils"
	"github.com/vitelabs/go-vite/common/db/xleveldb/errors"
	"github.com/vitelabs/go-vite/common/fork"
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/ledger"
	"github.com/vitelabs/go-vite/node"
	"github.com/vitelabs/go-vite/vm/contracts/dex"
	"gopkg.in/urfave/cli.v1"
	"math/big"
	"strings"
	"time"
)

type ExportNodeManager struct {
	ctx  *cli.Context
	node *node.Node

	chain chain.Chain
}

//var digits = big.NewInt(1000000000000000000)

func NewExportNodeManager(ctx *cli.Context, maker NodeMaker) (*ExportNodeManager, error) {
	node, err := maker.MakeNode(ctx)
	if err != nil {
		return nil, err
	}

	return &ExportNodeManager{
		ctx:  ctx,
		node: node,
	}, nil
}

func (nodeManager *ExportNodeManager) getBeforeTime() int64 {
	beforeTimeStamp := int64(0)
	if nodeManager.ctx.GlobalIsSet(utils.ExportBeforeTimeFlags.Name) {
		beforeTimeStamp = nodeManager.ctx.GlobalInt64(utils.ExportBeforeTimeFlags.Name)
	}
	return beforeTimeStamp
}

func (nodeManager *ExportNodeManager) getSbHeight() uint64 {
	sbHeight := uint64(0)
	if nodeManager.ctx.GlobalIsSet(utils.ExportSbHeightFlags.Name) {
		sbHeight = nodeManager.ctx.GlobalUint64(utils.ExportSbHeightFlags.Name)
	}
	return sbHeight
}
func (nodeManager *ExportNodeManager) getTokenIdList() []types.TokenTypeId {
	var tokenIds []types.TokenTypeId

	if nodeManager.ctx.GlobalIsSet(utils.ExportTokenFlags.Name) {
		tokenFlag := nodeManager.ctx.GlobalString(utils.ExportTokenFlags.Name)
		tokenIdStrs := strings.Split(tokenFlag, ",")
		for _, tokenIdStr := range tokenIdStrs {
			tokenIdStr = strings.TrimSpace(tokenIdStr)

			tokenId, err := types.HexToTokenTypeId(tokenIdStr)
			if err != nil {
				panic(err)
			}
			tokenIds = append(tokenIds, tokenId)
		}
	}

	return tokenIds
}

func (nodeManager *ExportNodeManager) Start() error {
	if err := nodeManager.initChain(); err != nil {
		return err
	}

	c := nodeManager.chain

	sbHeight := nodeManager.getSbHeight()
	var sbHeader *ledger.SnapshotBlock
	if sbHeight >= 0 {
		var err error
		sbHeader, err = c.GetSnapshotHeaderByHeight(sbHeight)
		if err != nil {
			return err
		}

	}

	beforeTime := nodeManager.getBeforeTime()
	if sbHeader == nil && beforeTime > 0 {
		var err error
		t := time.Unix(beforeTime, 0)
		sbHeader, err = c.GetSnapshotHeaderBeforeTime(&t)
		if err != nil {
			return err
		}
	}
	if sbHeader == nil {
		return errors.New(fmt.Sprintf("sbHeader is nil, beforeTime is %d, height is %d", beforeTime, sbHeight))
	}

	tokenIds := nodeManager.getTokenIdList()
	if len(tokenIds) <= 0 {
		return errors.New("len(tokenIds) is 0")
	}

	var addrList []types.Address
	c.IterateAccounts(func(addr types.Address, accountId uint64, err error) bool {
		addrStr := addr.String()
		if !types.IsContractAddr(addr) &&
			addrStr != "vite_d9d01e1cef6e5b90bcad0a453de34b73c25c0a5b575a2933bc" &&
			addrStr != "vite_08fd8c3ff9369031e70e116d23ef60b1263c6d16f38126b619" &&
			addrStr != "vite_ff38174de69ddc63b2e05402e5c67c356d7d17e819a0ffadee" {
			addrList = append(addrList, addr)

		}
		return true
	})

	// dex fund
	dexResult, err := nodeManager.getDexFunds(sbHeader.Hash, addrList, tokenIds)
	if err != nil {
		return err
	}
	for addr, balanceMap := range dexResult {
		for tokenId, balance := range balanceMap {
			fmt.Println(addr, tokenId, balance)
		}
	}

	balanceSummary := make(map[types.Address][]*decimal.Decimal, 0)
	for _, tokenId := range tokenIds {
		balanceMap, err := c.GetConfirmedBalanceList(addrList, tokenId, sbHeader.Hash)
		if err != nil {
			return err
		}
		tokenInfo, err := c.GetTokenInfoById(tokenId)
		if err != nil {
			return err
		}

		for _, addr := range addrList {
			balance, ok := balanceMap[addr]
			if !ok {
				balance = big.NewInt(0)
			}

			if accountDex, ok := dexResult[addr]; ok {
				if accountDexFund, ok2 := accountDex[tokenId]; ok2 {
					balance.Add(balance, accountDexFund)
				}
			}

			decimal := decimal.NewFromBigInt(balance, 0).Div(decimal.New(1, int32(tokenInfo.Decimals)))
			balanceSummary[addr] = append(balanceSummary[addr], &decimal)
		}
	}

	totals := make([]decimal.Decimal, len(tokenIds))
	for _, amounts := range balanceSummary {
		for index, amount := range amounts {
			totals[index] = totals[index].Add(*amount)
		}
	}
	for index, total := range totals {
		fmt.Println(tokenIds[index], total)
	}

	var content [][]string
	for addr, balances := range balanceSummary {
		line := []string{addr.String()}
		for _, balance := range balances {
			if balance != nil {
				line = append(line, balance.String())
			} else {
				line = append(line, "0")
			}
		}
		if len(line) != len(tokenIds)+1 {
			panic(fmt.Sprintf("The length of %s is %d", addr, len(line)))
		}
		content = append(content, line)

	}

	timeFormat := fmt.Sprintf("%d_%d_%d_%d_%d_%d", sbHeader.Timestamp.Year(),
		sbHeader.Timestamp.Month(),
		sbHeader.Timestamp.Day(),
		sbHeader.Timestamp.Hour(),
		sbHeader.Timestamp.Minute(),
		sbHeader.Timestamp.Second())
	if err := nodeManager.writeExcel(timeFormat+".xlsx", content); err != nil {
		return err
	}

	fmt.Printf("export %d address. time is %s, snapshot height is %d, snapshot hash is %s.\n", len(content), timeFormat, sbHeader.Height, sbHeader.Hash)

	return nil

}

func (nodeManager *ExportNodeManager) getDexFunds(snapshotBlockHash types.Hash, addrList []types.Address, tokenIdList []types.TokenTypeId) (map[types.Address]map[types.TokenTypeId]*big.Int, error) {

	sd, err := nodeManager.chain.NewSnapshotStorageDatabase(snapshotBlockHash, types.AddressDexFund)
	if err != nil {
		return nil, err
	}
	result := make(map[types.Address]map[types.TokenTypeId]*big.Int)
	for _, addr := range addrList {
		data, err := sd.GetValue(dex.GetUserFundKey(addr))
		if err != nil {
			return nil, err
		}
		if len(data) <= 0 {
			continue
		}

		dexFund := &dex.UserFund{}
		if err := dexFund.DeSerialize(data); err != nil {
			return nil, err
		}

		result[addr] = make(map[types.TokenTypeId]*big.Int)
		for _, tokenId := range tokenIdList {
			fundInfo, err := dex.GetAccountFundInfo(dexFund, &tokenId)
			if err != nil {
				return nil, err
			}

			amount := big.NewInt(0)

			for _, fund := range fundInfo {
				amount.Add(fund.Available, fund.Locked)
			}

			result[addr][tokenId] = amount
		}
	}

	return result, nil
}

func (nodeManager *ExportNodeManager) writeExcel(filename string, content [][]string) error {
	file := xlsx.NewFile()
	sheet, err := file.AddSheet("Sheet1")
	if err != nil {
		return err
	}
	sheet.SetColWidth(0, 0, 58)

	sheet.SetColWidth(1, len(content)-1, 36)

	for _, line := range content {
		row := sheet.AddRow()

		for _, col := range line {
			cell := row.AddCell()

			cell.Value = col

		}
	}

	if filename == "" {
		filename = "account_tokens.xlsx"
	}

	return file.Save(filename)
}

func (nodeManager *ExportNodeManager) initChain() error {
	// Start up the node
	node := nodeManager.node
	viteConfig := node.ViteConfig()

	dataDir := viteConfig.DataDir
	chainCfg := viteConfig.Chain
	genesisCfg := viteConfig.Genesis
	// set fork points
	fork.SetForkPoints(viteConfig.ForkPoints)

	c := chain.NewChain(dataDir, chainCfg, genesisCfg)

	nodeManager.chain = c

	if err := c.Init(); err != nil {
		return err
	}

	if err := c.Start(); err != nil {
		return err
	}
	return nil
}
