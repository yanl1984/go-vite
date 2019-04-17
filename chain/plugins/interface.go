package chain_plugins

import (
	"github.com/syndtr/goleveldb/leveldb"
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/ledger"
)

type Chain interface {
	NewDb(dirName string) (*leveldb.DB, error)
	GetLatestSnapshotBlock() *ledger.SnapshotBlock
	GetSnapshotBlocksByHeight(height uint64, higher bool, count uint64) ([]*ledger.SnapshotBlock, error)
	GetSubLedger(startHeight, endHeight uint64) ([]*ledger.SnapshotChunk, error)
	GetAccountBlockByHash(blockHash types.Hash) (*ledger.AccountBlock, error)

	IsAccountBlockExisted(hash types.Hash) (bool, error)
	IsGenesisAccountBlock(hash types.Hash) bool

	GetAllUnconfirmedBlocks() []*ledger.AccountBlock
}

type Plugin interface {
	InsertAccountBlock(*leveldb.Batch, *ledger.AccountBlock) error

	InsertSnapshotBlock(*leveldb.Batch, *ledger.SnapshotBlock, []*ledger.AccountBlock) error

	DeleteAccountBlocks(*leveldb.Batch, []*ledger.AccountBlock) error

	DeleteSnapshotBlocks(*leveldb.Batch, []*ledger.SnapshotChunk) error
}