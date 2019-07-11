package nodemanager

import (
	"fmt"
	"github.com/tealeg/xlsx"
	"github.com/vitelabs/go-vite/chain"
	"github.com/vitelabs/go-vite/cmd/utils"
	"github.com/vitelabs/go-vite/common/db/xleveldb/errors"
	"github.com/vitelabs/go-vite/common/fork"
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/node"
	"gopkg.in/urfave/cli.v1"
	"math/big"

	"strings"
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

	sbHeight := nodeManager.getSbHeight()
	if sbHeight <= 0 {
		return errors.New("sbHeight is 0")
	}

	tokenIds := nodeManager.getTokenIdList()
	if len(tokenIds) <= 0 {
		return errors.New("len(tokenIds) is 0")
	}

	c := nodeManager.chain

	sbHeader, err := c.GetSnapshotHeaderByHeight(sbHeight)
	if err != nil {
		return err
	}

	if sbHeader == nil {
		return errors.New(fmt.Sprintf("sbHeader is nil, height is %d", sbHeight))
	}
	var addrList []types.Address
	c.IterateAccounts(func(addr types.Address, accountId uint64, err error) bool {
		addrList = append(addrList, addr)
		return true
	})

	balanceSummary := make(map[types.Address][]*big.Int, 0)
	for _, tokenId := range tokenIds {
		balanceMap, err := c.GetConfirmedBalanceList(addrList, tokenId, sbHeader.Hash)
		if err != nil {
			return err
		}
		for _, addr := range addrList {
			balance, ok := balanceMap[addr]
			if !ok {
				balance = big.NewInt(0)
			}
			balanceSummary[addr] = append(balanceSummary[addr], balance)

		}

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

	if err := nodeManager.writeExcel(content); err != nil {
		return err
	}

	fmt.Printf("export %d address. snapshot height is %d, snapshot hash is %s.\n", len(content), sbHeight, sbHeader.Hash)

	return nil

}

func (nodeManager *ExportNodeManager) writeExcel(content [][]string) error {
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

	excelFileName := "account_tokens.xlsx"
	return file.Save(excelFileName)
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
