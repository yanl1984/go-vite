package generator

import (
	"errors"
	"math/big"

	"github.com/vitelabs/go-vite/common/helper"
	"github.com/vitelabs/go-vite/common/math"
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/ledger"
	"github.com/vitelabs/go-vite/vm_db"
)

// IncomingMessageToBlock returns a complete block by a IncomingMessage.
func IncomingMessageToBlock(vmDb vm_db.VmDb, im *IncomingMessage) (*ledger.AccountBlock, error) {
	block := &ledger.AccountBlock{
		BlockType:      im.BlockType,
		AccountAddress: im.AccountAddress,
		// after vm
		Quota:         0,
		SendBlockList: nil,
		LogHash:       nil,
		Hash:          types.Hash{},
		Signature:     nil,
		PublicKey:     nil,
	}
	switch im.BlockType {
	case ledger.BlockTypeSendCreate, ledger.BlockTypeSendRefund, ledger.BlockTypeSendCall:
		block.Data = im.Data

		block.FromBlockHash = types.Hash{}
		if im.ToAddress != nil {
			block.ToAddress = *im.ToAddress
		} else if im.BlockType != ledger.BlockTypeSendCreate {
			return nil, errors.New("pack send failed, toAddress can't be nil")
		}

		if im.TokenId == nil || *im.TokenId == types.ZERO_TOKENID {
			if im.Amount != nil && im.Amount.Cmp(helper.Big0) != 0 {
				return nil, errors.New("pack send failed, tokenId can't be empty when amount have actual value")
			}
			block.Amount = big.NewInt(0)
			block.TokenId = types.ZERO_TOKENID
		} else {
			if im.Amount == nil {
				block.Amount = big.NewInt(0)
			} else {
				if im.Amount.Sign() < 0 || im.Amount.BitLen() > math.MaxBigIntLen {
					return nil, errors.New("pack send failed, amount out of bounds")
				}
				block.Amount = im.Amount
			}
			block.TokenId = *im.TokenId
		}
		// PrevHash, Height
		prevBlock, err := vmDb.PrevAccountBlock()
		if err != nil {
			return nil, err
		}
		if prevBlock == nil {
			return nil, errors.New("account address doesn't exist")
		}

		block.Height = prevBlock.Height + 1
		block.PrevHash = prevBlock.Hash
	case ledger.BlockTypeReceive:
		block.Data = nil

		block.ToAddress = types.Address{}
		if im.FromBlockHash != nil && *im.FromBlockHash != types.ZERO_HASH {
			block.FromBlockHash = *im.FromBlockHash
		} else {
			return nil, errors.New("pack recvBlock failed, cause sendBlock.Hash is invaild")
		}

		if im.Amount != nil && im.Amount.Cmp(helper.Big0) != 0 {
			return nil, errors.New("pack recvBlock failed, amount is invalid")
		}
		if im.TokenId != nil && *im.TokenId != types.ZERO_TOKENID {
			return nil, errors.New("pack recvBlock failed, cause tokenId is invaild")
		}

		// PrevHash, Height
		prevBlock, err := vmDb.PrevAccountBlock()
		if err != nil {
			return nil, err
		}
		var prevHash types.Hash
		var preHeight uint64
		if prevBlock != nil {
			prevHash = prevBlock.Hash
			preHeight = prevBlock.Height
		}
		block.PrevHash = prevHash
		block.Height = preHeight + 1

	default:
		//ledger.BlockTypeReceiveError:
		return nil, errors.New("generator can't solve this block type " + string(im.BlockType))
	}
	return block, nil
}

// IncomingMessage carries the necessary transaction info.
type IncomingMessage struct {
	BlockType byte

	AccountAddress types.Address
	ToAddress      *types.Address
	FromBlockHash  *types.Hash

	TokenId *types.TokenTypeId
	Amount  *big.Int
	Data    []byte

	Difficulty *big.Int
}
