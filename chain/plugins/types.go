package chain_plugins

import (
	"fmt"
	"github.com/golang/protobuf/proto"
	"github.com/vitelabs/go-vite/common/db/xleveldb/errors"
	"github.com/vitelabs/go-vite/common/helper"
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/ledger"
	"github.com/vitelabs/go-vite/vitepb"
	"math/big"
	"sync"
)

var (
	SnapshotDayHour        = uint64(24)
	SnapshotBakDay         = uint64(4)
	SnapshotHourHeight     = uint64(60 * 60)
	SnapshotCacheCapHeight = uint64(2 * SnapshotHourHeight)
	SnapshotBuildDayLimit  = uint64(30)
)

func encounterCheckPoint(height uint64) bool {
	return height%SnapshotHourHeight == 0
}

type signUnreceivedSnapshotMeta struct {
	HashHeight     *ledger.HashHeight
	UnreceivedInfo signAddrUnreceivedMap
}

type snapshotCache struct {
	cap          int
	currentIndex int
	signMetas    []*signUnreceivedSnapshotMeta

	heightIndexMap map[uint64]int

	mu sync.RWMutex
}

func newSnapshotCache() *snapshotCache {
	return &snapshotCache{
		cap:            0,
		currentIndex:   -1,
		signMetas:      make([]*signUnreceivedSnapshotMeta, int(SnapshotCacheCapHeight)),
		heightIndexMap: make(map[uint64]int),
	}
}

func (cache *snapshotCache) Len() int { return len(cache.signMetas) }

func (cache *snapshotCache) Reset() {
	cache.mu.Lock()
	defer cache.mu.Unlock()

	cache.cap = 0
	cache.currentIndex = -1
	cache.heightIndexMap = make(map[uint64]int)
	cache.signMetas = make([]*signUnreceivedSnapshotMeta, int(SnapshotCacheCapHeight))
}

func (cache *snapshotCache) nextIndex(i int) int {
	return (i + 1) % len(cache.signMetas)
}

func (cache *snapshotCache) lastIndex(i int) int {
	lastI := -1
	if i == 0 {
		lastI = len(cache.signMetas) - 1
	} else {
		lastI = i - 1
	}
	if meta := cache.signMetas[lastI]; meta != nil {
		return lastI
	}
	return -1
}

func (cache *snapshotCache) Tail() (int, *signUnreceivedSnapshotMeta) {
	cache.mu.RLock()
	defer cache.mu.RUnlock()

	if cache.currentIndex < 0 {
		return -1, nil
	}
	return cache.currentIndex, cache.signMetas[cache.currentIndex]
}

func (cache *snapshotCache) Prev(i int) *signUnreceivedSnapshotMeta {
	cache.mu.RLock()
	defer cache.mu.RUnlock()

	if lastI := cache.lastIndex(i); lastI > 0 {
		return cache.signMetas[cache.currentIndex]
	}
	return nil
}

func (cache *snapshotCache) Head() (int, *signUnreceivedSnapshotMeta) {
	cache.mu.RLock()
	defer cache.mu.RUnlock()

	headIndex := cache.currentIndex
	if cache.cap <= cache.currentIndex+1 {
		headIndex = cache.currentIndex + 1 - cache.cap
	} else {
		headIndex = cache.Len() - (cache.cap - (cache.currentIndex + 1))
	}
	return headIndex, cache.signMetas[headIndex]
}

func (cache *snapshotCache) Push(height *ledger.HashHeight, meta signAddrUnreceivedMap) int {
	cache.mu.Lock()
	defer cache.mu.Unlock()

	cache.currentIndex = cache.nextIndex(cache.currentIndex)
	cache.signMetas[cache.currentIndex] = &signUnreceivedSnapshotMeta{HashHeight: height, UnreceivedInfo: meta}
	cache.heightIndexMap[height.Height] = cache.currentIndex
	if cache.cap < len(cache.signMetas) {
		cache.cap++
	}
	return cache.currentIndex
}

func (cache *snapshotCache) Pop(height *ledger.HashHeight) int {
	cache.mu.Lock()
	defer cache.mu.Unlock()

	cache.signMetas[cache.currentIndex] = nil
	delete(cache.heightIndexMap, height.Height)
	if cache.cap > 0 {
		cache.cap--
	}
	if cache.currentIndex = cache.lastIndex(cache.currentIndex); cache.currentIndex < 0 {
		cache.currentIndex = -1
		cache.cap = 0
	}
	return cache.currentIndex
}

func (cache *snapshotCache) RemoveAfter(index int) {
	cache.mu.Lock()
	defer cache.mu.Unlock()

	if index == cache.currentIndex {
		return
	}

	meta := cache.signMetas[index]
	if meta == nil {
		return
	}

	for k, _ := range cache.heightIndexMap {
		if k > meta.HashHeight.Height {
			delete(cache.heightIndexMap, k)
		}
	}
	currentIndex := cache.currentIndex
	deleteCap := 0
	if index <= currentIndex {
		deleteCap = currentIndex - index
	} else {
		deleteCap = cache.cap - index + currentIndex
	}
	if cache.cap >= deleteCap {
		cache.cap = cache.cap - deleteCap
	}
	cache.currentIndex = index

}

func (cache *snapshotCache) ExistCachePoint(hash types.Hash, height uint64) (int, bool) { // todo
	cache.mu.RLock()
	defer cache.mu.RUnlock()

	if index, ok := cache.heightIndexMap[height]; ok {
		if meta := cache.signMetas[index]; meta != nil && meta.HashHeight.Equal(hash, height) {
			return index, true
		}
	}
	return -1, false
}

func (cache *snapshotCache) getSnapshotMetasAfterHeight(startHeight uint64) ([]signAddrUnreceivedMap, error) { // todo
	cacheSlice := make([]*signUnreceivedSnapshotMeta, 0)
	startIndex, startOk := cache.heightIndexMap[startHeight]
	if !startOk {
		return nil, errors.New("height isn't exits")
	}
	if startIndex <= cache.currentIndex {
		cacheSlice = append(cacheSlice, cache.signMetas[startIndex:cache.currentIndex]...)
	} else {
		cacheSlice = append(cacheSlice, cache.signMetas[startIndex:]...)
		cacheSlice = append(cacheSlice, cache.signMetas[0:cache.currentIndex]...)
	}
	result := make([]signAddrUnreceivedMap, 0, len(cacheSlice))
	for _, v := range cacheSlice {
		if v != nil {
			result = append(result, v.UnreceivedInfo)
		}
	}
	return result, nil
}

type UnreceivedSnapshotMeta struct {
	HashHeight     *ledger.HashHeight
	UnreceivedInfo AddrUnreceivedMap
}

func (snapshotMeta *UnreceivedSnapshotMeta) Serialize() ([]byte, error) {
	pb := &vitepb.UnreceivedSnapshotMeta{}
	pb.HashHeight = &vitepb.UnreceivedSnapshotMeta_SnapshotHashHeight{
		Hash:   snapshotMeta.HashHeight.Hash.Bytes(),
		Height: snapshotMeta.HashHeight.Height,
	}
	pb.UnreceivedInfos = make([]*vitepb.UnreceivedSnapshotMeta_AddrUnreceivedInfo, 0, len(snapshotMeta.UnreceivedInfo))
	for addr, tokenInfoMap := range snapshotMeta.UnreceivedInfo {
		pbAddrInfo := make([]*vitepb.UnreceivedSnapshotMeta_TokenBalanceInfo, 0, len(tokenInfoMap))
		for tokenId, balanceInfo := range tokenInfoMap {
			pbAddrInfo = append(pbAddrInfo, &vitepb.UnreceivedSnapshotMeta_TokenBalanceInfo{
				TokenId: tokenId.Bytes(),
				Number:  balanceInfo.Number,
				Amount:  balanceInfo.TotalAmount.Bytes(),
			})
		}
		pb.UnreceivedInfos = append(pb.UnreceivedInfos, &vitepb.UnreceivedSnapshotMeta_AddrUnreceivedInfo{
			Address:      addr.Bytes(),
			BalanceInfos: pbAddrInfo,
		})
	}
	buf, err := proto.Marshal(pb)
	if err != nil {
		return nil, err
	}
	return buf, nil
}

func (snapshotMeta *UnreceivedSnapshotMeta) Deserialize(buf []byte) error {
	pb := &vitepb.UnreceivedSnapshotMeta{}
	if unmarshalErr := proto.Unmarshal(buf, pb); unmarshalErr != nil {
		return unmarshalErr
	}
	var err error
	snapshotMeta.HashHeight = &ledger.HashHeight{Height: pb.HashHeight.Height}
	if snapshotMeta.HashHeight.Hash, err = types.BytesToHash(pb.HashHeight.Hash); err != nil {
		return err
	}
	snapshotMeta.UnreceivedInfo = make(map[types.Address]TokenBalanceInfoMap)
	for _, pbAddrInfo := range pb.UnreceivedInfos {
		if len(pbAddrInfo.BalanceInfos) <= 0 {
			continue
		}
		var addr types.Address
		if addr, err = types.BytesToAddress(pbAddrInfo.Address); err != nil {
			return err
		}
		tokenInfoMap := make(TokenBalanceInfoMap)
		for _, pbTokenInfo := range pbAddrInfo.BalanceInfos {
			var tokenId types.TokenTypeId
			if tokenId, err = types.BytesToTokenTypeId(pbTokenInfo.TokenId); err != nil {
				return err
			}
			amount := big.NewInt(0)
			if len(pbTokenInfo.Amount) > 0 {
				amount.SetBytes(pbTokenInfo.Amount)
			}
			tokenInfoMap[tokenId] = &ledger.TokenBalanceInfo{
				TotalAmount: *amount,
				Number:      pbTokenInfo.Number,
			}
		}
		snapshotMeta.UnreceivedInfo[addr] = tokenInfoMap
	}
	return nil
}

type signBalanceInfo struct {
	amount big.Int
	number big.Int
}

type signTokenBalanceInfoMap map[types.TokenTypeId]*signBalanceInfo

type TokenBalanceInfoMap map[types.TokenTypeId]*ledger.TokenBalanceInfo

func (m TokenBalanceInfoMap) Add(signMap signTokenBalanceInfoMap) error {
	var conflictErr string
	for tkId, signMeta := range signMap {
		tkInfo, ok := m[tkId]
		if !ok || tkInfo == nil {
			tkInfo = &ledger.TokenBalanceInfo{
				TotalAmount: *big.NewInt(0),
				Number:      0,
			}
			m[tkId] = tkInfo
		}
		num := new(big.Int).SetUint64(tkInfo.Number)
		diffNum := num.Add(num, &signMeta.number)
		diffAmount := tkInfo.TotalAmount.Add(&tkInfo.TotalAmount, &signMeta.amount)
		if diffAmount.Sign() < 0 || diffNum.Sign() < 0 || (diffAmount.Sign() > 0 && diffNum.Sign() == 0) {
			conflictErr += fmt.Sprintf("%v tkId=%v diffAmount=%v diffNum=%v", updateInfoErr, tkId, diffAmount, diffNum) + " | "
			continue
		}
		tkInfo.TotalAmount = *diffAmount
		tkInfo.Number = diffNum.Uint64()
	}
	if len(conflictErr) > 0 {
		return errors.New(conflictErr)
	}
	return nil
}

type signAddrUnreceivedMap map[types.Address]signTokenBalanceInfoMap

type AddrUnreceivedMap map[types.Address]TokenBalanceInfoMap

func (m AddrUnreceivedMap) Add(signMap signAddrUnreceivedMap) error {
	for addr, signMeta := range signMap {
		addrInfo, ok := m[addr]
		if !ok || addrInfo == nil {
			addrInfo = make(TokenBalanceInfoMap)
			m[addr] = addrInfo
		}
		if err := addrInfo.Add(signMeta); err != nil {
			return fmt.Errorf("AddrUnreceivedMap.Add conflict,addr=%s,err=%s", addr, err.Error())
		}
	}
	return nil
}

func aggregateBlocks(chain Chain, blocks []*ledger.AccountBlock) (signAddrUnreceivedMap, error) {
	if len(blocks) <= 0 {
		return nil, nil
	}

	cutMap := make(map[types.Hash]*ledger.AccountBlock)
	for _, block := range blocks {
		if block.IsSendBlock() {
			v, ok := cutMap[block.Hash]
			if ok && v != nil && v.IsReceiveBlock() {
				delete(cutMap, block.Hash)
			} else {
				cutMap[block.Hash] = block
			}
			continue
		}

		if chain.IsGenesisAccountBlock(block.Hash) {
			continue
		}

		// receive block
		v, ok := cutMap[block.FromBlockHash]
		if ok && v != nil && v.IsSendBlock() {
			delete(cutMap, block.FromBlockHash)
		} else {
			cutMap[block.FromBlockHash] = block
		}

		// sendBlockList
		if !types.IsContractAddr(block.AccountAddress) || len(block.SendBlockList) <= 0 {
			continue
		}
		for _, subSend := range block.SendBlockList {
			v, ok := cutMap[subSend.Hash]
			if ok && v != nil && v.IsReceiveBlock() {
				delete(cutMap, subSend.Hash)
			} else {
				cutMap[subSend.Hash] = subSend
			}
		}
	}

	signUnreceivedAddrMap := make(signAddrUnreceivedMap)
	for _, block := range cutMap {
		var addr types.Address
		var tk types.TokenTypeId
		amount := big.NewInt(0)
		if block.IsSendBlock() {
			addr = block.ToAddress
			tk = block.TokenId
			if block.Amount != nil {
				amount.Add(amount, block.Amount)
			}
		} else {
			addr = block.AccountAddress
			fromBlock, gErr := chain.GetAccountBlockByHash(block.FromBlockHash)
			if gErr != nil || fromBlock == nil {
				return nil, fmt.Errorf("failed to get unreceived block by hash, err:%v, unreceived[%v,%v], receivedHash[%v]", gErr, block.AccountAddress, block.FromBlockHash, block.Hash)
			}
			tk = fromBlock.TokenId
			if fromBlock.Amount != nil {
				amount.Sub(amount, fromBlock.Amount)
			}
		}
		addrInfo, ok := signUnreceivedAddrMap[addr]
		if !ok || addrInfo == nil {
			addrInfo = make(signTokenBalanceInfoMap)
			signUnreceivedAddrMap[addr] = addrInfo
		}
		tkInfo, ok := addrInfo[tk]
		if !ok || tkInfo == nil {
			tkInfo = &signBalanceInfo{
				amount: *big.NewInt(0),
				number: *big.NewInt(0),
			}
			addrInfo[tk] = tkInfo
		}
		tkInfo.amount.Add(&tkInfo.amount, amount)
		tkInfo.number.Add(&tkInfo.number, helper.Big1)
	}

	return signUnreceivedAddrMap, nil
}

func aggregateCacheUnreceivedInfoMap(assignAddrMap map[types.Address]bool, snapshotUnreceivedInfoList []signAddrUnreceivedMap, signSnapshotMetaList ...signAddrUnreceivedMap) signAddrUnreceivedMap {
	addrSignUnreceivedInfo := make(signAddrUnreceivedMap)
	assignFlag := false
	if len(assignAddrMap) > 0 {
		assignFlag = true
	}

	for _, snapshotMeta := range snapshotUnreceivedInfoList {
		for addr, tkInfoMap := range snapshotMeta {
			if tkInfoMap == nil {
				continue
			}
			if assignFlag {
				if _, ok := assignAddrMap[addr]; !ok {
					continue
				}
			}

			addrInfo, ok := addrSignUnreceivedInfo[addr]
			if !ok || addrInfo == nil {
				addrInfo = make(signTokenBalanceInfoMap)
				addrSignUnreceivedInfo[addr] = addrInfo
			}

			for tk, signBalance := range tkInfoMap {
				if signBalance == nil {
					continue
				}
				tkInfo, ok := addrInfo[tk]
				if !ok || tkInfo == nil {
					tkInfo = &signBalanceInfo{
						amount: *big.NewInt(0),
						number: *big.NewInt(0),
					}
					addrInfo[tk] = tkInfo
				}
				tkInfo.amount.Add(&tkInfo.amount, &signBalance.amount)
				tkInfo.number.Add(&tkInfo.number, &signBalance.number)
			}
		}
	}
	for _, v := range signSnapshotMetaList {
		for addr, tkInfoMap := range v {
			if tkInfoMap == nil {
				continue
			}
			if assignFlag {
				if _, ok := assignAddrMap[addr]; !ok {
					continue
				}
			}

			addrInfo, ok := addrSignUnreceivedInfo[addr]
			if !ok || addrInfo == nil {
				addrInfo = make(signTokenBalanceInfoMap)
				addrSignUnreceivedInfo[addr] = addrInfo
			}

			for tk, signBalance := range tkInfoMap {
				if signBalance == nil {
					continue
				}
				tkInfo, ok := addrInfo[tk]
				if !ok || tkInfo == nil {
					tkInfo = &signBalanceInfo{
						amount: *big.NewInt(0),
						number: *big.NewInt(0),
					}
					addrInfo[tk] = tkInfo
				}
				tkInfo.amount.Add(&tkInfo.amount, &signBalance.amount)
				tkInfo.number.Add(&tkInfo.number, &signBalance.number)
			}
		}
	}

	return addrSignUnreceivedInfo
}
