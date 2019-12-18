package defi

import (
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/rpcapi/api/dex"
	"github.com/vitelabs/go-vite/vm/contracts/defi"
)

type RpcLoan struct {
	Id               uint64 `json:"id"`
	Address          string `json:"address"`
	Token            string `json:"token"`
	ShareAmount      string `json:"shareAmount"`
	Shares           int32  `json:"shares"`
	Interest         string `json:"interest,omitempty"`
	DayRate          int32  `json:"dayRate"`
	SubscribeDays    int32  `json:"subscribeDays"`
	ExpireDays       int32  `json:"expireDays"`
	SubscribedShares int32  `json:"subscribedShares,omitempty"`
	ExpireHeight     uint64 `json:"expireHeight,omitempty"`
	Invested         string `json:"invested,omitempty"`
	Status           int32  `json:"status,omitempty"`
	Created          int64  `json:"created"`
	StartTime        int64  `json:"startTime,omitempty"`
	Updated          int64  `json:"updated,omitempty"`
}

func LoanToRpc(loan *defi.Loan) *RpcLoan {
	var rl *RpcLoan = nil
	if loan != nil {
		address, _ := types.BytesToAddress(loan.Address)
		token, _ := types.BytesToTokenTypeId(loan.Token)
		rl = &RpcLoan{
			Id:               loan.Id,
			Address:          address.String(),
			Token:            token.String(),
			ShareAmount:      dex.AmountBytesToString(loan.ShareAmount),
			Shares:           loan.Shares,
			Interest:         dex.AmountBytesToString(loan.Interest),
			DayRate:          loan.DayRate,
			SubscribeDays:    loan.SubscribeDays,
			ExpireDays:       loan.ExpireDays,
			SubscribedShares: loan.SubscribedShares,
			ExpireHeight:     loan.ExpireHeight,
			Invested:         dex.AmountBytesToString(loan.Invested),
			Status:           loan.Status,
			Created:          loan.Created,
			StartTime:        loan.StartTime,
			Updated:          loan.Updated,
		}
	}
	return rl
}

type RpcSubscription struct {
	LoanId      uint64 `json:"id"`
	Address     string `json:"address"`
	Token       string `json:"token"`
	ShareAmount string `json:"shareAmount"`
	Shares      int32  `json:"shares"`
	Interest    string `json:"interest,omitempty"`
	Status      int32  `json:"status,omitempty"`
	Created     int64  `json:"created"`
	Updated     int64  `json:"updated,omitempty"`
}

func SubscriptionToRpc(sub *defi.Subscription) *RpcSubscription {
	var rs *RpcSubscription = nil
	if sub != nil {
		address, _ := types.BytesToAddress(sub.Address)
		token, _ := types.BytesToTokenTypeId(sub.Token)
		rs = &RpcSubscription{
			LoanId:      sub.LoanId,
			Address:     address.String(),
			Token:       token.String(),
			ShareAmount: dex.AmountBytesToString(sub.ShareAmount),
			Shares:      sub.Shares,
			Interest:    dex.AmountBytesToString(sub.Interest),
			Status:      sub.Status,
			Created:     sub.Created,
			Updated:     sub.Updated,
		}
	}
	return rs
}
