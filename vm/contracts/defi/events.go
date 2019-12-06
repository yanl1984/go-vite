package defi

import "github.com/vitelabs/go-vite/common/types"

const newLoanEventName = "newLoanEvent"
const loanUpdateEventName = "loanUpdateEvent"
const txEventName = "txEvent"


type DeFiEvent interface {
	GetTopicId() types.Hash
	toDataBytes() []byte
	FromBytes([]byte) interface{}
}

