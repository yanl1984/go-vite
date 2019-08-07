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
	"github.com/vitelabs/go-vite/vm/contracts/abi"
	"github.com/vitelabs/go-vite/vm/contracts/dex"
	"gopkg.in/urfave/cli.v1"
	"math/big"
	"strconv"
	"strings"
	"time"
)

type ExportNodeManager struct {
	ctx  *cli.Context
	node *node.Node

	chain chain.Chain
}

//var digits = big.NewInt(1000000000000000000)

func formatTime(timestamp time.Time) string {
	return fmt.Sprintf("%d_%d_%d_%d_%d_%d", timestamp.Year(),
		timestamp.Month(),
		timestamp.Day(),
		timestamp.Hour(),
		timestamp.Minute(),
		timestamp.Second())
}

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

func (nodeManager *ExportNodeManager) Start() error {
	if err := nodeManager.initChain(); err != nil {
		return err
	}

	endSbHeader, err := nodeManager.getSbHeader(true)
	if err != nil {
		return err
	}
	if endSbHeader == nil {
		return errors.New("endSbHeader is nil")
	}

	addrList := nodeManager.getAllAddrList()

	dataType := nodeManager.getDataType()

	if dataType == "balance" {
		tokenIds := nodeManager.getTokenIdList()
		if len(tokenIds) <= 0 {
			return errors.New("len(tokenIds) is 0")
		}

		return nodeManager.exportBalance(endSbHeader, addrList, tokenIds)
	} else if dataType == "pledge" {

		endPledgeAmounts, err := nodeManager.getSnapshotPledgeAmounts(endSbHeader.Hash, addrList)
		if err != nil {
			return err
		}

		if len(endPledgeAmounts) != len(addrList) {
			return errors.New(fmt.Sprintf("endPledgeAmounts.length is %d,  addrList.length is %d", len(endPledgeAmounts), len(addrList)))
		}

		startSbHeader, err := nodeManager.getSbHeader(false)
		if err != nil {
			return err
		}

		var fileName = ""
		var content [][]string

		if startSbHeader != nil {
			pledgeAmounts := make(map[types.Address][]decimal.Decimal)

			startPledgeAmounts, err := nodeManager.getSnapshotPledgeAmounts(startSbHeader.Hash, addrList)
			if err != nil {
				return err
			}
			for addr, amount := range endPledgeAmounts {
				pledgeAmounts[addr] = make([]decimal.Decimal, 3)

				startAmount, ok := startPledgeAmounts[addr]
				if !ok {
					pledgeAmounts[addr][0] = amount
				} else {
					pledgeAmounts[addr][0] = amount.Sub(startAmount)
				}
				pledgeAmounts[addr][1] = startAmount
				pledgeAmounts[addr][2] = amount
			}

			fileName = "pledge_" + formatTime(*startSbHeader.Timestamp) + "~" + formatTime(*endSbHeader.Timestamp)

			for addr, amounts := range pledgeAmounts {
				content = append(content, []string{addr.String(), amounts[0].String(), amounts[1].String(), amounts[2].String()})
			}

		} else {
			fileName = "pledge_" + formatTime(*endSbHeader.Timestamp)
			for addr, amounts := range endPledgeAmounts {
				content = append(content, []string{addr.String(), amounts.String()})
			}

		}
		if err := nodeManager.writeExcel(fileName+".xlsx", content); err != nil {
			return err
		}

		return nil
	} else if dataType == "transaction" {
		startSbHeader, err := nodeManager.getSbHeader(false)
		if err != nil {
			return err
		}
		if startSbHeader == nil {
			return errors.New("startSbHeader is nil")
		}
		txCountsMap := make(map[string]map[types.Address]uint64)
		totalTxCounts := make(map[types.Address]uint64)

		for h := startSbHeader.Height; h < endSbHeader.Height; {
			chunks, err := nodeManager.chain.GetSubLedger(h, h+75)
			if err != nil {
				return err
			}

			txCounts := make(map[types.Address]uint64)

			for _, chunk := range chunks {
				if len(chunk.AccountBlocks) > 0 {
					for _, ab := range chunk.AccountBlocks {
						if _, ok := txCounts[ab.AccountAddress]; !ok {
							txCounts[ab.AccountAddress] = 0
						}
						txCounts[ab.AccountAddress]++

						if _, ok := totalTxCounts[ab.AccountAddress]; !ok {
							totalTxCounts[ab.AccountAddress] = 0
						}
						totalTxCounts[ab.AccountAddress]++
					}
				}
			}

			txCountsMap[fmt.Sprintf("%d~%d", h, h+75)] = txCounts
			h = h + 75

		}

		if err := nodeManager.printTx("totaltx_"+formatTime(*startSbHeader.Timestamp)+"~"+formatTime(*endSbHeader.Timestamp), map[string]map[types.Address]uint64{"total": totalTxCounts}); err != nil {
			return err
		}

		var content [][]string
		for roundStr, txCounts := range txCountsMap {
			for addr, count := range txCounts {
				items := []string{roundStr, addr.String(), strconv.FormatUint(count, 10)}
				content = append(content, items)
			}

		}

		if err := nodeManager.writeExcel("txdetail_"+formatTime(*startSbHeader.Timestamp)+"_"+formatTime(*endSbHeader.Timestamp), content); err != nil {
			return err
		}

		return nil

	}

	panic(fmt.Sprintf("Unknown dataType: %s", dataType))
	return nil
}

func (nodeManager *ExportNodeManager) printTx(fileName string, txMapList map[string]map[types.Address]uint64) error {

	file := xlsx.NewFile()

	for key, txMap := range txMapList {
		var content [][]string
		for addr, amounts := range txMap {
			content = append(content, []string{addr.String(), strconv.FormatUint(amounts, 10)})
		}

		file.AppendSheet(nodeManager.createSheet(content), key)
	}

	return file.Save("./xlsx_outputs/" + fileName + ".xlsx")
}

func (nodeManager *ExportNodeManager) getEndTimestamp() int64 {
	endTimestamp := int64(0)
	if nodeManager.ctx.GlobalIsSet(utils.ExportEndTimeFlags.Name) {
		endTimestamp = nodeManager.ctx.GlobalInt64(utils.ExportEndTimeFlags.Name)
	}
	return endTimestamp
}

func (nodeManager *ExportNodeManager) getBeginTimestamp() int64 {
	beginTimestamp := int64(0)
	if nodeManager.ctx.GlobalIsSet(utils.ExportBeginTimeFlags.Name) {
		beginTimestamp = nodeManager.ctx.GlobalInt64(utils.ExportBeginTimeFlags.Name)
	}
	return beginTimestamp
}

func (nodeManager *ExportNodeManager) getEndSbHeight() uint64 {
	sbHeight := uint64(0)
	if nodeManager.ctx.GlobalIsSet(utils.ExportEndSbHeightFlags.Name) {
		sbHeight = nodeManager.ctx.GlobalUint64(utils.ExportEndSbHeightFlags.Name)
	}
	return sbHeight
}

func (nodeManager *ExportNodeManager) getBeginSbHeight() uint64 {
	sbHeight := uint64(0)
	if nodeManager.ctx.GlobalIsSet(utils.ExportBeginSbHeightFlags.Name) {
		sbHeight = nodeManager.ctx.GlobalUint64(utils.ExportBeginSbHeightFlags.Name)
	}
	return sbHeight
}

func (nodeManager *ExportNodeManager) getDataType() string {
	dataType := "balance"
	if nodeManager.ctx.GlobalIsSet(utils.ExportTypeFlags.Name) {
		dataType = nodeManager.ctx.GlobalString(utils.ExportTypeFlags.Name)
	}
	return dataType
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

func (nodeManager *ExportNodeManager) getSbHeader(isEnd bool) (*ledger.SnapshotBlock, error) {

	c := nodeManager.chain

	sbHeight := uint64(0)
	if isEnd {
		sbHeight = nodeManager.getEndSbHeight()
	} else {
		sbHeight = nodeManager.getBeginSbHeight()
	}

	if sbHeight > 0 {
		return c.GetSnapshotHeaderByHeight(sbHeight)
	}

	if isEnd {
		endTimestamp := nodeManager.getEndTimestamp()
		if endTimestamp <= 0 {
			return nil, nil
		}
		t := time.Unix(endTimestamp, 1)
		return c.GetSnapshotHeaderBeforeTime(&t)

	} else {
		beginTimestamp := nodeManager.getBeginTimestamp()
		if beginTimestamp <= 0 {
			return nil, nil
		}
		t := time.Unix(beginTimestamp, 0)

		sbHeader, err := c.GetSnapshotHeaderBeforeTime(&t)
		if err != nil {
			return nil, err
		}
		if sbHeader == nil {
			return nil, nil
		}
		return c.GetSnapshotHeaderByHeight(sbHeader.Height + 1)
	}

}

func (nodeManager *ExportNodeManager) getAllAddrList() []types.Address {
	var addrList []types.Address
	nodeManager.chain.IterateAccounts(func(addr types.Address, accountId uint64, err error) bool {
		addrStr := addr.String()
		if !types.IsContractAddr(addr) &&
			addrStr != "vite_d9d01e1cef6e5b90bcad0a453de34b73c25c0a5b575a2933bc" &&
			addrStr != "vite_08fd8c3ff9369031e70e116d23ef60b1263c6d16f38126b619" &&
			addrStr != "vite_ff38174de69ddc63b2e05402e5c67c356d7d17e819a0ffadee" {
			addrList = append(addrList, addr)

		}
		return true
	})
	return addrList

}

func (nodeManager *ExportNodeManager) getSnapshotPledgeAmounts(sbHash types.Hash, addrList []types.Address) (map[types.Address]decimal.Decimal, error) {
	c := nodeManager.chain

	pledgeAmounts := make(map[types.Address]decimal.Decimal)

	//latestAb, err := c.GetLatestAccountBlock(types.AddressPledge)
	//prevHash, err := getPrevBlockHash(c, addr)
	//if err != nil {
	//	return nil, err
	//}

	_, _, stateDB := c.DBs()
	sd, err := stateDB.NewStorageDatabase(sbHash, types.AddressPledge)

	//db, err := vm_db.NewVmDb(c, &types.AddressPledge, &sbHash, &latestAb.Hash)
	if err != nil {
		return nil, err
	}

	for _, addr := range addrList {
		_, amount, err := abi.GetPledgeInfoList(sd, addr)
		if err != nil {
			return nil, err
		}

		if amount.Cmp(big.NewInt(0)) > 0 {
			newAmount := decimal.NewFromBigInt(amount, 0).Div(decimal.New(1, int32(18)))

			pledgeAmounts[addr] = newAmount
		} else {
			pledgeAmounts[addr] = decimal.New(0, 0)
		}

	}
	return pledgeAmounts, nil
}

func (nodeManager *ExportNodeManager) exportBalance(sbHeader *ledger.SnapshotBlock, addrList []types.Address, tokenIds []types.TokenTypeId) error {
	c := nodeManager.chain
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
	_, _, stateDB := nodeManager.chain.DBs()
	sd, err := stateDB.NewStorageDatabase(snapshotBlockHash, types.AddressDexFund)
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

func (nodeManager *ExportNodeManager) createSheet(content [][]string) xlsx.Sheet {
	sheet := xlsx.Sheet{}

	sheet.SetColWidth(0, 0, 58)

	sheet.SetColWidth(1, len(content)-1, 36)

	for _, line := range content {
		row := sheet.AddRow()

		for _, col := range line {
			cell := row.AddCell()

			cell.Value = col

		}
	}
	return sheet
}
func (nodeManager *ExportNodeManager) writeExcel(filename string, content [][]string) error {
	file := xlsx.NewFile()

	file.AppendSheet(nodeManager.createSheet(content), "Sheet1")
	if filename == "" {
		filename = "./xlsx_outputs/account_tokens.xlsx"
	}
	return file.Save("./xlsx_outputs/" + filename + ".xlsx")
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
