package chain_plugins

import (
	"fmt"
	"github.com/vitelabs/go-vite/chain/db"
	"github.com/vitelabs/go-vite/chain/flusher"
	"github.com/vitelabs/go-vite/chain/utils"
	"github.com/vitelabs/go-vite/common/db/xleveldb"
	"github.com/vitelabs/go-vite/common/db/xleveldb/errors"
	"github.com/vitelabs/go-vite/common/db/xleveldb/util"
	"github.com/vitelabs/go-vite/common/math"
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/ledger"
	"github.com/vitelabs/go-vite/log15"
	"math/big"
	"os"
	"path"
	"sync"
)

const PluginKeyUnreceivedInfo string = "plugin_unreceived"

type UnreceivedInfo struct {
	chain Chain

	store   *chain_db.Store
	flusher *chain_flusher.Flusher

	currentHashHeight *ledger.HashHeight // chain success event
	snapshotCache     *snapshotCache     // chain success event

	mu sync.RWMutex

	log log15.Logger
}

func NewUnreceivedInfo(chain Chain, chainDir string) (Plugin, error) {
	store, err := chain_db.NewStore(path.Join(chainDir, PluginKeyUnreceivedInfo), PluginKeyUnreceivedInfo)
	if err != nil {
		return nil, err
	}
	return &UnreceivedInfo{
		chain:         chain,
		store:         store,
		snapshotCache: newSnapshotCache(),
		log:           log15.New("plugin", PluginKeyUnreceivedInfo),
	}, nil
}

func (ui *UnreceivedInfo) GetName() string {
	return PluginKeyUnreceivedInfo
}

func (ui *UnreceivedInfo) GetStore() (bool, *chain_db.Store) {
	return true, ui.store
}

func (ui *UnreceivedInfo) SetStore(store *chain_db.Store) {
	return // custom store
}

func (ui *UnreceivedInfo) Init(flusher *chain_flusher.Flusher) error {

	ui.flusher = flusher

	latestSb := ui.chain.GetLatestSnapshotBlock()
	if latestSb == nil {
		ui.log.Error("Init failed cause latest snapshotblock is nil")
		return fmt.Errorf("failed to get latest snapshotblock")
	}
	latestSnapshotCheckpoint, err := ui.getNearestCheckpoint(latestSb.Height)
	if err != nil {
		return err
	}
	if _, err := ui.prepareSnapshotData(&ledger.HashHeight{Height: latestSb.Height, Hash: latestSb.Hash}, latestSnapshotCheckpoint); err != nil {
		return err
	}
	ui.chain.GetAllUnconfirmedBlocks()
	return nil
}

func (ui *UnreceivedInfo) InsertAccountBlock(batch *leveldb.Batch, block *ledger.AccountBlock) error {
	ui.mu.RLock()
	defer ui.mu.RUnlock()
	return nil
}

func (ui *UnreceivedInfo) DeleteAccountBlocks(batch *leveldb.Batch, blocks []*ledger.AccountBlock) error {
	ui.mu.RLock()
	defer ui.mu.RUnlock()
	return nil
}

func (ui *UnreceivedInfo) InsertSnapshotBlock(batch *leveldb.Batch, snapshotBlock *ledger.SnapshotBlock, confirmedBlocks []*ledger.AccountBlock) error {
	ui.mu.RLock()
	defer ui.mu.RUnlock()

	confirmedUnreceivedInfo, err := aggregateBlocks(ui.chain, confirmedBlocks)
	if err != nil {
		return err
	}
	if encounterCheckPoint(snapshotBlock.Height) {
		if err := ui.writeSnapshot(batch, snapshotBlock, confirmedUnreceivedInfo); err != nil {
			return err
		}
	}
	return nil
}

func (ui *UnreceivedInfo) InsertSnapshotBlockSuccess(batch *leveldb.Batch, snapshotBlock *ledger.SnapshotBlock, confirmedBlocks []*ledger.AccountBlock) error {
	ui.currentHashHeight = &ledger.HashHeight{Height: snapshotBlock.Height, Hash: snapshotBlock.Hash}
	var confirmedUnreceivedInfo signAddrUnreceivedMap
	if !encounterCheckPoint(snapshotBlock.Height) {
		var err error
		if confirmedUnreceivedInfo, err = aggregateBlocks(ui.chain, confirmedBlocks); err != nil {
			return err
		}
	}
	ui.snapshotCache.Push(ui.currentHashHeight, confirmedUnreceivedInfo)
	return nil
}

func (ui *UnreceivedInfo) DeleteSnapshotBlocks(batch *leveldb.Batch, chunks []*ledger.SnapshotChunk) error {
	var deleteToHeight *uint64
	for _, v := range chunks {
		if deleteToHeight == nil || v.SnapshotBlock != nil && (v.SnapshotBlock.Height < *deleteToHeight) {
			deleteToHeight = &v.SnapshotBlock.Height
		}
	}
	if deleteToHeight != nil {
		return ui.deleteCheckpointAfter(batch, *deleteToHeight)
	}
	return nil
}

func (ui *UnreceivedInfo) DeleteSnapshotBlockSuccess(batch *leveldb.Batch, chunks []*ledger.SnapshotChunk) error { // todo
	var deleteToHeight *ledger.HashHeight
	for _, v := range chunks {
		if deleteToHeight == nil || v.SnapshotBlock != nil && (v.SnapshotBlock.Height < deleteToHeight.Height) {
			deleteToHeight = &ledger.HashHeight{Height: v.SnapshotBlock.Height, Hash: v.SnapshotBlock.Hash}
		}
	}
	if index, exist := ui.snapshotCache.ExistCachePoint(deleteToHeight.Hash, deleteToHeight.Height); exist {
		ui.snapshotCache.RemoveAfter(ui.snapshotCache.lastIndex(index))
	}
	return nil
}

func (ui *UnreceivedInfo) RemoveNewUnconfirmed(rollbackBatch *leveldb.Batch, allUnconfirmedBlocks []*ledger.AccountBlock) error {
	return nil
}

func (ui *UnreceivedInfo) InsertAccountBlockSuccess(*leveldb.Batch, []*ledger.AccountBlock) error {
	return nil
}

func (ui *UnreceivedInfo) DeleteAccountBlocksSuccess(*leveldb.Batch, []*ledger.AccountBlock) error {
	return nil
}

func (ui *UnreceivedInfo) GetAccountInfo(addr types.Address) (*ledger.AccountInfo, error) {
	ui.mu.RLock()
	defer ui.mu.RUnlock()

	// get current snapshot unreceived info map
	var addrInfoMap TokenBalanceInfoMap
	latestCheckPoint, err := ui.getNearestCheckpoint(ui.currentHashHeight.Height)
	if err != nil {
		return nil, err
	}
	if addrInfoMap, _ = latestCheckPoint.UnreceivedInfo[addr]; addrInfoMap == nil {
		addrInfoMap = make(TokenBalanceInfoMap)
	}

	// get unconfirmed blocks,if exists plus to snapshot meta
	unconfirmedSnapshotMeta, err := aggregateBlocks(ui.chain, ui.chain.GetUnconfirmedBlocks(addr))
	if err != nil {
		return nil, err
	}

	snapshotCache, err := ui.snapshotCache.getSnapshotMetasAfterHeight(latestCheckPoint.HashHeight.Height + 1)
	if err != nil {
		return nil, err
	}

	if signInfoMap := aggregateCacheUnreceivedInfoMap(map[types.Address]bool{addr: true}, snapshotCache, unconfirmedSnapshotMeta); signInfoMap != nil {
		if signAddrInfo, ok := signInfoMap[addr]; ok && signAddrInfo != nil {
			if err := addrInfoMap.Add(signAddrInfo); err != nil {
				return nil, err
			}
		}
	}

	totalNumber := uint64(0)
	for _, v := range addrInfoMap {
		totalNumber += v.Number
	}
	return &ledger.AccountInfo{
		AccountAddress:      addr,
		TotalNumber:         totalNumber,
		TokenBalanceInfoMap: addrInfoMap,
	}, nil
}

func (ui *UnreceivedInfo) rebuildAll() (*UnreceivedSnapshotMeta, error) {
	latestHashHeight := ui.getLatestSnapshotHashHeight()
	if latestHashHeight == nil {
		return nil, errors.New("failed to get latestSnapshot")
	}

	if err := ui.store.Close(); err != nil {
		return nil, err
	}

	os.RemoveAll(ui.store.GetDataDir())
	ui.log.Info("rebuildAll remove db_dir:"+ui.store.GetDataDir(), "latestHashHeight", fmt.Sprintf("[%v,%v]", latestHashHeight.Height, latestHashHeight.Hash))

	store, err := chain_db.NewStore(ui.store.GetDataDir(), PluginKeyUnreceivedInfo)
	if err != nil {
		return nil, err
	}
	ui.store = store
	ui.log.Info("rebuildAll new db_dir:"+ui.store.GetDataDir(), "latestHashHeight", fmt.Sprintf("[%v,%v]", latestHashHeight.Height, latestHashHeight.Hash))

	ui.flusher.ReplaceStore(ui.store.Id(), store)

	unsavedInfoMap, err := ui.loadAllUnreceived(latestHashHeight)
	if err != nil {
		return nil, err
	}
	if err := ui.writeSnapshotDirectly(latestHashHeight, unsavedInfoMap); err != nil {
		return nil, err
	}

	ui.flusher.Flush()
	ui.log.Info("rebuildAll data success", "latestHashHeight", fmt.Sprintf("[%v,%v]", latestHashHeight.Height, latestHashHeight.Hash))

	return &UnreceivedSnapshotMeta{
		HashHeight:     latestHashHeight,
		UnreceivedInfo: unsavedInfoMap,
	}, nil
}

func (ui *UnreceivedInfo) writeSnapshot(batch *leveldb.Batch, snapshotBlock *ledger.SnapshotBlock, signMetaMap signAddrUnreceivedMap) error {
	currentHashHeight := &ledger.HashHeight{Height: snapshotBlock.Height - 1, Hash: snapshotBlock.PrevHash}
	hashHeight := &ledger.HashHeight{Height: snapshotBlock.Height, Hash: snapshotBlock.Hash}
	// get latest available checkpoint
	nearestCheckpoint, err := ui.getNearestCheckpoint(currentHashHeight.Height)
	if err != nil {
		return err
	}
	latestCheckpoint, err := ui.prepareSnapshotData(currentHashHeight, nearestCheckpoint)
	if err != nil {
		return err
	}

	// aggregate data and write new checkpoint
	var unreceivedInfoMeta AddrUnreceivedMap
	if unreceivedInfoMeta = latestCheckpoint.UnreceivedInfo; unreceivedInfoMeta == nil {
		unreceivedInfoMeta = make(AddrUnreceivedMap)
	}

	snapshotCache, err := ui.snapshotCache.getSnapshotMetasAfterHeight(latestCheckpoint.HashHeight.Height)
	if err != nil {
		return err
	}

	if signInfoMap := aggregateCacheUnreceivedInfoMap(nil, snapshotCache, signMetaMap); signInfoMap != nil {
		if err := unreceivedInfoMeta.Add(signInfoMap); err != nil {
			return err
		}
	}
	if err := ui.writeCheckpoint(batch, hashHeight, unreceivedInfoMeta); err != nil {
		return err
	}

	// delete older checkpoint
	if err := ui.deleteCheckpointBefore(batch, currentHashHeight.Height-SnapshotBakDay*SnapshotHourHeight); err != nil {
		ui.log.Info(fmt.Sprintf("deleteCheckpointBefore failed, err: %v", err))
	}
	return nil
}

func (ui *UnreceivedInfo) writeSnapshotDirectly(hashHeight *ledger.HashHeight, unreceivedInfoMap map[types.Address]TokenBalanceInfoMap) error {
	batch := ui.store.NewBatch()
	if err := ui.writeCheckpoint(batch, hashHeight, unreceivedInfoMap); err != nil {
		return err
	}
	ui.store.WriteSnapshot(batch, nil)

	ui.currentHashHeight = hashHeight
	ui.snapshotCache.Push(hashHeight, nil)

	return nil
}

func (ui *UnreceivedInfo) getNearestCheckpoint(currentHeight uint64) (*UnreceivedSnapshotMeta, error) {
	iter := ui.store.NewIterator(&util.Range{Start: CreateUnreceivedInfoPrefixKey(1), Limit: CreateUnreceivedInfoPrefixKey(currentHeight)})
	defer iter.Release()

	iterExist := iter.Last()
	for iterExist {
		key := iter.Key()
		height := chain_utils.BytesToUint64(key[1:9])
		hash, err := types.BytesToHash(key[9:])
		if err != nil {
			return nil, err
		}
		existHash, err := ui.chain.GetSnapshotHashByHeight(height)
		if err != nil {
			return nil, err
		}
		if existHash != nil && hash == *existHash {
			meta := &UnreceivedSnapshotMeta{}
			if err := meta.Deserialize(iter.Value()); err != nil {
				return nil, err
			}
			return meta, nil
		}
		iterExist = iter.Prev()
	}
	if err := iter.Error(); err != nil && err != leveldb.ErrNotFound {
		return nil, err
	}
	return nil, nil
}

func (ui *UnreceivedInfo) prepareSnapshotData(currentHashHeight *ledger.HashHeight, nearestCheckpoint *UnreceivedSnapshotMeta) (*UnreceivedSnapshotMeta, error) {
	if currentHashHeight == nil || currentHashHeight.Height < 1 {
		return nil, errors.New("currentHeight invalid")
	}

	latestCheckpoint := nearestCheckpoint
	if nearestCheckpoint == nil || currentHashHeight.Height >= nearestCheckpoint.HashHeight.Height+(SnapshotDayHour*SnapshotBuildDayLimit) {
		var rebuildErr error
		if latestCheckpoint, rebuildErr = ui.rebuildAll(); rebuildErr != nil {
			return nil, rebuildErr
		}
	}
	if latestCheckpoint == nil {
		return nil, fmt.Errorf("failed to ensure a latestCheckpoint, currentHeight=%v", currentHashHeight.Height)
	}

	startHeight := latestCheckpoint.HashHeight.Height + 1

	if _, exist := ui.snapshotCache.ExistCachePoint(latestCheckpoint.HashHeight.Hash, latestCheckpoint.HashHeight.Height); !exist {
		ui.snapshotCache.Reset()
	}
	if _, tailMeta := ui.snapshotCache.Tail(); tailMeta != nil {
		cacheTailHeight := tailMeta.HashHeight.Height
		for {
			snapshotBlocks, err := ui.chain.GetSnapshotBlocksByHeight(cacheTailHeight, false, roundSize)
			if err != nil {
				return nil, err
			}
			for _, v := range snapshotBlocks { // height desc
				if index, exist := ui.snapshotCache.ExistCachePoint(v.Hash, v.Height); exist {
					ui.snapshotCache.RemoveAfter(index)
					startHeight = v.Height + 1
					break
				}
			}
			cacheTailHeight = cacheTailHeight - roundSize
		}
	}

	if err := ui.buildSnapshots(startHeight, currentHashHeight.Height); err != nil {
		return nil, err
	}

	return latestCheckpoint, nil
}

func (ui *UnreceivedInfo) writeCheckpoint(batch *leveldb.Batch, hashHeight *ledger.HashHeight, unreceivedInfoMap map[types.Address]TokenBalanceInfoMap) error {
	snapshotMeta := &UnreceivedSnapshotMeta{HashHeight: hashHeight, UnreceivedInfo: unreceivedInfoMap}
	value, err := snapshotMeta.Serialize()
	if err != nil {
		return err
	}
	batch.Put(CreateUnreceivedInfoKey(hashHeight), value)
	return nil
}

func (ui *UnreceivedInfo) readCheckpoint(height *ledger.HashHeight) (*UnreceivedSnapshotMeta, error) {
	value, err := ui.store.Get(CreateUnreceivedInfoKey(height))
	if err != nil {
		return nil, err
	}
	if len(value) <= 0 {
		return nil, nil
	}
	meta := &UnreceivedSnapshotMeta{}
	if err := meta.Deserialize(value); err != nil {
		return nil, err
	}
	return meta, nil
}

func (ui *UnreceivedInfo) deleteCheckpointBefore(batch *leveldb.Batch, deleteToHeight uint64) error {
	iter := ui.store.NewIterator(&util.Range{Start: CreateUnreceivedInfoPrefixKey(1), Limit: CreateUnreceivedInfoPrefixKey(deleteToHeight)})
	defer iter.Release()

	for iter.Next() {
		batch.Delete(iter.Key())
	}
	if err := iter.Error(); err != nil && err != leveldb.ErrNotFound {
		return err
	}
	return nil
}

func (ui *UnreceivedInfo) deleteCheckpointAfter(batch *leveldb.Batch, deleteToHeight uint64) error {
	iter := ui.store.NewIterator(&util.Range{Start: CreateUnreceivedInfoPrefixKey(deleteToHeight), Limit: CreateUnreceivedInfoPrefixKey(math.MaxUint64)})
	defer iter.Release()

	iterExist := iter.Last()
	for iterExist {
		batch.Delete(iter.Key())
		iterExist = iter.Prev()
	}
	if err := iter.Error(); err != nil && err != leveldb.ErrNotFound {
		return err
	}
	return nil
}

func (ui *UnreceivedInfo) getLatestSnapshotHashHeight() *ledger.HashHeight {
	if latestSb := ui.chain.GetLatestSnapshotBlock(); latestSb != nil {
		return &ledger.HashHeight{Height: latestSb.Height, Hash: latestSb.Hash}
	}
	return nil
}

func (ui *UnreceivedInfo) buildSnapshots(startHeight, endHeight uint64) error {
	h := startHeight
	for h < endHeight {
		targetH := h + roundSize
		if targetH > endHeight {
			targetH = endHeight
		}

		chunks, err := ui.chain.GetSubLedger(h, targetH)
		if err != nil {
			return err
		}

		ui.log.Info(fmt.Sprintf("buildSnapshots %d - %d", h+1, targetH))

		for _, chunk := range chunks {
			if chunk.SnapshotBlock != nil && chunk.SnapshotBlock.Height == h {
				continue
			}

			// write ab
			for _, ab := range chunk.AccountBlocks {
				batch := ui.store.NewBatch()
				if err := ui.InsertAccountBlock(batch, ab); err != nil {
					return err
				}
				ui.store.WriteAccountBlock(batch, ab)
			}

			// write sb
			batch := ui.store.NewBatch()
			if err := ui.InsertSnapshotBlock(batch, chunk.SnapshotBlock, chunk.AccountBlocks); err != nil {
				return fmt.Errorf("InsertSnapshotBlock fail, err:%v, sb[%v, %v,len=%v] ", err, chunk.SnapshotBlock.Height, chunk.SnapshotBlock.Hash, len(chunk.AccountBlocks))
			}
			ui.store.WriteSnapshot(batch, chunk.AccountBlocks)
		}

		ui.flusher.Flush()
		h = targetH
	}
	return nil
}

const SegReadRoutinesCount = 5

type AddHashMap map[types.Address][]types.Hash

func (ui *UnreceivedInfo) loadAllUnreceived(latestHashHeight *ledger.HashHeight) (unreceivedInfoMap map[types.Address]TokenBalanceInfoMap, resultErr error) {
	defer func() {
		if err := recover(); err != nil {
			ui.log.Error(fmt.Sprintf("loadAllUnreceived panic error %v", err), "latestHashHeight", fmt.Sprintf("[%v,%v]", latestHashHeight.Height, latestHashHeight.Hash))
			resultErr = errors.New("loadAllUnreceived failed")
		}
	}()

	chainUnreceivedMap, err := ui.chain.LoadAllOnRoad()
	if err != nil {
		return nil, err
	}
	if len(chainUnreceivedMap) <= 0 {
		return nil, nil
	}

	unreceivedInfoMap = make(map[types.Address]TokenBalanceInfoMap)

	totalNum := 0
	i := 0
	mapList := make([]AddHashMap, SegReadRoutinesCount)
	for addr, hashList := range chainUnreceivedMap {
		addMap := mapList[i%SegReadRoutinesCount]
		if addMap == nil {
			addMap = make(AddHashMap, 0)
			mapList[i%SegReadRoutinesCount] = addMap
		}
		addMap[addr] = hashList
		totalNum += len(hashList)
		i++
	}

	ui.log.Info(fmt.Sprintf("LoadAllOnRoad addrCount=%v hashCount=%v", len(chainUnreceivedMap), totalNum), "latestHashHeight", fmt.Sprintf("[%v,%v]", latestHashHeight.Height, latestHashHeight.Hash))

	var uMutex sync.Mutex
	var wg sync.WaitGroup
	for _, v := range mapList {
		if v == nil {
			continue
		}

		v1 := v
		wg.Add(1)
		go func(resultMap map[types.Address]TokenBalanceInfoMap, addHashListMap AddHashMap, wg *sync.WaitGroup, mu *sync.Mutex) {
			for addr, hashList := range addHashListMap {
				addrInfoMap := make(TokenBalanceInfoMap, 0)
				for _, v := range hashList {
					block, gErr := ui.chain.GetAccountBlockByHash(v)
					if gErr != nil || block == nil {
						panic(fmt.Sprintf("failed to get unreceived block by hash, err:%v, unreceived[%v,%v]", gErr, addr, v))
					}
					meta, ok := addrInfoMap[block.TokenId]
					if !ok || meta == nil {
						meta = &ledger.TokenBalanceInfo{
							TotalAmount: *big.NewInt(0),
							Number:      0,
						}
					}
					meta.TotalAmount.Add(&meta.TotalAmount, block.Amount)
					meta.Number++
					addrInfoMap[block.TokenId] = meta
				}
				mu.Lock()
				resultMap[addr] = addrInfoMap
				mu.Unlock()
			}
			ui.log.Info("rebuild seg map success", "latestHashHeight", fmt.Sprintf("[%v,%v]", latestHashHeight.Height, latestHashHeight.Hash))

			wg.Done()

		}(unreceivedInfoMap, v1, &wg, &uMutex)
	}
	wg.Wait()

	return unreceivedInfoMap, nil
}
