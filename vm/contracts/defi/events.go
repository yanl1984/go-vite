package defi

import (
	"github.com/golang/protobuf/proto"
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/vm/contracts/common"
	defiproto "github.com/vitelabs/go-vite/vm/contracts/defi/proto"
	"github.com/vitelabs/go-vite/vm_db"
)

const newLoanEventName = "newLoanEvent"
const loanUpdateEventName = "loanUpdateEvent"
const txEventName = "txEvent"

func AddNewLoanEvent(db vm_db.VmDb, ln *Loan) {
	event := &NewLoanEvent{}
	event.Loan = ln.Loan
	common.DoEmitEventLog(db, event)
}

type NewLoanEvent struct {
	defiproto.Loan
}

func (ln Loan) GetTopicId() types.Hash {
	return common.FromNameToHash(newLoanEventName)
}

func (ln Loan) ToDataBytes() []byte {
	data, _ := proto.Marshal(&ln.Loan)
	return data
}

func (ln Loan) FromBytes(data []byte) interface{} {
	event := NewLoanEvent{}
	if err := proto.Unmarshal(data, &event.Loan); err != nil {
		return nil
	} else {
		return event
	}
}
