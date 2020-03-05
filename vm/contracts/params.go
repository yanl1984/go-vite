package contracts

import (
	"math/big"
)

const (
	registrationNameLengthMax int = 40
)

var (
	// SbpStakeAmountMainnet defines SBP register stake amount
	SbpStakeAmountMainnet = new(big.Int).SetInt64(0)
)
