package util

import (
	"github.com/vitelabs/go-vite/common/helper"
)

const (
	// CommonQuotaMultiplier defines base quota multiplier for all accounts
	CommonQuotaMultiplier   uint8  = 10
	quotaMultiplierDivision uint64 = 10
	// QuotaAccumulationBlockCount defines max quota accumulation count
	QuotaAccumulationBlockCount uint64 = 75
)

// MultipleCost multiply quota
func MultipleCost(cost uint64, quotaMultiplier uint8) (uint64, error) {
	if quotaMultiplier < CommonQuotaMultiplier {
		return 0, ErrInvalidQuotaMultiplier
	}
	if quotaMultiplier == CommonQuotaMultiplier {
		return cost, nil
	}
	ratioUint64 := uint64(quotaMultiplier)
	if cost > helper.MaxUint64/ratioUint64 {
		return 0, ErrGasUintOverflow
	}
	return cost * ratioUint64 / quotaMultiplierDivision, nil
}

// UseQuota check out of quota and return quota left
func UseQuota(quotaLeft, cost uint64) (uint64, error) {
	if quotaLeft < cost {
		return 0, ErrOutOfQuota
	}
	quotaLeft = quotaLeft - cost
	return quotaLeft, nil
}

// UseQuotaWithFlag check out of quota and return quota left
func UseQuotaWithFlag(quotaLeft, cost uint64, flag bool) (uint64, error) {
	if flag {
		return UseQuota(quotaLeft, cost)
	}
	return quotaLeft + cost, nil
}

// BlockGasCost calculate base quota cost of a block
func BlockGasCost(data []byte, baseGas uint64, snapshotCount uint8, quotaTable *QuotaTable) (uint64, error) {
	var gas uint64
	gas = baseGas
	gasData, err := DataQuotaCost(data, quotaTable)
	if err != nil || helper.MaxUint64-gas < gasData {
		return 0, ErrGasUintOverflow
	}
	gas = gas + gasData
	if snapshotCount == 0 {
		return gas, nil
	}
	confirmGas := uint64(snapshotCount) * quotaTable.SnapshotQuota
	if helper.MaxUint64-gas < confirmGas {
		return 0, ErrGasUintOverflow
	}
	return gas + confirmGas, nil
}

// DataQuotaCost calculate quota cost by request block data
func DataQuotaCost(data []byte, quotaTable *QuotaTable) (uint64, error) {
	var gas uint64
	if l := uint64(len(data)); l > 0 {
		if helper.MaxUint64/quotaTable.TxDataQuota < l {
			return 0, ErrGasUintOverflow
		}
		gas = l * quotaTable.TxDataQuota
	}
	return gas, nil
}

// RequestQuotaCost calculate quota cost by a request block
func RequestQuotaCost(data []byte, quotaTable *QuotaTable) (uint64, error) {
	dataCost, err := DataQuotaCost(data, quotaTable)
	if err != nil {
		return 0, err
	}
	totalCost, overflow := helper.SafeAdd(quotaTable.TxQuota, dataCost)
	if overflow {
		return 0, err
	}
	return totalCost, nil
}

// CalcQuotaUsed calculate stake quota and total quota used by a block
func CalcQuotaUsed(useQuota bool, quotaTotal, quotaLeft uint64, err error) (qStakeUsed uint64, qUsed uint64) {
	if !useQuota {
		return 0, 0
	}
	if err == ErrOutOfQuota {
		return 0, 0
	}
	qUsed = quotaTotal - quotaLeft
	return qUsed, qUsed
}

// QuotaTable is used to query quota used by op code and transactions
type QuotaTable struct {
	AddQuota            uint64
	MulQuota            uint64
	SubQuota            uint64
	DivQuota            uint64
	SDivQuota           uint64
	ModQuota            uint64
	SModQuota           uint64
	AddModQuota         uint64
	MulModQuota         uint64
	ExpQuota            uint64
	ExpByteQuota        uint64
	SignExtendQuota     uint64
	LtQuota             uint64
	GtQuota             uint64
	SltQuota            uint64
	SgtQuota            uint64
	EqQuota             uint64
	IsZeroQuota         uint64
	AndQuota            uint64
	OrQuota             uint64
	XorQuota            uint64
	NotQuota            uint64
	ByteQuota           uint64
	ShlQuota            uint64
	ShrQuota            uint64
	SarQuota            uint64
	Blake2bQuota        uint64
	Blake2bWordQuota    uint64
	AddressQuota        uint64
	BalanceQuota        uint64
	CallerQuota         uint64
	CallValueQuota      uint64
	CallDataLoadQuota   uint64
	CallDataSizeQuota   uint64
	CallDataCopyQuota   uint64
	MemCopyWordQuota    uint64
	CodeSizeQuota       uint64
	CodeCopyQuota       uint64
	ReturnDataSizeQuota uint64
	ReturnDataCopyQuota uint64
	TimestampQuota      uint64
	HeightQuota         uint64
	TokenIDQuota        uint64
	AccountHeightQuota  uint64
	PreviousHashQuota   uint64
	FromBlockHashQuota  uint64
	SeedQuota           uint64
	RandomQuota         uint64
	PopQuota            uint64
	MloadQuota          uint64
	MstoreQuota         uint64
	Mstore8Quota        uint64
	SloadQuota          uint64
	SstoreResetQuota    uint64
	SstoreInitQuota     uint64
	SstoreCleanQuota    uint64
	SstoreNoopQuota     uint64
	SstoreMemQuota      uint64
	JumpQuota           uint64
	JumpiQuota          uint64
	PcQuota             uint64
	MsizeQuota          uint64
	JumpdestQuota       uint64
	PushQuota           uint64
	DupQuota            uint64
	SwapQuota           uint64
	LogQuota            uint64
	LogTopicQuota       uint64
	LogDataQuota        uint64
	CallMinusQuota      uint64
	MemQuotaDivision    uint64
	SnapshotQuota       uint64
	CodeQuota           uint64
	MemQuota            uint64

	TxQuota               uint64
	TxDataQuota           uint64
	CreateTxRequestQuota  uint64
	CreateTxResponseQuota uint64

	RegisterQuota                    uint64
	UpdateBlockProducingAddressQuota uint64
	RevokeQuota                      uint64
}

// QuotaTableByHeight returns different quota table by hard fork version
func QuotaTableByHeight(sbHeight uint64) *QuotaTable {
	return &viteQuotaTable
}

var (
	viteQuotaTable = newViteQuotaTable()
)

func newViteQuotaTable() QuotaTable {
	return QuotaTable{
		AddQuota:            2,
		MulQuota:            2,
		SubQuota:            2,
		DivQuota:            3,
		SDivQuota:           5,
		ModQuota:            3,
		SModQuota:           4,
		AddModQuota:         4,
		MulModQuota:         5,
		ExpQuota:            10,
		ExpByteQuota:        50,
		SignExtendQuota:     2,
		LtQuota:             2,
		GtQuota:             2,
		SltQuota:            2,
		SgtQuota:            2,
		EqQuota:             2,
		IsZeroQuota:         1,
		AndQuota:            2,
		OrQuota:             2,
		XorQuota:            2,
		NotQuota:            2,
		ByteQuota:           2,
		ShlQuota:            2,
		ShrQuota:            2,
		SarQuota:            3,
		Blake2bQuota:        20,
		Blake2bWordQuota:    1,
		AddressQuota:        1,
		BalanceQuota:        150,
		CallerQuota:         1,
		CallValueQuota:      1,
		CallDataLoadQuota:   2,
		CallDataSizeQuota:   1,
		CallDataCopyQuota:   3,
		MemCopyWordQuota:    3,
		CodeSizeQuota:       1,
		CodeCopyQuota:       3,
		ReturnDataSizeQuota: 1,
		ReturnDataCopyQuota: 3,
		TimestampQuota:      1,
		HeightQuota:         1,
		TokenIDQuota:        1,
		AccountHeightQuota:  1,
		PreviousHashQuota:   1,
		FromBlockHashQuota:  1,
		SeedQuota:           200,
		RandomQuota:         250,
		PopQuota:            1,
		MloadQuota:          2,
		MstoreQuota:         1,
		Mstore8Quota:        1,
		SloadQuota:          150,
		SstoreResetQuota:    15000,
		SstoreInitQuota:     15000,
		SstoreCleanQuota:    0,
		SstoreNoopQuota:     200,
		SstoreMemQuota:      200,
		JumpQuota:           4,
		JumpiQuota:          4,
		PcQuota:             1,
		MsizeQuota:          1,
		JumpdestQuota:       1,
		PushQuota:           1,
		DupQuota:            1,
		SwapQuota:           2,
		LogQuota:            375,
		LogTopicQuota:       375,
		LogDataQuota:        12,
		CallMinusQuota:      13500,
		MemQuotaDivision:    1024,
		SnapshotQuota:       40,
		CodeQuota:           160,
		MemQuota:            1,

		TxQuota:               21000,
		TxDataQuota:           68,
		CreateTxRequestQuota:  31000,
		CreateTxResponseQuota: 31000,

		RegisterQuota:                    168000,
		UpdateBlockProducingAddressQuota: 168000,
		RevokeQuota:                      126000,
	}
}
