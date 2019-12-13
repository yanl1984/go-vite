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
const newSubscriptionEventName = "newSubscriptionEvent"
const subscriptionUpdateEventName = "subscriptionUpdateEvent"

func AddNewLoanEvent(db vm_db.VmDb, ln *Loan) {
	event := &NewLoanEvent{}
	event.Loan = ln.Loan
	common.DoEmitEventLog(db, event)
}

func AddLoanUpdateEvent(db vm_db.VmDb, ln *Loan) {
	event := &LoanUpdateEvent{}
	event.Id = ln.Id
	event.SubscribedShares = ln.SubscribedShares
	event.Invested = ln.Invested
	event.Status = ln.Status
	event.Timestamp = ln.Updated
	common.DoEmitEventLog(db, event)
}

func AddNewSubscriptionEvent(db vm_db.VmDb, sub *Subscription) {
	event := &NewSubscriptionEvent{}
	event.Subscription = sub.Subscription
	common.DoEmitEventLog(db, event)
}

func AddSubscriptionUpdateEvent(db vm_db.VmDb, sub *Subscription) {
	event := &SubscriptionUpdateEvent{}
	event.LoanId = sub.LoanId
	event.Address = sub.Address
	event.Shares = sub.Shares
	event.Status = sub.Status
	common.DoEmitEventLog(db, event)
}

type NewLoanEvent struct {
	defiproto.Loan
}

func (ln NewLoanEvent) GetTopicId() types.Hash {
	return common.FromNameToHash(newLoanEventName)
}

func (ln NewLoanEvent) ToDataBytes() []byte {
	data, _ := proto.Marshal(&ln.Loan)
	return data
}

func (ln NewLoanEvent) FromBytes(data []byte) interface{} {
	event := NewLoanEvent{}
	if err := proto.Unmarshal(data, &event.Loan); err != nil {
		return nil
	} else {
		return event
	}
}

type LoanUpdateEvent struct {
	defiproto.LoanUpdate
}

func (lu LoanUpdateEvent) GetTopicId() types.Hash {
	return common.FromNameToHash(loanUpdateEventName)
}

func (lu LoanUpdateEvent) ToDataBytes() []byte {
	data, _ := proto.Marshal(&lu.LoanUpdate)
	return data
}

func (lu LoanUpdateEvent) FromBytes(data []byte) interface{} {
	event := LoanUpdateEvent{}
	if err := proto.Unmarshal(data, &event.LoanUpdate); err != nil {
		return nil
	} else {
		return event
	}
}

type NewSubscriptionEvent struct {
	defiproto.Subscription
}

func (sb NewSubscriptionEvent) GetTopicId() types.Hash {
	return common.FromNameToHash(newSubscriptionEventName)
}

func (sb NewSubscriptionEvent) ToDataBytes() []byte {
	data, _ := proto.Marshal(&sb.Subscription)
	return data
}

func (sb NewSubscriptionEvent) FromBytes(data []byte) interface{} {
	event := NewSubscriptionEvent{}
	if err := proto.Unmarshal(data, &event.Subscription); err != nil {
		return nil
	} else {
		return event
	}
}

type SubscriptionUpdateEvent struct {
	defiproto.SubscriptionUpdate
}

func (su SubscriptionUpdateEvent) GetTopicId() types.Hash {
	return common.FromNameToHash(subscriptionUpdateEventName)
}

func (su SubscriptionUpdateEvent) ToDataBytes() []byte {
	data, _ := proto.Marshal(&su.SubscriptionUpdate)
	return data
}

func (su SubscriptionUpdateEvent) FromBytes(data []byte) interface{} {
	event := SubscriptionUpdateEvent{}
	if err := proto.Unmarshal(data, &event.SubscriptionUpdate); err != nil {
		return nil
	} else {
		return event
	}
}
