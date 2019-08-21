package ledger

import (
	"encoding/binary"
	"github.com/vitelabs/go-vite/common/types"
)

type ContractMeta struct {
	Gid types.Gid // belong to the consensus group id

	SendConfirmedTimes uint8

	CreateBlockHash types.Hash // hash of send create block for creating the contract
	QuotaRatio      uint8      // the ratio of quota cost for the send block

	SeedConfirmedTimes uint8

	IsDelete             uint8 // 0 is not delete, 1 is deleted
	DeleteSnapshotHeight uint64
}

const LengthBeforeSeedFork = types.GidSize + 1 + types.HashSize + 1
const LengthSupportDestroy = LengthBeforeSeedFork + 10

func (cm *ContractMeta) IsDeleted() bool {
	return cm.IsDelete > 0
}

//
//func (cm *ContractMeta) IsSnapshotDeleted(snapshotHeight uint64) bool {
//	return cm.IsDeleted() && cm.DeleteSnapshotHeight >= snapshotHeight
//}

func (cm *ContractMeta) Serialize() []byte {
	buf := make([]byte, 0, LengthSupportDestroy)

	buf = append(buf, cm.Gid.Bytes()...)
	buf = append(buf, cm.SendConfirmedTimes)
	buf = append(buf, cm.CreateBlockHash.Bytes()...)
	buf = append(buf, cm.QuotaRatio)

	buf = append(buf, cm.SeedConfirmedTimes)

	buf = append(buf, cm.IsDelete)

	deleteSnapshotHeightBytes := make([]byte, 8)
	binary.BigEndian.PutUint64(deleteSnapshotHeightBytes, cm.DeleteSnapshotHeight)
	buf = append(buf, deleteSnapshotHeightBytes...)

	return buf
}

func (cm *ContractMeta) Deserialize(buf []byte) error {

	gid, err := types.BytesToGid(buf[:types.GidSize])
	if err != nil {
		return err
	}

	CreateBlockHashBuf := buf[1+types.GidSize : 1+types.GidSize+types.HashSize]
	CreateBlockHash, err := types.BytesToHash(CreateBlockHashBuf)
	if err != nil {
		return err
	}

	cm.Gid = gid
	cm.SendConfirmedTimes = buf[types.GidSize]
	cm.CreateBlockHash = CreateBlockHash
	cm.QuotaRatio = buf[types.GidSize+1+types.HashSize]

	if len(buf) <= LengthBeforeSeedFork {
		cm.SeedConfirmedTimes = cm.SendConfirmedTimes
	} else {
		cm.SeedConfirmedTimes = buf[LengthBeforeSeedFork]
	}

	if len(buf) >= LengthSupportDestroy {
		cm.IsDelete = buf[LengthBeforeSeedFork+1]
		cm.DeleteSnapshotHeight = binary.BigEndian.Uint64(buf[LengthBeforeSeedFork+2 : LengthBeforeSeedFork+10])
	}

	return nil
}

func GetBuiltinContractMeta(addr types.Address) *ContractMeta {
	if types.IsBuiltinContractAddrInUseWithSendConfirm(addr) {
		return &ContractMeta{
			Gid:                types.DELEGATE_GID,
			SendConfirmedTimes: 1,
			QuotaRatio:         getBuiltinContractQuotaRatio(addr),
		}
	} else if types.IsBuiltinContractAddrInUse(addr) {
		return &ContractMeta{
			Gid:        types.DELEGATE_GID,
			QuotaRatio: getBuiltinContractQuotaRatio(addr),
		}
	}
	return nil
}
func getBuiltinContractQuotaRatio(addr types.Address) uint8 {
	// TODO use special quota ratio for dex contracts
	return 10
}
