package util

import "errors"

var (
	ErrInvalidMethodParam        = errors.New("invalid method param")
	ErrInvalidQuotaRatio         = errors.New("invalid quota ratio")
	ErrRewardIsNotDrained        = errors.New("reward is not drained")
	ErrInsufficientBalance       = errors.New("insufficient balance for transfer")
	ErrCalcPoWTwice              = errors.New("calc PoW twice referring to one snapshot block")
	ErrAbiMethodNotFound         = errors.New("abi: method not found")
	ErrInvalidConfirmTime        = errors.New("invalid confirm time")
	ErrInvalidSeedCount          = errors.New("invalid seed count")
	ErrAddressNotMatch           = errors.New("current address not match")
	ErrTransactionTypeNotSupport = errors.New("transaction type not supported")
	ErrVersionNotSupport         = errors.New("feature not supported in current snapshot height")
	ErrBlockTypeNotSupported     = errors.New("block type not supported")
	ErrDataNotExist              = errors.New("data not exist")
	ErrContractNotExists         = errors.New("contract not exists")
	ErrContractDeleted           = errors.New("contract deleted")
	ErrNoReliableStatus          = errors.New("no reliable status")
	ErrSendBlockListLimitReached = errors.New("too many send blocks")
	ErrNoTaskDue                 = errors.New("currently no task is due")

	ErrAddressCollision = errors.New("contract address collision")
	ErrIdCollision      = errors.New("id collision")
	ErrRewardNotDue     = errors.New("reward not due")

	ErrExecutionReverted = errors.New("execution reverted")
	ErrDepth             = errors.New("max call depth exceeded")

	ErrGasUintOverflow          = errors.New("gas uint64 overflow")
	ErrMemSizeOverflow          = errors.New("memory size uint64 overflow")
	ErrReturnDataOutOfBounds    = errors.New("vm: return data out of bounds")
	ErrBlockQuotaLimitReached   = errors.New("quota limit for block reached")
	ErrAccountQuotaLimitReached = errors.New("quota limit for account reached")
	ErrOutOfQuota               = errors.New("out of quota")
	ErrInvalidUnconfirmedQuota  = errors.New("calc quota failed, invalid unconfirmed quota")

	ErrStackLimitReached      = errors.New("stack limit reached")
	ErrStackUnderflow         = errors.New("stack underflow")
	ErrInvalidJumpDestination = errors.New("invalid jump destination")
	ErrInvalidOpCode          = errors.New("invalid opcode")

	ErrChainForked          = errors.New("chain forked")
	ErrContractCreationFail = errors.New("contract creation failed")

	ErrExecutionCanceled = errors.New("vm execution canceled")
)

// DealWithErr panics if err is not nil.
// Used when chain forked or db error.
func DealWithErr(v interface{}) {
	if v != nil {
		panic(v)
	}
}
