package vm

import (
	"github.com/vitelabs/go-vite/common/db/xleveldb/errors"
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/interfaces"
	"github.com/vitelabs/go-vite/ledger"
	"math/big"
)

type mockDB struct {
	currentAddr            *types.Address
	latestSnapshotBlock    *ledger.SnapshotBlock
	prevAccountBlock       *ledger.AccountBlock
	quotaInfo              []types.QuotaInfo
	pledgeBeneficialAmount *big.Int
	balanceMap             map[types.TokenTypeId]*big.Int
	balanceMapOrigin       map[types.TokenTypeId]*big.Int
	storageMap             map[string]string
	storageMapOrigin       map[string]string
	contractMetaMap        map[types.Address]*ledger.ContractMeta
	contractMetaMapOrigin  map[types.Address]*ledger.ContractMeta
	logList                []*ledger.VmLog
	code                   []byte
}

func NewMockDB(addr *types.Address,
	latestSnapshotBlock *ledger.SnapshotBlock,
	prevAccountBlock *ledger.AccountBlock,
	quotaInfo []types.QuotaInfo,
	pledgeBeneficialAmount *big.Int,
	balanceMap map[types.TokenTypeId]string,
	storage map[string]string,
	contractMetaMap map[types.Address]*ledger.ContractMeta,
	code []byte) (*mockDB, error) {
	db := &mockDB{currentAddr: addr,
		latestSnapshotBlock:    latestSnapshotBlock,
		prevAccountBlock:       prevAccountBlock,
		quotaInfo:              quotaInfo,
		pledgeBeneficialAmount: new(big.Int).Set(pledgeBeneficialAmount),
		logList:                make([]*ledger.VmLog, 0),
		balanceMap:             make(map[types.TokenTypeId]*big.Int),
		storageMap:             make(map[string]string),
		contractMetaMap:        make(map[types.Address]*ledger.ContractMeta),
		code:                   code,
	}
	balanceMapCopy := make(map[types.TokenTypeId]*big.Int)
	for tid, amount := range balanceMap {
		var ok bool
		balanceMapCopy[tid], ok = new(big.Int).SetString(amount, 16)
		if !ok {
			return nil, errors.New("invalid balance amount " + amount)
		}
	}
	db.balanceMapOrigin = balanceMapCopy

	storageMapCopy := make(map[string]string)
	for k, v := range storage {
		storageMapCopy[k] = v
	}
	db.storageMapOrigin = storageMapCopy

	contractMetaMapCopy := make(map[types.Address]*ledger.ContractMeta)
	for k, v := range contractMetaMap {
		contractMetaMapCopy[k] = v
	}
	db.contractMetaMapOrigin = contractMetaMapCopy
	return db, nil
}

func (db *mockDB) Address() *types.Address {
	return db.currentAddr
}
func (db *mockDB) LatestSnapshotBlock() (*ledger.SnapshotBlock, error) {
	if b := db.latestSnapshotBlock; b == nil {
		return nil, errors.New("latest snapshot block not exist")
	} else {
		return b, nil
	}
}
func (db *mockDB) PrevAccountBlock() (*ledger.AccountBlock, error) {
	return db.prevAccountBlock, nil
}
func (db *mockDB) GetLatestAccountBlock(addr types.Address) (*ledger.AccountBlock, error) {
	if addr != *db.currentAddr {
		return nil, errors.New("current account address not match")
	} else {
		return db.prevAccountBlock, nil
	}
}
func (db *mockDB) IsContractAccount() (bool, error) {
	if !types.IsContractAddr(*db.currentAddr) {
		return false, nil
	}
	if meta, err := db.GetContractMeta(); err != nil {
		return false, err
	} else {
		return meta != nil, nil
	}
}
func (db *mockDB) GetCallDepth(sendBlockHash *types.Hash) (uint16, error) {
	return 0, nil
}
func (db *mockDB) GetQuotaUsedList(addr types.Address) []types.QuotaInfo {
	if addr != *db.currentAddr {
		return nil
	} else {
		return db.quotaInfo
	}
}
func (db *mockDB) GetGlobalQuota() types.QuotaInfo {
	return types.QuotaInfo{}
}
func (db *mockDB) GetReceiptHash() *types.Hash {
	// TODO
	return &types.Hash{}
}
func (db *mockDB) Reset() {
	db.balanceMap = make(map[types.TokenTypeId]*big.Int)
	db.storageMap = make(map[string]string)
	db.contractMetaMap = make(map[types.Address]*ledger.ContractMeta)
	db.logList = make([]*ledger.VmLog, 0)
}
func (db *mockDB) Finish() {

}
func (db *mockDB) GetValue(key []byte) ([]byte, error) {
	// TODO
	return nil, nil
}
func (db *mockDB) GetOriginalValue(key []byte) ([]byte, error) {
	// TODO
	return nil, nil
}
func (db *mockDB) SetValue(key []byte, value []byte) error {
	// TODO
	return nil
}
func (db *mockDB) NewStorageIterator(prefix []byte) (interfaces.StorageIterator, error) {
	// TODO
	return nil, nil
}
func (db *mockDB) GetUnsavedStorage() [][2][]byte {
	return nil
}
func (db *mockDB) GetBalance(tokenTypeId *types.TokenTypeId) (*big.Int, error) {
	if balance, ok := db.balanceMap[*tokenTypeId]; ok {
		return new(big.Int).Set(balance), nil
	}
	if balance, ok := db.balanceMapOrigin[*tokenTypeId]; ok {
		return new(big.Int).Set(balance), nil
	}
	return big.NewInt(0), nil
}
func (db *mockDB) SetBalance(tokenTypeId *types.TokenTypeId, amount *big.Int) {
	db.balanceMap[*tokenTypeId] = amount
}
func (db *mockDB) GetBalanceMap() (map[types.TokenTypeId]*big.Int, error) {
	balanceMap := make(map[types.TokenTypeId]*big.Int)
	for tid, amount := range db.balanceMap {
		balanceMap[tid] = new(big.Int).Set(amount)
	}
	for tid, amount := range db.balanceMapOrigin {
		if _, ok := balanceMap[tid]; !ok {
			balanceMap[tid] = new(big.Int).Set(amount)
		}
	}
	return balanceMap, nil
}
func (db *mockDB) GetUnsavedBalanceMap() map[types.TokenTypeId]*big.Int {
	return nil
}
func (db *mockDB) AddLog(log *ledger.VmLog) {
	db.logList = append(db.logList, log)
}
func (db *mockDB) GetLogList() ledger.VmLogList {
	return db.logList
}
func (db *mockDB) GetHistoryLogList(logHash *types.Hash) (ledger.VmLogList, error) {
	return nil, nil
}
func (db *mockDB) GetLogListHash() *types.Hash {
	// TODO
	return nil
}
func (db *mockDB) GetUnconfirmedBlocks(address types.Address) []*ledger.AccountBlock {
	return nil
}
func (db *mockDB) GetGenesisSnapshotBlock() *ledger.SnapshotBlock {
	return nil
}
func (db *mockDB) GetConfirmSnapshotHeader(blockHash types.Hash) (*ledger.SnapshotBlock, error) {
	return nil, nil
}
func (db *mockDB) GetConfirmedTimes(blockHash types.Hash) (uint64, error) {
	return 0, nil
}
func (db *mockDB) SetContractMeta(toAddr types.Address, meta *ledger.ContractMeta) {
	db.contractMetaMap[toAddr] = meta
}
func (db *mockDB) GetContractMeta() (*ledger.ContractMeta, error) {
	if meta, ok := db.contractMetaMap[*db.currentAddr]; ok {
		return meta, nil
	}
	if meta, ok := db.contractMetaMapOrigin[*db.currentAddr]; ok {
		return meta, nil
	}
	return nil, nil
}
func (db *mockDB) GetContractMetaInSnapshot(contractAddress types.Address, snapshotBlock *ledger.SnapshotBlock) (meta *ledger.ContractMeta, err error) {
	if meta, ok := db.contractMetaMap[contractAddress]; ok {
		return meta, nil
	}
	if meta, ok := db.contractMetaMapOrigin[contractAddress]; ok {
		return meta, nil
	}
	return nil, nil
}
func (db *mockDB) SetContractCode(code []byte) {
	db.code = code
}
func (db *mockDB) GetContractCode() ([]byte, error) {
	return db.code, nil
}
func (db *mockDB) GetContractCodeBySnapshotBlock(addr *types.Address, snapshotBlock *ledger.SnapshotBlock) ([]byte, error) {
	return nil, nil
}
func (db *mockDB) GetUnsavedContractMeta() map[types.Address]*ledger.ContractMeta {
	return nil
}
func (db *mockDB) GetUnsavedContractCode() []byte {
	return nil
}
func (db *mockDB) GetPledgeBeneficialAmount(addr *types.Address) (*big.Int, error) {
	if *addr != *db.currentAddr {
		return nil, errors.New("current account address not match")
	} else {
		return db.pledgeBeneficialAmount, nil
	}
}
func (db *mockDB) DebugGetStorage() (map[string][]byte, error) {
	return nil, nil
}
func (db *mockDB) CanWrite() bool {
	return false
}
