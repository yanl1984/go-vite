package chain_plugins

import (
	"github.com/vitelabs/go-vite/chain/utils"
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/ledger"
)

const (
	OnRoadInfoKeyPrefix = byte(1)

	DiffTokenHash = byte(2)

	UnreceivedSnapshotInfoPrefix = byte(3)
)

func CreateOnRoadInfoKey(addr *types.Address, tId *types.TokenTypeId) []byte {
	key := make([]byte, 0, 1+types.AddressSize+types.TokenTypeIdSize)
	key = append(key, OnRoadInfoKeyPrefix)
	key = append(key, addr.Bytes()...)
	key = append(key, tId.Bytes()...)
	return key
}

func CreateOnRoadInfoPrefixKey(addr *types.Address) []byte {
	key := make([]byte, 0, 1+types.AddressSize)
	key = append(key, OnRoadInfoKeyPrefix)
	key = append(key, addr.Bytes()...)
	return key
}

func CreateUnreceivedInfoPrefixKey(height uint64) []byte {
	key := make([]byte, 0, 1+8+types.HashSize)
	key = append(key, UnreceivedSnapshotInfoPrefix)
	key = append(key, chain_utils.Uint64ToBytes(height)...)
	return key
}

// new plugin: key[3+height+hash]: value[serialize(addrMap)]
func CreateUnreceivedInfoKey(hashHeight *ledger.HashHeight) []byte {
	key := make([]byte, 0, 1+8+types.HashSize)
	key = append(key, UnreceivedSnapshotInfoPrefix)
	key = append(key, chain_utils.Uint64ToBytes(hashHeight.Height)...)
	key = append(key, hashHeight.Hash.Bytes()...)
	return key
}
